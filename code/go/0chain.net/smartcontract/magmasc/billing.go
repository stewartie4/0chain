package magmasc

import (
	"encoding/json"

	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

type (
	Billing struct {
		Amount    int64         `json:"amount"`
		DataUsage *DataUsage    `json:"data_usage"`
		SessionID datastore.Key `json:"session_id"`
	}
)

var (
	// Make sure tokenPool implements Serializable interface.
	_ util.Serializable = (*Billing)(nil)
)

// Decode implements util.Serializable interface.
func (m *Billing) Decode(blob []byte) error {
	var bill Billing
	if err := json.Unmarshal(blob, &bill); err != nil {
		return errDecodeData.WrapErr(err)
	}
	if err := bill.DataUsage.validate(); err != nil {
		return errDecodeData.WrapErr(err)
	}

	m.Amount = bill.Amount
	m.DataUsage = bill.DataUsage
	m.SessionID = bill.SessionID

	return nil
}

// Encode implements util.Serializable interface.
func (m *Billing) Encode() []byte {
	blob, _ := json.Marshal(m)
	return blob
}

// validate checks given data usage is correctness for the billing.
func (m *Billing) validate(dataUsage *DataUsage) error {
	switch {
	case dataUsage == nil: // is invalid: data usage cannon be nil
	case m.SessionID != dataUsage.SessionID: // is invalid: invalid session id

	case m.DataUsage == nil: // is valid: have no data usage yet
		return nil

	// is invalid cases
	case m.DataUsage.SessionTime > dataUsage.SessionTime:
	case m.DataUsage.UploadBytes > dataUsage.UploadBytes:
	case m.DataUsage.DownloadBytes > dataUsage.DownloadBytes:

	default: // is valid: everything is ok
		return nil
	}

	return errDataUsageInvalid
}

// uid returns uniq id used to saving billing data into chain state.
func (m *Billing) uid(scID datastore.Key) datastore.Key {
	return "sc:" + scID + ":datausage:" + m.SessionID
}
