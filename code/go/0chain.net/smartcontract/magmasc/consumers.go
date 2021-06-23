package magmasc

import (
	"encoding/json"
	"sort"

	chain "0chain.net/chaincore/chain/state"
	"0chain.net/core/util"
)

type (
	// Consumers represents sorted Consumer nodes, used to inserting,
	// removing or getting from state.StateContextI with AllConsumersKey.
	Consumers struct {
		Nodes sortedConsumers
	}

	// sortedConsumers represents slice of Consumer sorted in alphabetic order by ID.
	// sortedConsumers allows O(logN) access.
	sortedConsumers []*Consumer
)

var (
	// Make sure Consumers implements Serializable interface.
	_ util.Serializable = (*Consumers)(nil)
)

// Decode implements util.Serializable interface.
func (m *Consumers) Decode(blob []byte) error {
	var consumers Consumers
	if err := json.Unmarshal(blob, &consumers); err != nil {
		return wrapError(errCodeDecode, errTextDecode, err)
	}

	*m = consumers

	return nil
}

// Encode implements util.Serializable interface.
func (m *Consumers) Encode() []byte {
	blob, _ := json.Marshal(m)
	return blob
}

// contains looks for given Consumer by provided
// smart-contract ID and state.StateContextI.
// If Consumer will be found it returns true, else false.
func (m *Consumers) contains(scID string, consumer *Consumer, sci chain.StateContextI) bool {
	id := consumer.ID
	for _, c := range m.Nodes {
		if c.ID == id {
			return true
		}
	}

	uid := nodeUID(scID, id, consumerType)
	if _, err := sci.GetTrieNode(uid); err == nil {
		return true
	}

	return false
}

func (m *sortedConsumers) add(consumer *Consumer) bool {
	sc := *m

	size := len(sc)
	if size == 0 {
		*m = append(sc, consumer)
		return true // appended
	}

	id := consumer.ID
	idx := sort.Search(size, func(idx int) bool {
		return sc[idx].ID >= id
	})
	if idx == size { // out of bounds
		*m = append(sc, consumer)
		return true // appended
	}

	if sc[idx].ID == id { // the same
		sc[idx] = consumer // replace
		return false       // already have
	}

	*m = append(sc[:idx], append([]*Consumer{consumer}, sc[idx:]...)...) // next

	return true // appended
}

func (m *sortedConsumers) get(id string) (*Consumer, bool) {
	sc := *m
	size := len(sc)

	idx := sort.Search(size, func(idx int) bool {
		return sc[idx].ID >= id
	})
	if idx == size || sc[idx].ID != id { // not found
		return nil, false
	}

	return sc[idx], true // found
}

func (m *sortedConsumers) getIndex(id string) (int, bool) {
	sc := *m
	size := len(sc)

	idx := sort.Search(size, func(idx int) bool {
		return sc[idx].ID >= id
	})
	if idx == size || sc[idx].ID != id { // not found
		return 0, false
	}

	return idx, true // found
}

func (m *sortedConsumers) remove(id string) bool {
	idx, ok := m.getIndex(id)
	if ok {
		m.removeByIndex(idx)
	}

	return ok
}

func (m *sortedConsumers) removeByIndex(idx int) {
	sc := *m
	*m = append(sc[:idx], sc[idx+1:]...)
}

func (m *sortedConsumers) update(consumer *Consumer) bool {
	idx, ok := m.getIndex(consumer.ID)
	if ok {
		(*m)[idx] = consumer // replace if found
	}

	return ok
}

// extractConsumers extracts all consumers represented in JSON bytes
// stored in state.StateContextI with AllConsumersKey.
// extractConsumers returns err if state.StateContextI does not contain
// consumers or stored bytes have invalid format.
func extractConsumers(sci chain.StateContextI) (*Consumers, error) {
	consumers := Consumers{}

	list, err := sci.GetTrieNode(AllConsumersKey)
	if list == nil || err == util.ErrValueNotPresent {
		return &consumers, nil
	}
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(list.Encode(), &consumers); err != nil {
		return nil, wrapError(errCodeDecode, errTextDecode, err)
	}

	return &consumers, nil
}
