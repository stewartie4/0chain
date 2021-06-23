package magmasc

import (
	"encoding/json"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

type (
	// DataUsage represents session data sage implementation.
	DataUsage struct {
		Amount        int64            `json:"amount"`
		DownloadBytes int64            `json:"download_bytes"`
		UploadBytes   int64            `json:"upload_bytes"`
		SessionID     datastore.Key    `json:"session_id"`
		Timestamp     common.Timestamp `json:"timestamp"`
	}
)

var (
	// Make sure tokenPool implements Serializable interface.
	_ util.Serializable = (*DataUsage)(nil)
)

// Decode implements util.Serializable interface.
func (m *DataUsage) Decode(blob []byte) error {
	var dataUsage DataUsage
	if err := json.Unmarshal(blob, &dataUsage); err != nil {
		return wrapError(errCodeDecode, errTextDecode, err)
	}

	*m = dataUsage

	return nil
}

// Encode implements util.Serializable interface.
func (m *DataUsage) Encode() []byte {
	blob, _ := json.Marshal(m)
	return blob
}

// validate checks DataUsage for correctness.
// If it is not return errDataUsageInvalid.
func (m *DataUsage) validate() error {
	switch { // is invalid
	case m.SessionID == "":
	case m.DownloadBytes <= 0:
	case m.UploadBytes <= 0:
	case m.Timestamp <= 0 || !common.Within(int64(m.Timestamp), int64(providerDataUsageDuration)):

	default: // is valid
		return nil
	}

	return errDataUsageInvalid
}

// uid returns uniq id used to saving Acknowledgment into chain state.
func (m *DataUsage) uid(scID datastore.Key) datastore.Key {
	return "sc:" + scID + ":datausage:" + m.SessionID
}
