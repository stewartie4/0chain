package magmasc

import (
	"encoding/json"

	"0chain.net/core/util"
)

type (
	Billing []*DataUsage
)

var (
	// Make sure tokenPool implements Serializable interface.
	_ util.Serializable = (*Billing)(nil)
)

// Amount returns the full sum of data usage amounts according to the billing data.
func (m *Billing) Amount() (amount int64) {
	for _, usage := range *m {
		amount += usage.Amount
	}

	return amount
}

// Decode implements util.Serializable interface.
func (m *Billing) Decode(blob []byte) error {
	var billing Billing
	if err := json.Unmarshal(blob, &billing); err != nil {
		return errWrap(errCodeDecode, errTextDecode, err)
	}
	if billing != nil {
		*m = billing
	}

	return nil
}

// Encode implements util.Serializable interface.
func (m *Billing) Encode() []byte {
	blob, _ := json.Marshal(m)
	return blob
}
