package magmasc

import (
	"encoding/json"
	"sync"

	chain "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

type (
	// consumerPools represents abstract layer of registered payoff pools
	// that stores list of token pools to pay to Provider.
	consumerPools struct {
		UID   datastore.Key                   `json:"uid"`   // consumer uid
		Pools map[datastore.Key]datastore.Key `json:"pools"` // token pools list

		mux sync.RWMutex
	}
)

var (
	// Make sure consumerPools implements Serializable interface.
	_ util.Serializable = (*consumerPools)(nil)
)

// Decode implements Serializable interface.
func (m *consumerPools) Decode(blob []byte) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	pools := consumerPools{}
	if err := json.Unmarshal(blob, &pools); err != nil {
		return errWrap(errCodeDecode, errTextDecode, err)
	}

	m.UID = pools.UID
	m.Pools = pools.Pools

	return nil
}

// Encode implements Serializable interface.
func (m *consumerPools) Encode() []byte {
	m.mux.RLock()
	defer m.mux.RUnlock()

	buff, _ := json.Marshal(m)

	return buff
}

// checkConditions checks conditions.
func (m *consumerPools) checkConditions(ackn *Acknowledgment, sci chain.StateContextI) error {
	volume := ackn.ProviderTerms.GetVolume()
	if volume < 0 {
		return errWrap(errCodeCheckCondition, errTextUnexpected, errNegativeTxnValue)
	}

	clientBalance, err := sci.GetClientBalance(ackn.ConsumerID)
	if err != nil && !errIs(err, util.ErrValueNotPresent) {
		return errWrap(errCodeCheckCondition, errTextUnexpected, err)
	}
	if clientBalance < state.Balance(volume) {
		return errWrap(errCodeCheckCondition, errTextUnexpected, errInsufficientFunds)
	}

	return nil
}

// tokenPollBalance returns token poll balance.
func (m *consumerPools) tokenPollBalance(ackn *Acknowledgment, sci chain.StateContextI) (state.Balance, error) {
	pool, err := m.tokenPollFetch(ackn, sci)
	if err != nil {
		return 0, errNew(errCodeTokenPoolBalance, err.Error())
	}

	return pool.Balance, nil
}

// tokenPollCreate creates token poll.
func (m *consumerPools) tokenPollCreate(id datastore.Key, ackn *Acknowledgment, sci chain.StateContextI) (string, error) {
	if err := m.checkConditions(ackn, sci); err != nil {
		return "", errNew(errCodeTokenPoolCreate, err.Error())
	}

	pool := tokenPool{
		ClientID:   ackn.ConsumerID,
		DelegateID: ackn.ProviderID,
	}

	resp, err := pool.create(id, ackn, sci)
	if err != nil {
		return "", errWrap(errCodeTokenPoolCreate, "dig token pool failed", err)
	}

	uid := pool.uid(m.UID)
	if _, err = sci.InsertTrieNode(uid, &pool); err != nil {
		return "", errWrap(errCodeTokenPoolCreate, "insert token pool failed", err)
	}

	m.mux.Lock()
	m.Pools[ackn.SessionID] = uid
	m.mux.Unlock()

	return resp, nil
}

// tokenPollFetch fetches token poll.
func (m *consumerPools) tokenPollFetch(ackn *Acknowledgment, sci chain.StateContextI) (*tokenPool, error) {
	m.mux.RLock()
	defer m.mux.RUnlock()

	uid, ok := m.Pools[ackn.SessionID]
	if !ok {
		return nil, errNew(errCodeFetchData, "not found token pool: "+ackn.SessionID)
	}

	data, err := sci.GetTrieNode(uid)
	if err != nil || data == nil {
		return nil, errWrap(errCodeFetchData, "fetch token pool failed", err)
	}

	pool := tokenPool{}
	if err = json.Unmarshal(data.Encode(), &pool); err != nil {
		return nil, errWrap(errCodeFetchData, "decode token pool failed", err)
	}
	if pool.ClientID != ackn.ConsumerID {
		return nil, errNew(errCodeFetchData, "cannot fetch not owned token pool: "+ackn.ConsumerID)
	}
	if pool.DelegateID != ackn.ProviderID {
		return nil, errNew(errCodeFetchData, "cannot fetch not delegated token pool: "+ackn.ProviderID)
	}

	return &pool, nil
}

// tokenPollRefund refunds remaining balance of token poll and remove it.
func (m *consumerPools) tokenPollRefund(ackn *Acknowledgment, sci chain.StateContextI) (string, error) {
	pool, err := m.tokenPollFetch(ackn, sci)
	if err != nil {
		return "", errNew(errCodeTokenPoolRefund, err.Error())
	}

	resp, err := pool.transfer(pool.DelegateID, pool.ClientID, sci)
	if err != nil {
		return "", errWrap(errCodeTokenPoolRefund, "transfer token pool failed", err)
	}
	if _, err = sci.DeleteTrieNode(pool.uid(m.UID)); err != nil {
		return "", errWrap(errCodeTokenPoolRefund, "delete token pool failed", err)
	}

	m.mux.Lock()
	delete(m.Pools, ackn.SessionID)
	m.mux.Unlock()

	return resp, nil
}

// tokenPollSpend spends token poll to delegated wallet.
func (m *consumerPools) tokenPollSpend(ackn *Acknowledgment, amount state.Balance, sci chain.StateContextI) (string, error) {
	pool, err := m.tokenPollFetch(ackn, sci)
	if err != nil {
		return "", errNew(errCodeTokenPoolSpend, err.Error())
	}

	if pool.Balance <= amount { // pool should be staked
		resp, err := pool.transfer(pool.ClientID, pool.DelegateID, sci)
		if err != nil {
			return "", errWrap(errCodeTokenPoolSpend, "transfer token pool failed", err)
		}
		if _, err = sci.DeleteTrieNode(pool.uid(m.UID)); err != nil {
			return "", errWrap(errCodeTokenPoolSpend, "delete token pool failed", err)
		}

		m.mux.Lock()
		delete(m.Pools, ackn.SessionID)
		m.mux.Unlock()

		return resp, nil
	}

	transfer, resp, err := pool.DrainPool(pool.ClientID, pool.DelegateID, amount, nil)
	if err != nil {
		return "", errWrap(errCodeTokenPoolSpend, "spend token pool failed", err)
	}
	if err = sci.AddTransfer(transfer); err != nil {
		return "", errWrap(errCodeTokenPoolSpend, "transfer token pool failed", err)
	}
	if _, err = sci.InsertTrieNode(pool.uid(m.UID), pool); err != nil {
		return "", errWrap(errCodeTokenPoolSpend, "update token pool failed", err)
	}

	return resp, nil
}
