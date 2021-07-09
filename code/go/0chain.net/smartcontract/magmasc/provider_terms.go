package magmasc

import (
	"encoding/json"
	"math/big"

	magma "github.com/magma/augmented-networks/accounting/protos"

	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/util"
)

type (
	// ProviderTerms represents information of provider and services terms.
	ProviderTerms struct {
		Terms
		QoS magma.QoS `json:"qos"`
	}

	// Terms represents information of provider terms for a session.
	Terms struct {
		Price     float32          `json:"price"`      // tokens per Megabyte
		Volume    uint64           `json:"volume"`     // bytes per a session
		ExpiredAt common.Timestamp `json:"expired_at"` // timestamp till a session valid
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
func (m *ProviderTerms) Equal(terms *ProviderTerms) bool {
	return m.Price == terms.Price &&
		m.ExpiredAt == terms.ExpiredAt &&
		m.QoS.UploadMbps == terms.QoS.UploadMbps &&
		m.QoS.DownloadMbps == terms.QoS.DownloadMbps
}

// GetAmount returns calculated amount value of provider terms.
// NOTE: math/big must be used to avoid inaccuracies of floating point operations.
func (m *ProviderTerms) GetAmount() (amount state.Balance) {
	price := m.GetPrice()
	if price > 0 {
		amount = state.Balance(price * m.GetVolume())
	}

	return amount
}

// GetVolume returns calculated volume value of provider terms.
// If the Volume value is empty it will be calculates by the terms.
// NOTE: math/big must be used to avoid inaccuracies of floating point operations.
func (m *ProviderTerms) GetVolume() uint64 {
	if m.Volume == 0 {
		mbps := big.NewFloat(0).Add( // UploadMbps + DownloadMbps
			big.NewFloat(float64(m.QoS.UploadMbps)),
			big.NewFloat(float64(m.QoS.DownloadMbps)),
		)

		m.Volume, _ = big.NewFloat(0).Mul(
			big.NewFloat(0).Quo(mbps, big.NewFloat(octet)),            // convert to bytes per second
			big.NewFloat(0).SetInt64(int64(m.ExpiredAt-common.Now())), // duration in seconds
		).Uint64() // rounded of bytes per second multiplied by duration
	}

	return m.Volume
}

// GetPrice returns calculated price value of provider terms.
// NOTE: math/big must be used to avoid inaccuracies of floating point operations.
func (m *ProviderTerms) GetPrice() (price uint64) {
	if m.Price > 0 {
		price, _ = big.NewFloat(0).Mul( // convert to token price
			big.NewFloat(billion),
			big.NewFloat(float64(m.Price)),
		).Uint64() // rounded of price multiplied by volume
	}

	return price
}

// decrease makes automatically decrease provider terms by config.
func (m *ProviderTerms) decrease() *ProviderTerms {
	// quality of service up
	m.QoS.UploadMbps += providerTermsAutoUpdateQoS
	m.QoS.DownloadMbps += providerTermsAutoUpdateQoS

	if m.Price > 0 { // price of service down
		m.Price -= providerTermsAutoUpdatePrice
	}

	m.ExpiredAt = common.Now() + providerTermsProlongDuration

	return m
}

// expired checks the expiration time of the provider's terms.
func (m *ProviderTerms) expired() bool {
	return m.ExpiredAt <= common.Now()+providerTermsExpiredDuration
}

// increase makes automatically increase provider terms by config.
func (m *ProviderTerms) increase() *ProviderTerms {
	// price of service up
	m.Price += providerTermsAutoUpdatePrice

	if m.QoS.UploadMbps > 0 { // quality of service down
		m.QoS.UploadMbps -= providerTermsAutoUpdateQoS
	}
	if m.QoS.DownloadMbps > 0 { // quality of service down
		m.QoS.DownloadMbps -= providerTermsAutoUpdateQoS
	}

	m.ExpiredAt = common.Now() + providerTermsProlongDuration

	return m
}

// validate checks ProviderTerms for correctness.
// If it is not return errProviderTermsInvalid.
func (m *ProviderTerms) validate() error {
	switch { // is invalid
	case m.QoS.UploadMbps <= 0:
	case m.QoS.DownloadMbps <= 0:

	default: // is valid
		return nil
	}

	return errProviderTermsInvalid
}
