package magmasc

import (
	"encoding/json"
	"reflect"
	"testing"

	"0chain.net/chaincore/state"
)

func Test_Billing_Amount(t *testing.T) {
	t.Parallel()

	tests := [2]struct {
		name string
		bill *Billing
		want state.Balance
	}{
		{
			name: "Amount_15_OK",
			bill: mockBilling(),
			want: 15,
		},
		{
			name: "Amount_Zero_OK",
			bill: &Billing{},
			want: 0,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.bill.Amount(); got != test.want {
				t.Errorf("Amount() got: %v | want: %v", got, test.want)
			}
		})
	}
}

func Test_Billing_Decode(t *testing.T) {
	t.Parallel()

	bill := mockBilling()
	blob, err := json.Marshal(bill)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v | want: %v", err, nil)
	}

	tests := [2]struct {
		name    string
		blob    []byte
		want    *Billing
		wantErr bool
	}{
		{
			name: "OK",
			blob: blob,
			want: bill,
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

			got := &Billing{}
			if err = got.Decode(test.blob); (err != nil) != test.wantErr {
				t.Errorf("Decode() error: %v | want: %v", err, nil)
			}
			if test.want != nil && !reflect.DeepEqual(got, test.want) {
				t.Errorf("Decode() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_Billing_Encode(t *testing.T) {
	t.Parallel()

	bill := mockBilling()
	blob, err := json.Marshal(bill)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v | want: %v", err, nil)
	}

	tests := [1]struct {
		name string
		bill *Billing
		want []byte
	}{
		{
			name: "OK",
			bill: bill,
			want: blob,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.bill.Encode(); !reflect.DeepEqual(got, test.want) {
				t.Errorf("Encode() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}
