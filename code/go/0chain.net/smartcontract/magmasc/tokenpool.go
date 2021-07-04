package magmasc

import (
	chain "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	tx "0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

type (
	// tokenPool represents token pool wrapper implementation.
	tokenPool struct {
		tokenpool.ZcnPool // embedded token pool

		ClientID   datastore.Key `json:"client_id"`
		DelegateID datastore.Key `json:"delegate_id"`
	}
)

var (
	// Make sure tokenPool implements Serializable interface.
	_ util.Serializable = (*tokenPool)(nil)
)

// create creates token poll by given acknowledgment.
func (m *tokenPool) create(txn *tx.Transaction, ackn *Acknowledgment, sci chain.StateContextI) (string, error) {
	m.Balance = state.Balance(ackn.ProviderTerms.GetVolume() * ackn.ProviderTerms.Price)
	if m.Balance < 0 {
		return "", errWrap(errCodeTokenPoolCreate, errTextUnexpected, errNegativeValue)
	}

	clientBalance, err := sci.GetClientBalance(ackn.ConsumerID)
	if err != nil {
		return "", errWrap(errCodeTokenPoolCreate, errTextUnexpected, err)
	}
	if clientBalance < m.Balance {
		return "", errWrap(errCodeTokenPoolCreate, errTextUnexpected, errInsufficientFunds)
	}

	m.ID = ackn.SessionID
	m.ClientID = ackn.ConsumerID
	m.DelegateID = ackn.ProviderID

	transfer := state.NewTransfer(m.ClientID, m.DelegateID, m.Balance)
	if err = sci.AddTransfer(transfer); err != nil {
		return "", errWrap(errCodeTokenPoolCreate, "transfer token pool failed", err)
	}

	resp := &tokenpool.TokenPoolTransferResponse{
		TxnHash:    txn.Hash,
		ToPool:     m.ID,
		Value:      m.Balance,
		FromClient: m.ClientID,
		ToClient:   m.DelegateID,
	}

	return string(resp.Encode()), nil
}

// spend spends token pool by given amount.
func (m *tokenPool) spend(amount state.Balance, sci chain.StateContextI) (string, error) {
	transfer, resp, err := m.DrainPool(m.ClientID, m.DelegateID, amount, nil)
	if err != nil {
		return "", errWrap(errCodeTokenPoolSpend, "spend token pool failed", err)
	}
	if err = sci.AddTransfer(transfer); err != nil {
		return "", errWrap(errCodeTokenPoolSpend, "transfer token pool failed", err)
	}

	return resp, nil
}

// transfer makes a transfer for token poll and remove it.
func (m *tokenPool) transfer(fromID, toID datastore.Key, sci chain.StateContextI) (string, error) {
	transfer, resp, err := m.EmptyPool(fromID, toID, nil)
	if err != nil {
		return "", errWrap(errCodeTokenPoolTransfer, "stake token pool failed", err)
	}
	if err = sci.AddTransfer(transfer); err != nil {
		return "", errWrap(errCodeTokenPoolTransfer, "transfer token pool failed", err)
	}

	return resp, nil
}

// uid returns uniq id used to saving tokenPool into chain state.
func (m *tokenPool) uid(parentUID datastore.Key) datastore.Key {
	return parentUID + ":tokenpool:" + m.ID
}
