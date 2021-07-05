package magmasc

import (
	"encoding/json"
	"math"

	magma "github.com/magma/augmented-networks/accounting/protos"

	"0chain.net/core/common"
	"0chain.net/core/util"
)

type (
	// ProviderTerms represents information of provider and services terms.
	ProviderTerms struct {
		Terms
		QoS magma.QoS `json:"qos"`
	}

	// Terms represents information of provider terms.
	Terms struct {
		Price     uint64           `json:"price"`      // per byte
		Volume    uint64           `json:"volume"`     // in bytes
		ExpiredAt common.Timestamp `json:"expired_at"` // valid till
	}
)

var (
	// Make sure ProviderTerms implements Serializable interface.
	_ util.Serializable = (*ProviderTerms)(nil)
)

// Decode implements util.Serializable interface.
func (m *ProviderTerms) Decode(blob []byte) error {
	var terms ProviderTerms
	if err := json.Unmarshal(blob, &terms); err != nil {
		return errDecodeData.WrapErr(err)
	}
	if err := terms.validate(); err != nil {
		return errDecodeData.WrapErr(err)
	}

	m.Price = terms.Price
	m.Volume = terms.Volume
	m.ExpiredAt = terms.ExpiredAt
	m.QoS.UploadMbps = terms.QoS.UploadMbps
	m.QoS.DownloadMbps = terms.QoS.DownloadMbps

	return nil
}

// Encode implements util.Serializable interface.
func (m *ProviderTerms) Encode() []byte {
	blob, _ := json.Marshal(m)
	return blob
}

// Equal reports whether the ProviderTerms are the same to given terms.
func (m *ProviderTerms) Equal(terms ProviderTerms) bool {
	return m.Price == terms.Price &&
		m.ExpiredAt == terms.ExpiredAt &&
		m.QoS.UploadMbps == terms.QoS.UploadMbps &&
		m.QoS.DownloadMbps == terms.QoS.DownloadMbps
}

// GetVolume returns the Volume value of provider terms.
// If the Volume value is empty calculates it by the terms.
func (m *ProviderTerms) GetVolume() uint64 {
	if m.Volume <= 0 {
		// convert to byte per second
		byteps := float64((m.QoS.UploadMbps + m.QoS.DownloadMbps) / octetSize)
		// duration of service
		duration := float64(m.ExpiredAt - common.Now())
		// round the volume of bps mul by duration
		m.Volume = uint64(math.Round(byteps * duration))
	}

	return m.Volume
}

// decrease makes automatically decrease provider terms by config.
func (m *ProviderTerms) decrease() *ProviderTerms {
	m.Price -= providerTermsAutoUpdatePrice // cents
	m.ExpiredAt = common.Now() + providerTermsProlongDuration

	m.QoS.UploadMbps += providerTermsAutoUpdateQoS   // 1KBPS
	m.QoS.DownloadMbps += providerTermsAutoUpdateQoS // 1KBPS

	return m
}

// expired checks the expiration time of the provider's terms.
func (m *ProviderTerms) expired() bool {
	return m.ExpiredAt <= common.Now()+providerTermsExpiredDuration
}

// increase makes automatically increase provider terms by config.
func (m *ProviderTerms) increase() *ProviderTerms {
	m.Price += providerTermsAutoUpdatePrice // cents
	m.ExpiredAt = common.Now() + providerTermsProlongDuration

	m.QoS.UploadMbps -= providerTermsAutoUpdateQoS   // 1KBPS
	m.QoS.DownloadMbps -= providerTermsAutoUpdateQoS // 1KBPS

	return m
}

// validate checks ProviderTerms for correctness.
// If it is not return errProviderTermsInvalid.
func (m *ProviderTerms) validate() error {
	switch { // is invalid
	case m.Price <= 0:
	case m.QoS.UploadMbps <= 0:
	case m.QoS.DownloadMbps <= 0:

	default: // is valid
		return nil
	}

	return errProviderTermsInvalid
}
