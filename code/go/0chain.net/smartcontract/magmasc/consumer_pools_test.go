package magmasc

import (
	"encoding/json"
	"reflect"
	"testing"
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
