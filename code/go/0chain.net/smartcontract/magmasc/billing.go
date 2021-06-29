package magmasc

import (
	"encoding/json"
	"sync"

	"0chain.net/core/util"
)

type (
	Billing struct {
		DataUsage []*DataUsage
		rwMutex   sync.RWMutex
	}
)

var (
	// Make sure tokenPool implements Serializable interface.
	_ util.Serializable = (*Billing)(nil)
)

// Amount returns the full sum of data usage amounts according to the billing data.
func (m *Billing) Amount() (amount int64) {
	m.rwMutex.Lock()
	defer m.rwMutex.Unlock()

	for _, usage := range m.DataUsage {
		amount += usage.Amount
	}

	return amount
}

// Decode implements util.Serializable interface.
func (m *Billing) Decode(blob []byte) error {
	var bill Billing
	if err := json.Unmarshal(blob, &bill); err != nil {
		return errWrap(errCodeDecode, errTextDecode, err)
	}
	if bill.DataUsage != nil {
		m.rwMutex.Lock()
		m.DataUsage = bill.DataUsage
		m.rwMutex.Unlock()
	}

	return nil
}

// Encode implements util.Serializable interface.
func (m *Billing) Encode() []byte {
	m.rwMutex.Lock()
	blob, _ := json.Marshal(m)
	m.rwMutex.Unlock()

	return blob
}
