package magmasc

import (
	"encoding/json"

	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

type (
	// Consumer represents consumers node stored in block chain.
	Consumer struct {
		ID datastore.Key `json:"id"`
	}
)

var (
	// Make sure Consumer implements Serializable interface.
	_ util.Serializable = (*Consumer)(nil)
)

// Decode implements util.Serializable interface.
func (m *Consumer) Decode(blob []byte) error {
	var consumer Consumer
	if err := json.Unmarshal(blob, &consumer); err != nil {
		return wrapError(errCodeDecode, errTextDecode, err)
	}

	*m = consumer

	return nil
}

// Encode implements util.Serializable interface.
func (m *Consumer) Encode() []byte {
	blob, _ := json.Marshal(m)
	return blob
}

// GetType returns Consumer's type.
func (m *Consumer) GetType() string {
	return consumerType
}

// consumerUID returns uniq id used to saving consumerPools into chain state.
func consumerUID(scID, id datastore.Key) datastore.Key {
	return "sc:" + scID + ":consumer:" + id
}
