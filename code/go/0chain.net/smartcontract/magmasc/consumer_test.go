package magmasc

import (
	"encoding/json"
	"reflect"
	"testing"

	"0chain.net/core/datastore"
)

func Test_Consumer_Decode(t *testing.T) {
	t.Parallel()

	cons := mockConsumer()
	blob, err := json.Marshal(cons)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v | want: %v", err, nil)
	}

	tests := [2]struct {
		name    string
		blob    []byte
		want    Consumer
		wantErr bool
	}{
		{
			name: "OK",
			blob: blob,
			want: cons,
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

			got := Consumer{}
			if err = got.Decode(test.blob); (err != nil) != test.wantErr {
				t.Errorf("Decode() error: %v | want: %v", err, nil)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("Decode() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_Consumer_Encode(t *testing.T) {
	t.Parallel()

	cons := mockConsumer()
	blob, err := json.Marshal(cons)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v | want: %v", err, nil)
	}

	tests := [1]struct {
		name string
		cons Consumer
		want []byte
	}{
		{
			name: "OK",
			cons: cons,
			want: blob,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.cons.Encode(); !reflect.DeepEqual(got, test.want) {
				t.Errorf("Encode() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_Consumer_GetType(t *testing.T) {
	t.Parallel()

	tests := [1]struct {
		name string
		want string
	}{
		{
			name: "OK",
			want: consumerType,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			cons := Consumer{}
			if got := cons.GetType(); got != test.want {
				t.Errorf("GetType() got: %v | want: %v", got, test.want)
			}
		})
	}
}

func Test_consumerUID(t *testing.T) {
	t.Parallel()

	const (
		scID    = "sc_id"
		consID  = "consumer_id"
		consUID = "sc:" + scID + ":consumer:" + consID
	)

	tests := [1]struct {
		name   string
		scID   datastore.Key
		consID datastore.Key
		want   datastore.Key
	}{
		{
			name:   "OK",
			scID:   scID,
			consID: consID,
			want:   consUID,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := consumerUID(test.scID, test.consID); got != test.want {
				t.Errorf("consumerUID() got: %v | want: %v", got, test.want)
			}
		})
	}
}
