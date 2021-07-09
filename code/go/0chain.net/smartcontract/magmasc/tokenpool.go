package magmasc

import (
	"encoding/json"

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

		PayerID datastore.Key `json:"payer_id"`
		PayeeID datastore.Key `json:"payee_id"`
	}
)

var (
	// Make sure tokenPool implements Serializable interface.
	_ util.Serializable = (*tokenPool)(nil)
)

// Decode implements util.Serializable interface.
func (m *tokenPool) Decode(blob []byte) error {
	var pool tokenPool
	if err := json.Unmarshal(blob, &pool); err != nil {
		return errDecodeData.WrapErr(err)
	}

	m.ID = pool.ID
	m.Balance = pool.Balance
	m.PayerID = pool.PayerID
	m.PayeeID = pool.PayeeID

	return nil
}

// Encode implements util.Serializable interface.
func (m *tokenPool) Encode() []byte {
	blob, _ := json.Marshal(m)
	return blob
}

// create creates token poll by given acknowledgment.
func (m *tokenPool) create(txn *tx.Transaction, ackn *Acknowledgment, sci chain.StateContextI) (string, error) {
	clientBalance, err := sci.GetClientBalance(ackn.ConsumerID)
	if err != nil {
		return "", errWrap(errCodeTokenPoolCreate, errTextUnexpected, errInsufficientFunds)
	}

	poolBalance := ackn.ProviderTerms.GetAmount()
	if clientBalance < poolBalance {
		return "", errWrap(errCodeTokenPoolCreate, errTextUnexpected, errInsufficientFunds)
	}

	m.ID = ackn.SessionID
	m.Balance = poolBalance
	m.PayerID = ackn.ConsumerID
	m.PayeeID = ackn.ProviderID

	transfer := state.NewTransfer(m.PayerID, txn.ToClientID, m.Balance)
	if err = sci.AddTransfer(transfer); err != nil {
		return "", errWrap(errCodeTokenPoolCreate, "transfer token pool failed", err)
	}

	resp := &tokenpool.TokenPoolTransferResponse{
		TxnHash:    txn.Hash,
		ToPool:     m.ID,
		Value:      m.Balance,
		FromClient: m.PayerID,
		ToClient:   txn.ToClientID, // delegate transfer to smart contract address
	}

	return string(resp.Encode()), nil
}

// spend spends token pool by given amount.
func (m *tokenPool) spend(txn *tx.Transaction, amount state.Balance, sci chain.StateContextI) (string, error) {
	transfer, resp, err := m.DrainPool(txn.ToClientID, m.PayeeID, amount, nil)
	if err != nil {
		return "", errWrap(errCodeTokenPoolSpend, "spend token pool failed", err)
	}
	if err = sci.AddTransfer(transfer); err != nil {
		return "", errWrap(errCodeTokenPoolSpend, "transfer token pool failed", err)
	}

	return resp, nil
}

// transfer makes a transfer for token poll and remove it.
func (m *tokenPool) transfer(payerID, payeeID datastore.Key, sci chain.StateContextI) (string, error) {
	transfer, resp, err := m.EmptyPool(payerID, payeeID, nil)
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
