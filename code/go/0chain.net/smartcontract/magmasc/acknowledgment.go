package magmasc

import (
	"encoding/json"

	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

type (
	// Acknowledgment contains the necessary data obtained when the consumer
	// accepts the provider terms and stores in the state of the blockchain
	// as a result of performing the consumerAcceptTerms MagmaSmartContract function.
	Acknowledgment struct {
		AccessPointID string        `json:"access_point_id"`
		ConsumerID    string        `json:"consumer_id"`
		ProviderID    string        `json:"provider_id"`
		SessionID     string        `json:"session_id"`
		ProviderTerms ProviderTerms `json:"provider_terms"`
	}
)

var (
	// Make sure Acknowledgment implements Serializable interface.
	_ util.Serializable = (*Acknowledgment)(nil)
)

// Decode implements util.Serializable interface.
func (m *Acknowledgment) Decode(blob []byte) error {
	var ack Acknowledgment
	if err := json.Unmarshal(blob, &ack); err != nil {
		return errWrap(errCodeDecode, errTextDecode, err)
	}

	*m = ack

	return nil
}

// Encode implements util.Serializable interface.
func (m *Acknowledgment) Encode() []byte {
	blob, _ := json.Marshal(m)
	return blob
}

// validate checks Acknowledgment for correctness.
// If it is not return errAcknowledgmentInvalid.
func (m *Acknowledgment) validate() error {
	switch { // is invalid
	case m.AccessPointID == "":
	case m.ProviderID == "":
	case m.SessionID == "":

	default:
		return nil // is valid
	}

	return errAcknowledgmentInvalid
}

// uid returns uniq id used to saving Acknowledgment into chain state.
func (m *Acknowledgment) uid(scID datastore.Key) datastore.Key {
	return "sc:" + scID + ":acknowledgment:" + m.SessionID
}
