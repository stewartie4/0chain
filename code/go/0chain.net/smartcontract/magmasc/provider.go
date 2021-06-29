package magmasc

import (
	"encoding/json"

	chain "0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

type (
	// Provider represents providers node stored in block chain.
	Provider struct {
		ID    datastore.Key `json:"id"`
		Terms ProviderTerms `json:"terms"`
	}
)

var (
	// Make sure Provider implements Serializable interface.
	_ util.Serializable = (*Provider)(nil)
)

// Decode implements util.Serializable interface.
func (m *Provider) Decode(blob []byte) error {
	var provider Provider
	if err := json.Unmarshal(blob, &provider); err != nil {
		return errWrap(errCodeDecode, errTextDecode, err)
	}

	*m = provider

	return nil
}

// Encode implements util.Serializable interface.
func (m *Provider) Encode() []byte {
	blob, _ := json.Marshal(m)
	return blob
}

// GetType returns Provider's type.
func (m *Provider) GetType() string {
	return providerType
}

// termsIncrease increases Provider's terms.
func (m *Provider) termsIncrease() *Provider {
	m.Terms.increase()
	return m
}

// termsDecrease decreases Provider's terms.
func (m *Provider) termsDecrease() *Provider {
	m.Terms.decrease()
	return m
}

// extractProvider extracts Provider represented in JSON bytes
// stored in state.StateContextI and returns error if state.StateContextI
// does not contain Nodes or stored Nodes bytes have invalid format.
func extractProvider(scID, id string, sci chain.StateContextI) (*Provider, error) {
	data, err := sci.GetTrieNode(nodeUID(scID, id, providerType))
	if err != nil {
		return nil, err
	}

	provider := Provider{}
	if err = json.Unmarshal(data.Encode(), &provider); err != nil {
		return nil, errDecodeData.WrapErr(err)
	}

	return &provider, nil
}
