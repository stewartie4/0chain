package magmasc

import (
	"encoding/json"
	"reflect"
	"testing"
)

func Test_Billing_Amount(t *testing.T) {
	t.Parallel()

	tests := [2]struct {
		name string
		bill Billing
		want int64
	}{
		{
			name: "Amount_15_OK",
			bill: mockBilling(),
			want: 15,
		},
		{
			name: "Amount_Zero_OK",
			bill: make(Billing, 0),
			want: 0,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if gotAmount := test.bill.Amount(); gotAmount != test.want {
				t.Errorf("Amount() = %v, want %v", gotAmount, test.want)
			}
		})
	}
}

func Test_Billing_Decode(t *testing.T) {
	t.Parallel()

	bill := mockBilling()
	blob, _ := json.Marshal(bill)

	tests := [2]struct {
		name    string
		blob    []byte
		want    Billing
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

			got := Billing{}
			if err := got.Decode(test.blob); (err != nil) != test.wantErr {
				t.Errorf("Decode() error: %v | want: %v", err, nil)
			}
			if test.want != nil && !reflect.DeepEqual(got, test.want) {
				t.Errorf("Decode() got: %v | want: %v", got, test.want)
			}
		})
	}
}

func Test_Billing_Encode(t *testing.T) {
	t.Parallel()

	bill := mockBilling()
	blob, _ := json.Marshal(bill)

	tests := [1]struct {
		name string
		bill Billing
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
				t.Errorf("Encode() got: %v | want: %v", got, test.want)
			}
		})
	}
}
