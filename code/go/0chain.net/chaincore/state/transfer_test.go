package state

import (
	"0chain.net/core/datastore"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestNewTransfer(t *testing.T) {
	fromClientID := "from client id"
	toClientID := "to client id"
	amount := Balance(5)

	type args struct {
		fromClientID datastore.Key
		toClientID   datastore.Key
		amount       Balance
	}
	tests := []struct {
		name string
		args args
		want *Transfer
	}{
		{
			name: "OK",
			args: args{
				fromClientID: fromClientID,
				toClientID:   toClientID,
				amount:       amount,
			},
			want: &Transfer{Sender: fromClientID, Receiver: toClientID, Amount: amount},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewTransfer(tt.args.fromClientID, tt.args.toClientID, tt.args.amount); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewTransfer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTransfer_Encode(t1 *testing.T) {
	tr := NewTransfer("from client id", "to client id", 5)
	blob, err := json.Marshal(tr)
	if err != nil {
		t1.Fatal(err)
	}

	type fields struct {
		Sender   datastore.Key
		Receiver datastore.Key
		Amount   Balance
	}
	tests := []struct {
		name   string
		fields fields
		want   []byte
	}{
		{
			name:   "OK",
			fields: fields(*tr),
			want:   blob,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Transfer{
				Sender:   tt.fields.Sender,
				Receiver: tt.fields.Receiver,
				Amount:   tt.fields.Amount,
			}
			if got := t.Encode(); !reflect.DeepEqual(got, tt.want) {
				t1.Errorf("Encode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTransfer_Decode(t1 *testing.T) {
	tr := NewTransfer("from client id", "to client id", 5)
	blob, err := json.Marshal(tr)
	if err != nil {
		t1.Fatal(err)
	}

	type fields struct {
		Sender   datastore.Key
		Receiver datastore.Key
		Amount   Balance
	}
	type args struct {
		input []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *Transfer
		wantErr bool
	}{
		{
			name:    "OK",
			args:    args{input: blob},
			want:    tr,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Transfer{
				Sender:   tt.fields.Sender,
				Receiver: tt.fields.Receiver,
				Amount:   tt.fields.Amount,
			}
			if err := t.Decode(tt.args.input); (err != nil) != tt.wantErr {
				t1.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t1, tt.want, t)
		})
	}
}
