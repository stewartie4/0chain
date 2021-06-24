package magmasc

import (
	"0chain.net/chaincore/tokenpool"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

type (
	// tokenPool represents token pool wrapper implementation.
	tokenPool struct {
		tokenpool.ZcnPool `json:"pool"` // embedded token pool

		ClientID   datastore.Key `json:"client_id"`
		DelegateID datastore.Key `json:"delegate_id"`
	}
)

var (
	// Make sure tokenPool implements Serializable interface.
	_ util.Serializable = (*tokenPool)(nil)
)

// uid returns uniq id used to saving tokenPool into chain state.
func (m *tokenPool) uid(parentUID datastore.Key) datastore.Key {
	return parentUID + ":tokenpool:" + m.ID
}
