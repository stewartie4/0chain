package magmasc

import (
	"encoding/json"
	"reflect"
	"testing"

	"0chain.net/chaincore/chain/state"
	"0chain.net/core/util"
)

func Test_consumerPools_Decode(t *testing.T) {
	t.Parallel()

	pools := mockConsumerPools()
	blob, err := json.Marshal(pools)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v | want: %v", err, nil)
	}

	tests := [2]struct {
		name    string
		blob    []byte
		want    *consumerPools
		wantErr bool
	}{
		{
			name:    "OK",
			blob:    blob,
			want:    pools,
			wantErr: false,
		},
		{
			name:    "ERR",
			blob:    []byte(":"), // invalid json,
			want:    &consumerPools{},
			wantErr: true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := &consumerPools{}
			if err = got.Decode(test.blob); (err != nil) != test.wantErr {
				t.Errorf("Decode() error: %v | want: %v", err, test.wantErr)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("Decode() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_consumerPools_Encode(t *testing.T) {
	t.Parallel()

	pools := mockConsumerPools()
	blob, err := json.Marshal(pools)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v | want: %v", err, nil)
	}

	tests := [1]struct {
		name  string
		pools *consumerPools
		want  []byte
	}{
		{
			name:  "OK",
			pools: pools,
			want:  blob,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.pools.Encode(); !reflect.DeepEqual(got, test.want) {
				t.Errorf("Encode() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_consumerPools_checkConditions(t *testing.T) {
	t.Parallel()

	sci, pools := mockStateContextI(), mockConsumerPools()

	ackn := mockAcknowledgment()

	acknNegPermsVolumeERR := mockAcknowledgment()
	acknNegPermsVolumeERR.ProviderTerms.Price = -1

	acknNodeNotFoundERR := mockAcknowledgment()
	acknNodeNotFoundERR.ConsumerID = ""

	acknInsufficientFundsERR := mockAcknowledgment()
	acknInsufficientFundsERR.ProviderTerms.Price = 10

	tests := [4]struct {
		name  string
		ackn  *Acknowledgment
		sci   state.StateContextI
		pools *consumerPools
		error error
	}{
		{
			name:  "OK",
			ackn:  ackn,
			sci:   sci,
			pools: pools,
			error: nil,
		},
		{
			name:  "Neg_Perms_Volume_ERR",
			ackn:  acknNegPermsVolumeERR,
			sci:   sci,
			pools: pools,
			error: errNegativeValue,
		},
		{
			name:  "Node_Not_Found_ERR",
			ackn:  acknNodeNotFoundERR,
			sci:   sci,
			pools: pools,
			error: util.ErrNodeNotFound,
		},
		{
			name:  "Insufficient_Funds_ERR",
			ackn:  acknInsufficientFundsERR,
			sci:   sci,
			pools: pools,
			error: errInsufficientFunds,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if err := test.pools.checkConditions(test.ackn, test.sci); !errIs(err, test.error) {
				t.Errorf("checkConditions() error: %v | want: %v", err, test.error)
			}
		})
	}
}
