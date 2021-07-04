package magmasc

import (
	"encoding/json"
	"sync"

	"0chain.net/chaincore/state"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

type (
	Billing struct {
		SessionID datastore.Key `json:"session_id"`
		DataUsage []*DataUsage  `json:"data_usage"`

		rwMutex sync.RWMutex
	}
)

var (
	// Make sure tokenPool implements Serializable interface.
	_ util.Serializable = (*Billing)(nil)
)

// Amount returns the full sum of data usage amounts according to the billing data.
func (m *Billing) Amount() (amount state.Balance) {
	m.rwMutex.Lock()
	defer m.rwMutex.Unlock()

	for _, usage := range m.DataUsage {
		amount += state.Balance(usage.Amount)
	}

	return amount
}

// Decode implements util.Serializable interface.
func (m *Billing) Decode(blob []byte) error {
	var bill Billing
	if err := json.Unmarshal(blob, &bill); err != nil {
		return errDecodeData.WrapErr(err)
	}

	m.rwMutex.Lock()
	m.SessionID = bill.SessionID
	if bill.DataUsage != nil {
		m.DataUsage = bill.DataUsage
	}
	m.rwMutex.Unlock()

	return nil
}

// Encode implements util.Serializable interface.
func (m *Billing) Encode() []byte {
	m.rwMutex.Lock()
	blob, _ := json.Marshal(m)
	m.rwMutex.Unlock()

	return blob
}

// uid returns uniq id used to saving billing data into chain state.
func (m *Billing) uid(scID datastore.Key) datastore.Key {
	return "sc:" + scID + ":datausage:" + m.SessionID
}
