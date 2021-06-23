package magmasc

import (
	"encoding/json"
	"sort"

	chain "0chain.net/chaincore/chain/state"
	"0chain.net/core/util"
)

type (
	// Providers represents sorted Provider nodes, used to inserting,
	// removing or getting from state.StateContextI with AllProvidersKey.
	Providers struct {
		Nodes sortedProviders
	}

	// sortedProviders represents slice of Provider sorted in alphabetic order by ID.
	// sortedProviders allows O(logN) access.
	sortedProviders []*Provider
)

var (
	// Make sure Providers implements Serializable interface.
	_ util.Serializable = (*Providers)(nil)
)

// Decode implements util.Serializable interface.
func (m *Providers) Decode(blob []byte) error {
	var providers Providers
	if err := json.Unmarshal(blob, &providers); err != nil {
		return wrapError(errCodeDecode, errTextDecode, err)
	}

	*m = providers

	return nil
}

// Encode implements util.Serializable interface.
func (m *Providers) Encode() []byte {
	blob, _ := json.Marshal(m)
	return blob
}

// contains looks for given Provider by provided
// smart-contract ID and state.StateContextI.
// If Provider will be found it returns true, else false.
func (m *Providers) contains(scID string, provider *Provider, sci chain.StateContextI) bool {
	id := provider.ID
	for _, c := range m.Nodes {
		if c.ID == id {
			return true
		}
	}

	uid := nodeUID(scID, id, providerType)
	if _, err := sci.GetTrieNode(uid); err == nil {
		return true
	}

	return false
}

func (m *sortedProviders) add(provider *Provider) bool {
	sp := *m

	size := len(sp)
	if size == 0 {
		*m = append(sp, provider)
		return true // appended
	}

	idx := sort.Search(size, func(idx int) bool {
		return sp[idx].ID >= provider.ID
	})
	if idx == size { // out of bounds
		*m = append(sp, provider)
		return true // appended
	}

	if sp[idx].ID == provider.ID { // the same
		sp[idx] = provider // replace
		return false       // already have
	}

	*m = append(sp[:idx], append([]*Provider{provider}, sp[idx:]...)...) // next

	return true // appended
}

func (m *sortedProviders) get(id string) (*Provider, bool) {
	sp := *m
	size := len(sp)

	idx := sort.Search(size, func(idx int) bool {
		return sp[idx].ID >= id
	})
	if idx == size || sp[idx].ID != id {
		return nil, false // not found
	}

	return sp[idx], true // found
}

func (m *sortedProviders) getIndex(id string) (int, bool) {
	sp := *m
	size := len(sp)

	idx := sort.Search(size, func(idx int) bool {
		return sp[idx].ID >= id
	})
	if idx == size || sp[idx].ID != id {
		return 0, false // not found
	}

	return idx, true // found
}

func (m *sortedProviders) remove(id string) bool {
	idx, ok := m.getIndex(id)
	if ok {
		m.removeByIndex(idx)
	}

	return ok
}

func (m *sortedProviders) removeByIndex(idx int) {
	sp := *m
	*m = append(sp[:idx], sp[idx+1:]...)
}

func (m *sortedProviders) update(provider *Provider) bool {
	idx, ok := m.getIndex(provider.ID)
	if ok {
		(*m)[idx] = provider // replace if found
	}

	return ok
}

// extractProviders extracts all provider Nodes represented
// in JSON bytes stored in state.StateContextI with AllProvidersKey
// and returns err if state.StateContextI does not contain Nodes
// or stored Nodes bytes have invalid format.
func extractProviders(sci chain.StateContextI) (*Providers, error) {
	providers := Providers{}

	list, err := sci.GetTrieNode(AllProvidersKey)
	if err != nil && err != util.ErrValueNotPresent {
		return nil, err
	}
	if list == nil || err == util.ErrValueNotPresent {
		return &providers, nil
	}

	if err = json.Unmarshal(list.Encode(), &providers); err != nil {
		return nil, wrapError(errCodeDecode, errTextDecode, err)
	}

	return &providers, nil
}
