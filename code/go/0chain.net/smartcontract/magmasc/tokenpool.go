package magmasc

import (
	chain "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
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

// create creates a token poll by given acknowledgment.
func (m *tokenPool) create(id datastore.Key, ackn *Acknowledgment, sci chain.StateContextI) (string, error) {
	amount := ackn.ProviderTerms.GetVolume() * ackn.ProviderTerms.Price

	m.ID = id
	m.Balance = state.Balance(amount)

	transfer := state.NewTransfer(ackn.ConsumerID, ackn.ProviderID, m.Balance)
	if err := sci.AddTransfer(transfer); err != nil {
		return "", errWrap(errCodeTokenPoolCreate, "transfer token pool failed", err)
	}

	response := &tokenpool.TokenPoolTransferResponse{
		TxnHash:    id,
		FromClient: ackn.ConsumerID,
		ToClient:   ackn.ProviderID,
		ToPool:     m.ID,
		Value:      m.Balance,
	}

	return string(response.Encode()), nil
}

// transfer makes a transfer for token poll and remove it.
func (m *tokenPool) transfer(fromID, toID datastore.Key, sci chain.StateContextI) (string, error) {
	transfer, resp, err := m.EmptyPool(fromID, toID, nil)
	if err != nil {
		return "", errWrap(errCodeTokenPoolTransfer, "empty token pool failed", err)
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
