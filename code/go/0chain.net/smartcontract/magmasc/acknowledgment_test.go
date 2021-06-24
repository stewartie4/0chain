package magmasc

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"0chain.net/core/datastore"
)

func Test_Acknowledgment_Decode(t *testing.T) {
	t.Parallel()

	ackn := mockAcknowledgment()
	blob, _ := json.Marshal(ackn)

	tests := [2]struct {
		name    string
		blob    []byte
		want    Acknowledgment
		wantErr bool
	}{
		{
			name: "OK",
			blob: blob,
			want: ackn,
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

			got := Acknowledgment{}
			if err := got.Decode(test.blob); (err != nil) != test.wantErr {
				t.Errorf("Decode() error: %v | want: %v", err, nil)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("Decode() got: %v | want: %v", got, test.want)
			}
		})
	}
}

func Test_Acknowledgment_Encode(t *testing.T) {
	t.Parallel()

	ackn := mockAcknowledgment()
	blob, _ := json.Marshal(ackn)

	tests := [1]struct {
		name string
		ackn Acknowledgment
		want []byte
	}{
		{
			name: "OK",
			ackn: ackn,
			want: blob,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.ackn.Encode(); !reflect.DeepEqual(got, test.want) {
				t.Errorf("Encode() got: %v | want: %v", got, test.want)
			}
		})
	}
}

func Test_Acknowledgment_uid(t *testing.T) {
	t.Parallel()

	const (
		scID      = "sc_uid"
		sessionID = "session_id"
		acknUID   = "sc:" + scID + ":acknowledgment:" + sessionID
	)

	ackn := Acknowledgment{SessionID: sessionID}

	tests := [1]struct {
		name string
		ackn Acknowledgment
		scID datastore.Key
		want datastore.Key
	}{
		{
			name: "OK",
			ackn: ackn,
			scID: scID,
			want: acknUID,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.ackn.uid(test.scID); got != test.want {
				t.Errorf("uid() got: %v | want: %v", got, test.want)
			}
		})
	}
}

func Test_Acknowledgment_validate(t *testing.T) {
	t.Parallel()

	acknValid := mockAcknowledgment()

	acknEmptyAccessPointID := acknValid
	acknEmptyAccessPointID.AccessPointID = ""

	acknEmptyProviderID := acknValid
	acknEmptyProviderID.ProviderID = ""

	acknEmptySessionID := acknValid
	acknEmptySessionID.SessionID = ""

	tests := [4]struct {
		name string
		ackn Acknowledgment
		want error
	}{
		{
			name: "OK",
			ackn: acknValid,
			want: nil,
		},
		{
			name: "EmptyAccessPointID",
			ackn: acknEmptyAccessPointID,
			want: errAcknowledgmentInvalid,
		},
		{
			name: "EmptyProviderID",
			ackn: acknEmptyProviderID,
			want: errAcknowledgmentInvalid,
		},
		{
			name: "EmptySessionID",
			ackn: acknEmptySessionID,
			want: errAcknowledgmentInvalid,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if err := test.ackn.validate(); !errors.Is(err, test.want) {
				t.Errorf("validate() error: %v | want: %v", err, test.want)
			}
		})
	}
}
