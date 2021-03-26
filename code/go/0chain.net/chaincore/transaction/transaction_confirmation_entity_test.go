package transaction

import (
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/memorystore"
	"0chain.net/core/util"
	mocks "0chain.net/mocks/core/datastore"
	"context"
	"reflect"
	"testing"
)

func init() {
	SetupTxnConfirmationEntity(memorystore.GetStorageProvider())
}

func TestConfirmation_GetKey(t *testing.T) {
	type fields struct {
		Version               string
		Hash                  string
		BlockHash             string
		PreviousBlockHash     string
		Transaction           *Transaction
		CreationDateField     datastore.CreationDateField
		MinerID               datastore.Key
		Round                 int64
		Status                int
		RoundRandomSeed       int64
		MerkleTreeRoot        string
		MerkleTreePath        *util.MTPath
		ReceiptMerkleTreeRoot string
		ReceiptMerkleTreePath *util.MTPath
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
		t.Run(tt.name, func(t *testing.T) {
			c := &Confirmation{
				Version:               tt.fields.Version,
				Hash:                  tt.fields.Hash,
				BlockHash:             tt.fields.BlockHash,
				PreviousBlockHash:     tt.fields.PreviousBlockHash,
				Transaction:           tt.fields.Transaction,
				CreationDateField:     tt.fields.CreationDateField,
				MinerID:               tt.fields.MinerID,
				Round:                 tt.fields.Round,
				Status:                tt.fields.Status,
				RoundRandomSeed:       tt.fields.RoundRandomSeed,
				MerkleTreeRoot:        tt.fields.MerkleTreeRoot,
				MerkleTreePath:        tt.fields.MerkleTreePath,
				ReceiptMerkleTreeRoot: tt.fields.ReceiptMerkleTreeRoot,
				ReceiptMerkleTreePath: tt.fields.ReceiptMerkleTreePath,
			}

			c.SetKey(tt.want)
			if got := c.GetKey(); got != tt.want {
				t.Errorf("GetKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfirmation_ComputeProperties(t *testing.T) {
	type fields struct {
		Version               string
		Hash                  string
		BlockHash             string
		PreviousBlockHash     string
		Transaction           *Transaction
		CreationDateField     datastore.CreationDateField
		MinerID               datastore.Key
		Round                 int64
		Status                int
		RoundRandomSeed       int64
		MerkleTreeRoot        string
		MerkleTreePath        *util.MTPath
		ReceiptMerkleTreeRoot string
		ReceiptMerkleTreePath *util.MTPath
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "OK", // not implemented
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Confirmation{
				Version:               tt.fields.Version,
				Hash:                  tt.fields.Hash,
				BlockHash:             tt.fields.BlockHash,
				PreviousBlockHash:     tt.fields.PreviousBlockHash,
				Transaction:           tt.fields.Transaction,
				CreationDateField:     tt.fields.CreationDateField,
				MinerID:               tt.fields.MinerID,
				Round:                 tt.fields.Round,
				Status:                tt.fields.Status,
				RoundRandomSeed:       tt.fields.RoundRandomSeed,
				MerkleTreeRoot:        tt.fields.MerkleTreeRoot,
				MerkleTreePath:        tt.fields.MerkleTreePath,
				ReceiptMerkleTreeRoot: tt.fields.ReceiptMerkleTreeRoot,
				ReceiptMerkleTreePath: tt.fields.ReceiptMerkleTreePath,
			}

			c.ComputeProperties()
		})
	}
}

func TestConfirmation_Validate(t *testing.T) {
	type fields struct {
		Version               string
		Hash                  string
		BlockHash             string
		PreviousBlockHash     string
		Transaction           *Transaction
		CreationDateField     datastore.CreationDateField
		MinerID               datastore.Key
		Round                 int64
		Status                int
		RoundRandomSeed       int64
		MerkleTreeRoot        string
		MerkleTreePath        *util.MTPath
		ReceiptMerkleTreeRoot string
		ReceiptMerkleTreePath *util.MTPath
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
			name:    "OK", // not implemented
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Confirmation{
				Version:               tt.fields.Version,
				Hash:                  tt.fields.Hash,
				BlockHash:             tt.fields.BlockHash,
				PreviousBlockHash:     tt.fields.PreviousBlockHash,
				Transaction:           tt.fields.Transaction,
				CreationDateField:     tt.fields.CreationDateField,
				MinerID:               tt.fields.MinerID,
				Round:                 tt.fields.Round,
				Status:                tt.fields.Status,
				RoundRandomSeed:       tt.fields.RoundRandomSeed,
				MerkleTreeRoot:        tt.fields.MerkleTreeRoot,
				MerkleTreePath:        tt.fields.MerkleTreePath,
				ReceiptMerkleTreeRoot: tt.fields.ReceiptMerkleTreeRoot,
				ReceiptMerkleTreePath: tt.fields.ReceiptMerkleTreePath,
			}
			if err := c.Validate(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfirmation_Read(t *testing.T) {
	store := mocks.Store{}
	store.On("Read", context.Context(nil), "", new(Confirmation)).Return(
		func(_ context.Context, _ string, _ datastore.Entity) error {
			return nil
		},
	)

	transactionConfirmationEntityMetadata.Store = &store

	type fields struct {
		Version               string
		Hash                  string
		BlockHash             string
		PreviousBlockHash     string
		Transaction           *Transaction
		CreationDateField     datastore.CreationDateField
		MinerID               datastore.Key
		Round                 int64
		Status                int
		RoundRandomSeed       int64
		MerkleTreeRoot        string
		MerkleTreePath        *util.MTPath
		ReceiptMerkleTreeRoot string
		ReceiptMerkleTreePath *util.MTPath
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
		t.Run(tt.name, func(t *testing.T) {
			c := &Confirmation{
				Version:               tt.fields.Version,
				Hash:                  tt.fields.Hash,
				BlockHash:             tt.fields.BlockHash,
				PreviousBlockHash:     tt.fields.PreviousBlockHash,
				Transaction:           tt.fields.Transaction,
				CreationDateField:     tt.fields.CreationDateField,
				MinerID:               tt.fields.MinerID,
				Round:                 tt.fields.Round,
				Status:                tt.fields.Status,
				RoundRandomSeed:       tt.fields.RoundRandomSeed,
				MerkleTreeRoot:        tt.fields.MerkleTreeRoot,
				MerkleTreePath:        tt.fields.MerkleTreePath,
				ReceiptMerkleTreeRoot: tt.fields.ReceiptMerkleTreeRoot,
				ReceiptMerkleTreePath: tt.fields.ReceiptMerkleTreePath,
			}
			if err := c.Read(tt.args.ctx, tt.args.key); (err != nil) != tt.wantErr {
				t.Errorf("Read() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfirmation_Write(t *testing.T) {
	store := mocks.Store{}
	store.On("Write", context.Context(nil), new(Confirmation)).Return(
		func(_ context.Context, _ datastore.Entity) error {
			return nil
		},
	)

	transactionConfirmationEntityMetadata.Store = &store

	type fields struct {
		Version               string
		Hash                  string
		BlockHash             string
		PreviousBlockHash     string
		Transaction           *Transaction
		CreationDateField     datastore.CreationDateField
		MinerID               datastore.Key
		Round                 int64
		Status                int
		RoundRandomSeed       int64
		MerkleTreeRoot        string
		MerkleTreePath        *util.MTPath
		ReceiptMerkleTreeRoot string
		ReceiptMerkleTreePath *util.MTPath
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
		t.Run(tt.name, func(t *testing.T) {
			c := &Confirmation{
				Version:               tt.fields.Version,
				Hash:                  tt.fields.Hash,
				BlockHash:             tt.fields.BlockHash,
				PreviousBlockHash:     tt.fields.PreviousBlockHash,
				Transaction:           tt.fields.Transaction,
				CreationDateField:     tt.fields.CreationDateField,
				MinerID:               tt.fields.MinerID,
				Round:                 tt.fields.Round,
				Status:                tt.fields.Status,
				RoundRandomSeed:       tt.fields.RoundRandomSeed,
				MerkleTreeRoot:        tt.fields.MerkleTreeRoot,
				MerkleTreePath:        tt.fields.MerkleTreePath,
				ReceiptMerkleTreeRoot: tt.fields.ReceiptMerkleTreeRoot,
				ReceiptMerkleTreePath: tt.fields.ReceiptMerkleTreePath,
			}
			if err := c.Write(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Write() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfirmation_Delete(t *testing.T) {
	store := mocks.Store{}
	store.On("Delete", context.Context(nil), new(Confirmation)).Return(
		func(_ context.Context, _ datastore.Entity) error {
			return nil
		},
	)

	transactionConfirmationEntityMetadata.Store = &store

	type fields struct {
		Version               string
		Hash                  string
		BlockHash             string
		PreviousBlockHash     string
		Transaction           *Transaction
		CreationDateField     datastore.CreationDateField
		MinerID               datastore.Key
		Round                 int64
		Status                int
		RoundRandomSeed       int64
		MerkleTreeRoot        string
		MerkleTreePath        *util.MTPath
		ReceiptMerkleTreeRoot string
		ReceiptMerkleTreePath *util.MTPath
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
		t.Run(tt.name, func(t *testing.T) {
			c := &Confirmation{
				Version:               tt.fields.Version,
				Hash:                  tt.fields.Hash,
				BlockHash:             tt.fields.BlockHash,
				PreviousBlockHash:     tt.fields.PreviousBlockHash,
				Transaction:           tt.fields.Transaction,
				CreationDateField:     tt.fields.CreationDateField,
				MinerID:               tt.fields.MinerID,
				Round:                 tt.fields.Round,
				Status:                tt.fields.Status,
				RoundRandomSeed:       tt.fields.RoundRandomSeed,
				MerkleTreeRoot:        tt.fields.MerkleTreeRoot,
				MerkleTreePath:        tt.fields.MerkleTreePath,
				ReceiptMerkleTreeRoot: tt.fields.ReceiptMerkleTreeRoot,
				ReceiptMerkleTreePath: tt.fields.ReceiptMerkleTreePath,
			}
			if err := c.Delete(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfirmation_GetScore(t *testing.T) {
	num := int64(5)

	type fields struct {
		Version               string
		Hash                  string
		BlockHash             string
		PreviousBlockHash     string
		Transaction           *Transaction
		CreationDateField     datastore.CreationDateField
		MinerID               datastore.Key
		Round                 int64
		Status                int
		RoundRandomSeed       int64
		MerkleTreeRoot        string
		MerkleTreePath        *util.MTPath
		ReceiptMerkleTreeRoot string
		ReceiptMerkleTreePath *util.MTPath
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
		t.Run(tt.name, func(t *testing.T) {
			c := &Confirmation{
				Version:               tt.fields.Version,
				Hash:                  tt.fields.Hash,
				BlockHash:             tt.fields.BlockHash,
				PreviousBlockHash:     tt.fields.PreviousBlockHash,
				Transaction:           tt.fields.Transaction,
				CreationDateField:     tt.fields.CreationDateField,
				MinerID:               tt.fields.MinerID,
				Round:                 tt.fields.Round,
				Status:                tt.fields.Status,
				RoundRandomSeed:       tt.fields.RoundRandomSeed,
				MerkleTreeRoot:        tt.fields.MerkleTreeRoot,
				MerkleTreePath:        tt.fields.MerkleTreePath,
				ReceiptMerkleTreeRoot: tt.fields.ReceiptMerkleTreeRoot,
				ReceiptMerkleTreePath: tt.fields.ReceiptMerkleTreePath,
			}
			if got := c.GetScore(); got != tt.want {
				t.Errorf("GetScore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfirmation_GetHash(t *testing.T) {
	hash := encryption.Hash("data")

	type fields struct {
		Version               string
		Hash                  string
		BlockHash             string
		PreviousBlockHash     string
		Transaction           *Transaction
		CreationDateField     datastore.CreationDateField
		MinerID               datastore.Key
		Round                 int64
		Status                int
		RoundRandomSeed       int64
		MerkleTreeRoot        string
		MerkleTreePath        *util.MTPath
		ReceiptMerkleTreeRoot string
		ReceiptMerkleTreePath *util.MTPath
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name:   "OK",
			fields: fields{Hash: hash},
			want:   hash,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Confirmation{
				Version:               tt.fields.Version,
				Hash:                  tt.fields.Hash,
				BlockHash:             tt.fields.BlockHash,
				PreviousBlockHash:     tt.fields.PreviousBlockHash,
				Transaction:           tt.fields.Transaction,
				CreationDateField:     tt.fields.CreationDateField,
				MinerID:               tt.fields.MinerID,
				Round:                 tt.fields.Round,
				Status:                tt.fields.Status,
				RoundRandomSeed:       tt.fields.RoundRandomSeed,
				MerkleTreeRoot:        tt.fields.MerkleTreeRoot,
				MerkleTreePath:        tt.fields.MerkleTreePath,
				ReceiptMerkleTreeRoot: tt.fields.ReceiptMerkleTreeRoot,
				ReceiptMerkleTreePath: tt.fields.ReceiptMerkleTreePath,
			}
			if got := c.GetHash(); got != tt.want {
				t.Errorf("GetHash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfirmation_GetHashBytes(t *testing.T) {
	hash := encryption.Hash("data")

	type fields struct {
		Version               string
		Hash                  string
		BlockHash             string
		PreviousBlockHash     string
		Transaction           *Transaction
		CreationDateField     datastore.CreationDateField
		MinerID               datastore.Key
		Round                 int64
		Status                int
		RoundRandomSeed       int64
		MerkleTreeRoot        string
		MerkleTreePath        *util.MTPath
		ReceiptMerkleTreeRoot string
		ReceiptMerkleTreePath *util.MTPath
	}
	tests := []struct {
		name   string
		fields fields
		want   []byte
	}{
		{
			name:   "OK",
			fields: fields{Hash: hash},
			want:   util.HashStringToBytes(hash),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Confirmation{
				Version:               tt.fields.Version,
				Hash:                  tt.fields.Hash,
				BlockHash:             tt.fields.BlockHash,
				PreviousBlockHash:     tt.fields.PreviousBlockHash,
				Transaction:           tt.fields.Transaction,
				CreationDateField:     tt.fields.CreationDateField,
				MinerID:               tt.fields.MinerID,
				Round:                 tt.fields.Round,
				Status:                tt.fields.Status,
				RoundRandomSeed:       tt.fields.RoundRandomSeed,
				MerkleTreeRoot:        tt.fields.MerkleTreeRoot,
				MerkleTreePath:        tt.fields.MerkleTreePath,
				ReceiptMerkleTreeRoot: tt.fields.ReceiptMerkleTreeRoot,
				ReceiptMerkleTreePath: tt.fields.ReceiptMerkleTreePath,
			}
			if got := c.GetHashBytes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetHashBytes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTransactionConfirmationProvider(t *testing.T) {
	tests := []struct {
		name string
		want datastore.Entity
	}{
		{
			name: "OK",
			want: &Confirmation{
				Version: "1.0",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TransactionConfirmationProvider(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TransactionConfirmationProvider() = %v, want %v", got, tt.want)
			}
		})
	}
}
