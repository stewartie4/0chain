package magmasc

import (
	bmp "github.com/0chain/bandwidth_marketplace/code/core/magmasc"

	"0chain.net/core/datastore"
)

type (
	Billing struct{ *bmp.Billing }
)

// newBilling returns constructed Billing.
func newBilling(id datastore.Key) *Billing {
	return &Billing{
		Billing: &bmp.Billing{SessionID: id},
	}
}

// uid returns uniq id used to saving billing data into chain state.
func (m *Billing) uid(scID datastore.Key) datastore.Key {
	return "sc:" + scID + ":datausage:" + m.SessionID
}
