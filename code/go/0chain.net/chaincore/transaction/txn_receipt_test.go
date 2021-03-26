package transaction

import (
	"0chain.net/core/encryption"
	"0chain.net/core/util"
	"reflect"
	"testing"
)

func TestTxnReceipt_GetHash(t *testing.T) {
	hash := encryption.Hash("data")

	type fields struct {
		Transaction *Transaction
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "OK",
			fields: fields{
				Transaction: &Transaction{
					OutputHash: hash,
				},
			},
			want: hash,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rh := &TxnReceipt{
				Transaction: tt.fields.Transaction,
			}
			if got := rh.GetHash(); got != tt.want {
				t.Errorf("GetHash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTxnReceipt_GetHashBytes(t *testing.T) {
	hash := encryption.Hash("data")

	type fields struct {
		Transaction *Transaction
	}
	tests := []struct {
		name   string
		fields fields
		want   []byte
	}{
		{
			name: "OK",
			fields: fields{
				Transaction: &Transaction{
					OutputHash: hash,
				},
			},
			want: util.HashStringToBytes(hash),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rh := &TxnReceipt{
				Transaction: tt.fields.Transaction,
			}
			if got := rh.GetHashBytes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetHashBytes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewTransactionReceipt(t *testing.T) {
	txn := makeTestTxn()

	type args struct {
		t *Transaction
	}
	tests := []struct {
		name string
		args args
		want *TxnReceipt
	}{
		{
			name: "OK",
			args: args{t: &txn},
			want: &TxnReceipt{Transaction: &txn},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewTransactionReceipt(tt.args.t); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewTransactionReceipt() = %v, want %v", got, tt.want)
			}
		})
	}
}
