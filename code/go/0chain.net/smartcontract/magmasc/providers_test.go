package magmasc

import (
	"encoding/json"
	"reflect"
	"testing"

	"0chain.net/chaincore/chain/state"
)

func Test_Providers_Decode(t *testing.T) {
	t.Parallel()

	list := mockProviders()
	blob, err := json.Marshal(list.Nodes.Sorted)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v | want: %v", err, nil)
	}

	tests := [2]struct {
		name    string
		blob    []byte
		want    Providers
		wantErr bool
	}{
		{
			name: "OK",
			blob: blob,
			want: list,
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

			got := Providers{}
			if err = got.Decode(test.blob); (err != nil) != test.wantErr {
				t.Errorf("Decode() error: %v | want: %v", err, nil)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("Decode() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_Providers_Encode(t *testing.T) {
	t.Parallel()

	list := mockProviders()
	blob, err := json.Marshal(list.Nodes.Sorted)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v | want: %v", err, nil)
	}

	tests := [1]struct {
		name string
		list Providers
		want []byte
	}{
		{
			name: "OK",
			list: list,
			want: blob,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.list.Encode(); !reflect.DeepEqual(got, test.want) {
				t.Errorf("Encode() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_Providers_contains(t *testing.T) {
	t.Parallel()

	const scID = "sc_id"

	sci, prov, list := mockStateContextI(), mockProvider(), mockProviders()
	if _, err := sci.InsertTrieNode(nodeUID(scID, prov.ID, providerType), &prov); err != nil {
		t.Fatalf("InsertTrieNode() got: %v | want: %v", err, nil)
	}

	tests := [3]struct {
		name string
		scID string
		prov *Provider
		list Providers
		sci  state.StateContextI
		want bool
	}{
		{
			name: "FALSE",
			scID: scID,
			prov: &Provider{ID: "not_present_provider_id"},
			list: list,
			sci:  sci,
			want: false,
		},
		{
			name: "InNodeList_TRUE",
			scID: scID,
			prov: list.Nodes.Sorted[0],
			list: list,
			want: true,
		},
		{
			name: "InStateContext_TRUE",
			scID: scID,
			prov: &prov,
			list: list,
			sci:  sci,
			want: true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.list.contains(test.scID, test.prov, test.sci); got != test.want {
				t.Errorf("contains() got: %v | want: %v", got, test.want)
			}
		})
	}
}

func Test_extractProviders(t *testing.T) {
	t.Parallel()

	sci, list := mockStateContextI(), mockProviders()
	if _, err := sci.InsertTrieNode(AllProvidersKey, &list); err != nil {
		t.Fatalf("InsertTrieNode() got: %v | want: %v", err, nil)
	}

	tests := [2]struct {
		name    string
		sci     state.StateContextI
		want    *Providers
		wantErr bool
	}{
		{
			name:    "OK",
			sci:     mockStateContextI(),
			want:    &Providers{Nodes: &providersSorted{}},
			wantErr: false,
		},
		{
			name:    "Nodes_OK",
			sci:     sci,
			want:    &list,
			wantErr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			got, err := extractProviders(test.sci)
			if (err != nil) != test.wantErr {
				t.Errorf("extractProviders() error: %v | want: %v", err, test.wantErr)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("extractProviders() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}
