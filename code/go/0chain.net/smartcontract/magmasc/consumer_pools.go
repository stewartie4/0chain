package magmasc

import (
	"encoding/json"

	chain "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	tx "0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

type (
	// consumerPools represents abstract layer of registered payoff pools
	// that stores list of token pools to pay to Provider.
	consumerPools struct {
		UID   datastore.Key                `json:"uid"`   // consumer uid
		Pools map[datastore.Key]*tokenPool `json:"pools"` // token pools list
	}
)

var (
	// Make sure consumerPools implements Serializable interface.
	_ util.Serializable = (*consumerPools)(nil)
)

// Decode implements Serializable interface.
func (m *consumerPools) Decode(blob []byte) error {
	var pools consumerPools
	if err := json.Unmarshal(blob, &pools); err != nil {
		return errWrap(errCodeDecode, errTextDecode, err)
	}

	*m = pools

	return nil
}

// Encode implements Serializable interface.
func (m *consumerPools) Encode() []byte {
	buff, _ := json.Marshal(m)
	return buff
}

// checkConditions checks conditions.
func (m *consumerPools) checkConditions(txn *tx.Transaction, sci chain.StateContextI) error {
	if txn.Value < 0 {
		return errNew(errCodeCheckCondition, "negative transaction value")
	}

	clientBalance, err := sci.GetClientBalance(txn.ClientID)
	if err != nil && !errIs(err, util.ErrValueNotPresent) {
		return errWrap(errCodeCheckCondition, errTextUnexpected, err)
	}

	if clientBalance < state.Balance(txn.Value) {
		return errNew(errCodeCheckCondition, errTextInsufficientFunds)
	}

	return nil
}

// tokenPollBalance returns token poll balance.
func (m *consumerPools) tokenPollBalance(id datastore.Key, txn *tx.Transaction) (int64, error) {
	pool, err := m.tokenPollFetch(id, txn)
	if err != nil {
		return 0, errWrap(errCodeFetchData, "fetch token pool failed", err)
	}

	return int64(pool.Balance), nil
}

// tokenPollCreate creates token poll.
func (m *consumerPools) tokenPollCreate(id datastore.Key, txn *tx.Transaction, sci chain.StateContextI) (string, error) {
	if err := m.checkConditions(txn, sci); err != nil {
		return "", errNew(errCodeTokenPoolCreate, err.Error())
	}

	pool := tokenPool{
		ClientID:   txn.ClientID,
		DelegateID: txn.ToClientID,
	}

	transfer, resp, err := pool.DigPool(id, txn)
	if err != nil {
		return "", errWrap(errCodeTokenPoolCreate, "dig token pool failed", err)
	}
	if err = sci.AddTransfer(transfer); err != nil {
		return "", errWrap(errCodeTokenPoolCreate, "transfer token pool failed", err)
	}
	if _, err = sci.InsertTrieNode(pool.uid(m.UID), &pool); err != nil {
		return "", errWrap(errCodeTokenPoolCreate, "insert token pool failed", err)
	}

	m.Pools[pool.ID] = &pool

	return resp, nil
}

// tokenPollEmpty makes a transfer for token poll and remove it.
func (m *consumerPools) tokenPollEmpty(pool *tokenPool, fromID, toID datastore.Key, sci chain.StateContextI) (string, error) {
	transfer, resp, err := pool.EmptyPool(fromID, toID, nil)
	if err != nil {
		return "", errWrap(errCodeTokenPoolEmpty, "empty token pool failed", err)
	}
	if err = sci.AddTransfer(transfer); err != nil {
		return "", errWrap(errCodeTokenPoolEmpty, "transfer token pool failed", err)
	}
	if _, err = sci.DeleteTrieNode(pool.uid(m.UID)); err != nil {
		return "", errWrap(errCodeTokenPoolEmpty, "delete token pool failed", err)
	}

	delete(m.Pools, pool.ID)

	return resp, nil
}

// tokenPollFetch fetches token poll.
func (m *consumerPools) tokenPollFetch(id datastore.Key, txn *tx.Transaction) (*tokenPool, error) {
	pool, ok := m.Pools[id]
	if !ok {
		return nil, errNew(errCodeFetchData, "not found token pool: "+txn.Hash)
	}
	if pool.ClientID != txn.ClientID {
		return nil, errNew(errCodeFetchData, "cannot fetch not owned token pool: "+txn.ClientID)
	}
	if pool.DelegateID != txn.ToClientID {
		return nil, errNew(errCodeFetchData, "cannot fetch not delegated token pool: "+txn.ToClientID)
	}

	return pool, nil
}

// tokenPollRefund refunds remaining balance of token poll and remove it.
func (m *consumerPools) tokenPollRefund(id datastore.Key, txn *tx.Transaction, sci chain.StateContextI) (string, error) {
	pool, err := m.tokenPollFetch(id, txn)
	if err != nil {
		return "", errWrap(errCodeTokenPoolRefund, "fetch token pool failed", err)
	}

	resp, err := m.tokenPollEmpty(pool, pool.DelegateID, pool.ClientID, sci)
	if err != nil {
		return "", errWrap(errCodeTokenPoolRefund, "refund token pool failed", err)
	}

	return resp, nil
}

// tokenPollSpend spends token poll to delegated wallet.
func (m *consumerPools) tokenPollSpend(id datastore.Key, txn *tx.Transaction, sci chain.StateContextI) (string, error) {
	pool, err := m.tokenPollFetch(id, txn)
	if err != nil {
		return "", errWrap(errCodeTokenPoolSpend, "fetch token pool failed", err)
	}

	amount := state.Balance(txn.Value)
	if pool.Balance <= amount { // pool should be staked
		return m.tokenPollEmpty(pool, pool.ClientID, pool.DelegateID, sci)
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
