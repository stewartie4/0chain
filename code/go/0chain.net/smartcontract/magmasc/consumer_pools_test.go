package magmasc

import (
	"encoding/json"
	"reflect"
	"testing"

	"0chain.net/chaincore/chain/state"
	tx "0chain.net/chaincore/transaction"
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

	tests := [4]struct {
		name  string
		txn   *tx.Transaction
		sci   state.StateContextI
		pools *consumerPools
		error error
	}{
		{
			name:  "OK",
			txn:   &tx.Transaction{ClientID: "client_id", Value: 1},
			sci:   sci,
			pools: pools,
			error: nil,
		},
		{
			name:  "Neg_TXN_Value_ERR",
			txn:   &tx.Transaction{Value: -1},
			sci:   sci,
			pools: pools,
			error: errNegativeTxnValue,
		},
		{
			name:  "Node_Not_Found_ERR",
			txn:   &tx.Transaction{},
			sci:   sci,
			pools: pools,
			error: util.ErrNodeNotFound,
		},
		{
			name:  "Insufficient_Funds_ERR",
			txn:   &tx.Transaction{ClientID: "client_id", Value: 1001},
			sci:   sci,
			pools: pools,
			error: errInsufficientFunds,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if err := test.pools.checkConditions(test.txn, test.sci); !errIs(err, test.error) {
				t.Errorf("checkConditions() error: %v | want: %v", err, test.error)
			}
		})
	}
}
