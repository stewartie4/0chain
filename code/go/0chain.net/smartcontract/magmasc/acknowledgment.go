package magmasc

import (
	bmp "github.com/0chain/bandwidth_marketplace/code/core/magmasc"

	"0chain.net/core/datastore"
)

type (
	// Acknowledgment contains the necessary data obtained when the consumer
	// accepts the provider terms and stores in the state of the blockchain
	// as a result of performing the consumerAcceptTerms MagmaSmartContract function.
	Acknowledgment struct{ *bmp.Acknowledgment }
)

// newAcknowledgment returns constructed Acknowledgment.
func newAcknowledgment(id datastore.Key) *Acknowledgment {
	return &Acknowledgment{
		Acknowledgment: &bmp.Acknowledgment{SessionID: id},
	}
}

// uid returns uniq id used to saving Acknowledgment into chain state.
func (m *Acknowledgment) uid(scID datastore.Key) datastore.Key {
	return "sc:" + scID + ":acknowledgment:" + m.SessionID
}
