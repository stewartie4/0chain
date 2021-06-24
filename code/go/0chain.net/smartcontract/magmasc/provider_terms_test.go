package magmasc

import (
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"testing"

	"0chain.net/core/common"
)

func Test_ProviderTerms_Decode(t *testing.T) {
	t.Parallel()

	terms := mockProviderTerms()
	blob, _ := json.Marshal(terms)

	tests := [2]struct {
		name    string
		blob    []byte
		want    ProviderTerms
		wantErr bool
	}{
		{
			name: "OK",
			blob: blob,
			want: terms,
		},
		{
			name:    "ERR",
			blob:    []byte(":"), // invalid json,
			wantErr: true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := ProviderTerms{}
			if err := got.Decode(test.blob); (err != nil) != test.wantErr {
				t.Errorf("Decode() error: %v | want: %v", err, nil)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("Decode() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_ProviderTerms_Encode(t *testing.T) {
	t.Parallel()

	terms := mockProviderTerms()
	blob, _ := json.Marshal(terms)

	tests := [1]struct {
		name  string
		terms ProviderTerms
		want  []byte
	}{
		{
			name:  "OK",
			terms: terms,
			want:  blob,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.terms.Encode(); !reflect.DeepEqual(got, test.want) {
				t.Errorf("Encode() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_ProviderTerms_Equal(t *testing.T) {
	t.Parallel()

	terms := mockProviderTerms()

	termsDiffPrice := terms
	termsDiffPrice.Price += 1

	termsDiffExpiredAt := terms
	termsDiffExpiredAt.ExpiredAt += 1

	termsDiffQoSUploadMbps := terms
	termsDiffQoSUploadMbps.QoS.UploadMbps += 0.1

	termsDiffQoSDownloadMbps := terms
	termsDiffQoSDownloadMbps.QoS.DownloadMbps += 0.1

	tests := [5]struct {
		name  string
		terms ProviderTerms
		with  ProviderTerms
		want  bool
	}{
		{
			name:  "EQUAL",
			terms: terms,
			with:  terms,
			want:  true,
		},
		{
			name:  "termsDiffPrice",
			terms: terms,
			with:  termsDiffPrice,
			want:  false,
		},
		{
			name:  "termsDiffExpiredAt",
			terms: terms,
			with:  termsDiffExpiredAt,
			want:  false,
		},
		{
			name:  "termsDiffQoSUploadMbps",
			terms: terms,
			with:  termsDiffQoSUploadMbps,
			want:  false,
		},
		{
			name:  "termsDiffQoSDownloadMbps",
			terms: terms,
			with:  termsDiffQoSDownloadMbps,
			want:  false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.terms.Equal(test.with); got != test.want {
				t.Errorf("Equal() got: %v | want: %v", got, test.want)
			}
		})
	}
}

func Test_ProviderTerms_GetVolume(t *testing.T) {
	t.Parallel()

	terms := mockProviderTerms()
	byteps := float64((terms.QoS.DownloadMbps + terms.QoS.DownloadMbps) / octetSize)
	duration := float64(terms.ExpiredAt - common.Now())
	volume := int64(math.Round(byteps * duration))

	tests := [1]struct {
		name  string
		terms ProviderTerms
		want  int64
	}{
		{
			name:  "OK",
			terms: terms,
			want:  volume,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if test.terms.Volume != 0 { // must be zero before first call GetVolume()
				t.Errorf("ProviderTerms.Volume is: %v | want: %v", test.terms.Volume, 0)
			}
			if got := test.terms.GetVolume(); got != test.want {
				t.Errorf("GetVolume() got: %v | want: %v", got, test.want)
			}
			if test.terms.Volume != test.want { // must be the same value with test.want after called GetVolume()
				t.Errorf("ProviderTerms.Volume is: %v | want: %v", test.terms.Volume, test.want)
			}
		})
	}
}

func Test_ProviderTerms_decrease(t *testing.T) {
	t.Parallel()

	terms := mockProviderTerms()

	termsDec := terms
	termsDec.Price -= providerTermsAutoUpdatePrice
	termsDec.ExpiredAt = common.Now() + common.Timestamp(providerTermsProlongDuration)
	termsDec.QoS.UploadMbps += providerTermsAutoUpdateQoS
	termsDec.QoS.DownloadMbps += providerTermsAutoUpdateQoS

	tests := [1]struct {
		name  string
		terms ProviderTerms
		want  *ProviderTerms
	}{
		{
			name:  "OK",
			terms: terms,
			want:  &termsDec,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.terms.decrease(); !reflect.DeepEqual(got, test.want) {
				t.Errorf("decrease() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_ProviderTerms_expired(t *testing.T) {
	t.Parallel()

	termsValid := mockProviderTerms()

	termsExpired := termsValid
	termsExpired.ExpiredAt = common.Now()

	tests := [2]struct {
		name  string
		terms ProviderTerms
		want  bool
	}{
		{
			name:  "FALSE",
			terms: termsValid,
			want:  false,
		},
		{
			name:  "TRUE",
			terms: termsExpired,
			want:  true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.terms.expired(); got != test.want {
				t.Errorf("expired() got: %v | want: %v", got, test.want)
			}
		})
	}
}

func Test_ProviderTerms_increase(t *testing.T) {
	t.Parallel()

	terms := mockProviderTerms()

	termsInc := terms
	termsInc.Price += providerTermsAutoUpdatePrice
	termsInc.ExpiredAt = common.Now() + common.Timestamp(providerTermsProlongDuration)
	termsInc.QoS.UploadMbps -= providerTermsAutoUpdateQoS
	termsInc.QoS.DownloadMbps -= providerTermsAutoUpdateQoS

	tests := [1]struct {
		name  string
		terms ProviderTerms
		want  *ProviderTerms
	}{
		{
			name:  "OK",
			terms: terms,
			want:  &termsInc,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.terms.increase(); !reflect.DeepEqual(got, test.want) {
				t.Errorf("increase() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_ProviderTerms_validate(t *testing.T) {
	t.Parallel()

	termsValid := mockProviderTerms()

	termsZeroPrice := termsValid
	termsZeroPrice.Price = 0

	termsZeroQoSUploadMbps := termsValid
	termsZeroQoSUploadMbps.QoS.UploadMbps = 0

	termsZeroQoSDownloadMbps := termsValid
	termsZeroQoSDownloadMbps.QoS.DownloadMbps = 0

	tests := [4]struct {
		name  string
		terms ProviderTerms
		want  error
	}{
		{
			name:  "OK",
			terms: termsValid,
			want:  nil,
		},
		{
			name:  "ZeroPrice",
			terms: termsZeroPrice,
			want:  errProviderTermsInvalid,
		},
		{
			name:  "ZeroQoSUploadMbps",
			terms: termsZeroQoSUploadMbps,
			want:  errProviderTermsInvalid,
		},
		{
			name:  "ZeroQoSDownloadMbps",
			terms: termsZeroQoSDownloadMbps,
			want:  errProviderTermsInvalid,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if err := test.terms.validate(); !errors.Is(err, test.want) {
				t.Errorf("validate() error: %v | want: %v", err, test.want)
			}
		})
	}
}
