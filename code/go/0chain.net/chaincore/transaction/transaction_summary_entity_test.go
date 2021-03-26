package transaction

import (
	"0chain.net/core/datastore"
	mocks "0chain.net/mocks/core/datastore"
	"context"
	"reflect"
	"testing"
)

func TestTransactionSummary_GetEntityMetadata(t1 *testing.T) {
	type fields struct {
		HashIDField datastore.HashIDField
		Round       int64
	}
	tests := []struct {
		name   string
		fields fields
		want   datastore.EntityMetadata
	}{
		{
			name: "OK",
			want: transactionSummaryEntityMetadata,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &TransactionSummary{
				HashIDField: tt.fields.HashIDField,
				Round:       tt.fields.Round,
			}
			if got := t.GetEntityMetadata(); !reflect.DeepEqual(got, tt.want) {
				t1.Errorf("GetEntityMetadata() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTransactionSummary_GetKey(t1 *testing.T) {
	type fields struct {
		HashIDField datastore.HashIDField
		Round       int64
	}
	tests := []struct {
		name   string
		fields fields
		want   datastore.Key
	}{
		{
			name: "OK",
			want: "key",
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &TransactionSummary{
				HashIDField: tt.fields.HashIDField,
				Round:       tt.fields.Round,
			}

			t.SetKey(tt.want)
			if got := t.GetKey(); got != tt.want {
				t1.Errorf("GetKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTransactionSummary_Read(t1 *testing.T) {
	store := mocks.Store{}
	store.On("Read", context.Context(nil), "", new(TransactionSummary)).Return(
		func(_ context.Context, _ string, _ datastore.Entity) error {
			return nil
		},
	)

	transactionSummaryEntityMetadata.Store = &store

	type fields struct {
		HashIDField datastore.HashIDField
		Round       int64
	}
	type args struct {
		ctx context.Context
		key datastore.Key
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "OK",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &TransactionSummary{
				HashIDField: tt.fields.HashIDField,
				Round:       tt.fields.Round,
			}
			if err := t.Read(tt.args.ctx, tt.args.key); (err != nil) != tt.wantErr {
				t1.Errorf("Read() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTransactionSummary_Write(t1 *testing.T) {
	store := mocks.Store{}
	store.On("Write", context.Context(nil), new(TransactionSummary)).Return(
		func(_ context.Context, _ datastore.Entity) error {
			return nil
		},
	)

	transactionSummaryEntityMetadata.Store = &store

	type fields struct {
		HashIDField datastore.HashIDField
		Round       int64
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "OK",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &TransactionSummary{
				HashIDField: tt.fields.HashIDField,
				Round:       tt.fields.Round,
			}
			if err := t.Write(tt.args.ctx); (err != nil) != tt.wantErr {
				t1.Errorf("Write() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTransactionSummary_Delete(t1 *testing.T) {
	store := mocks.Store{}
	store.On("Delete", context.Context(nil), new(TransactionSummary)).Return(
		func(_ context.Context, _ datastore.Entity) error {
			return nil
		},
	)

	transactionSummaryEntityMetadata.Store = &store

	type fields struct {
		HashIDField datastore.HashIDField
		Round       int64
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "OK",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &TransactionSummary{
				HashIDField: tt.fields.HashIDField,
				Round:       tt.fields.Round,
			}
			if err := t.Delete(tt.args.ctx); (err != nil) != tt.wantErr {
				t1.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTransactionSummary_GetScore(t1 *testing.T) {
	num := int64(5)

	type fields struct {
		HashIDField datastore.HashIDField
		Round       int64
	}
	tests := []struct {
		name   string
		fields fields
		want   int64
	}{
		{
			name:   "OK",
			fields: fields{Round: num},
			want:   num,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &TransactionSummary{
				HashIDField: tt.fields.HashIDField,
				Round:       tt.fields.Round,
			}
			if got := t.GetScore(); got != tt.want {
				t1.Errorf("GetScore() = %v, want %v", got, tt.want)
			}
		})
	}
}
