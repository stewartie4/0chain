package magmasc

import (
	"encoding/json"
	"reflect"
	"testing"

	chain "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

func Test_Provider_Decode(t *testing.T) {
	t.Parallel()

	prov := mockProvider()
	blob, _ := json.Marshal(prov)

	tests := [2]struct {
		name    string
		blob    []byte
		want    Provider
		wantErr bool
	}{
		{
			name: "OK",
			blob: blob,
			want: prov,
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

			got := Provider{}
			if err := got.Decode(test.blob); (err != nil) != test.wantErr {
				t.Errorf("Decode() error: %v | want: %v", err, nil)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("Decode() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_Provider_Encode(t *testing.T) {
	t.Parallel()

	prov := mockProvider()
	blob, _ := json.Marshal(prov)

	tests := [1]struct {
		name string
		prov Provider
		want []byte
	}{
		{
			name: "OK",
			prov: prov,
			want: blob,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.prov.Encode(); !reflect.DeepEqual(got, test.want) {
				t.Errorf("Encode() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_Provider_GetType(t *testing.T) {
	t.Parallel()

	tests := [1]struct {
		name string
		want string
	}{
		{
			name: "OK",
			want: providerType,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			prov := Provider{}
			if got := prov.GetType(); got != test.want {
				t.Errorf("GetType() got: %v | want: %v", got, test.want)
			}
		})
	}
}

func Test_Provider_termsDecrease(t *testing.T) {
	t.Parallel()

	prov := mockProvider()

	provDec := prov
	provDec.Terms.Price -= providerTermsAutoUpdatePrice
	provDec.Terms.ExpiredAt = common.Now() + common.Timestamp(providerTermsProlongDuration)
	provDec.Terms.QoS.UploadMbps += providerTermsAutoUpdateQoS
	provDec.Terms.QoS.DownloadMbps += providerTermsAutoUpdateQoS

	tests := [1]struct {
		name string
		prov Provider
		want *Provider
	}{
		{
			name: "OK",
			prov: prov,
			want: &provDec,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.prov.termsDecrease(); !reflect.DeepEqual(got, test.want) {
				t.Errorf("decrease() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_Provider_termsIncrease(t *testing.T) {
	t.Parallel()

	prov := mockProvider()

	provInc := prov
	provInc.Terms.Price += providerTermsAutoUpdatePrice
	provInc.Terms.ExpiredAt = common.Now() + common.Timestamp(providerTermsProlongDuration)
	provInc.Terms.QoS.UploadMbps -= providerTermsAutoUpdateQoS
	provInc.Terms.QoS.DownloadMbps -= providerTermsAutoUpdateQoS

	tests := [1]struct {
		name string
		prov Provider
		want *Provider
	}{
		{
			name: "OK",
			prov: prov,
			want: &provInc,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.prov.termsIncrease(); !reflect.DeepEqual(got, test.want) {
				t.Errorf("increase() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_extractProvider(t *testing.T) {
	t.Parallel()

	const scID = "sc_id"

	sci, prov := mockStateContextI(), mockProvider()
	if _, err := sci.InsertTrieNode(nodeUID(scID, prov.ID, providerType), &prov); err != nil {
		t.Fatalf("InsertTrieNode() got: %v | want: %v", err, nil)
	}
	node := mockInvalidJson{ID: "invalid_json_id"}
	if _, err := sci.InsertTrieNode(nodeUID(scID, node.ID, providerType), &node); err != nil {
		t.Fatalf("InsertTrieNode() got: %v | want: %v", err, nil)
	}

	tests := [3]struct {
		name    string
		sci     chain.StateContextI
		scID    datastore.Key
		nodeID  datastore.Key
		want    *Provider
		wantErr error
	}{
		{
			name:   "OK",
			sci:    sci,
			scID:   scID,
			nodeID: prov.ID,
			want:   &prov,
		},
		{
			name:    "ErrInvalidJSON",
			sci:     sci,
			scID:    scID,
			nodeID:  node.ID,
			wantErr: errDecodeData,
		},
		{
			name:    "ErrValueNotPresent",
			sci:     sci,
			scID:    scID,
			nodeID:  "node_not_present_id",
			wantErr: util.ErrValueNotPresent,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := extractProvider(test.scID, test.nodeID, test.sci)
			if err == nil && !reflect.DeepEqual(got, test.want) {
				t.Errorf("extractProvider() got: %v | want: %v", err, test.want)
				return
			}
			if !errIs(test.wantErr, err) {
				t.Errorf("extractProvider() error: %v | want: %v", err, test.wantErr)
			}
		})
	}
}
