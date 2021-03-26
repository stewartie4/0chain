package wallet

import (
	"0chain.net/chaincore/client"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/memorystore"
	mocks "0chain.net/mocks/core/datastore"
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"reflect"
	"testing"
)

func init() {
	transactionEntityMetadata := datastore.MetadataProvider()
	transactionEntityMetadata.Name = "txn"
	transactionEntityMetadata.DB = "txndb"
	transactionEntityMetadata.Provider = transaction.Provider
	transactionEntityMetadata.Store = memorystore.GetStorageProvider()
	datastore.RegisterEntityMetadata("txn", transactionEntityMetadata)

	clientEntityMetadata := datastore.MetadataProvider()
	clientEntityMetadata.Name = "client"
	clientEntityMetadata.Provider = client.Provider
	store := mocks.Store{}
	store.On("Write", context.TODO(), mock.AnythingOfType("*client.Client")).Return(
		func(_ context.Context, _ datastore.Entity) error {
			return nil
		},
	)
	clientEntityMetadata.Store = &store
	datastore.RegisterEntityMetadata("client", clientEntityMetadata)
	client.SetEntityMetadata(clientEntityMetadata)

	SetupWallet()
}

func TestWallet_CreateRandomSendTransaction(t *testing.T) {
	var (
		w                      = Wallet{}
		toClient               = "to client"
		value    state.Balance = 5
		fee      state.Balance = 2
	)
	if err := w.Initialize("ed25519"); err != nil {
		t.Fatal(err)
	}

	config.DevConfiguration.IsFeeEnabled = true

	txn := transactionMetadataProvider.Instance().(*transaction.Transaction)
	txn.ClientID = w.ClientID
	txn.ToClientID = toClient
	txn.Value = value
	txn.Fee = fee

	if _, err := txn.Sign(w.SignatureScheme); err != nil {
		t.Fatal(err)
	}

	type fields struct {
		SignatureScheme encryption.SignatureScheme
		PublicKeyBytes  []byte
		PublicKey       string
		ClientID        string
		Balance         state.Balance
	}
	type args struct {
		toClient string
		value    state.Balance
		fee      state.Balance
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *transaction.Transaction
	}{
		{
			name: "OK",
			fields: fields{
				SignatureScheme: w.SignatureScheme,
				PublicKeyBytes:  w.PublicKeyBytes,
				PublicKey:       w.PublicKey,
				ClientID:        w.ClientID,
			},
			args: args{
				toClient: toClient,
				value:    value,
				fee:      fee,
			},
			want: &transaction.Transaction{
				HashIDField:           txn.HashIDField,
				CollectionMemberField: txn.CollectionMemberField,
				VersionField:          txn.VersionField,
				ClientID:              txn.ClientID,
				PublicKey:             txn.PublicKey,
				ToClientID:            txn.ToClientID,
				ChainID:               txn.ChainID,
				TransactionData:       txn.TransactionData,
				Value:                 txn.Value,
				Signature:             txn.Signature,
				CreationDate:          txn.CreationDate,
				Fee:                   txn.Fee,
				TransactionType:       txn.TransactionType,
				TransactionOutput:     txn.TransactionOutput,
				OutputHash:            txn.OutputHash,
				Status:                txn.Status,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Wallet{
				SignatureScheme: tt.fields.SignatureScheme,
				PublicKeyBytes:  tt.fields.PublicKeyBytes,
				PublicKey:       tt.fields.PublicKey,
				ClientID:        tt.fields.ClientID,
				Balance:         tt.fields.Balance,
			}

			got := w.CreateRandomSendTransaction(tt.args.toClient, tt.args.value, tt.args.fee)
			got.TransactionData = ""
			tt.want.TransactionData = ""
			got.Hash = ""
			tt.want.Hash = ""
			got.Signature = ""
			tt.want.Signature = ""
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWallet_CreateSCTransaction(t *testing.T) {
	var (
		w                      = Wallet{}
		toClient               = "to client"
		msg                    = "msg"
		value    state.Balance = 5
		fee      state.Balance = 2
	)
	if err := w.Initialize("ed25519"); err != nil {
		t.Fatal(err)
	}

	config.DevConfiguration.IsFeeEnabled = true

	txn := transactionMetadataProvider.Instance().(*transaction.Transaction)
	txn.ClientID = w.ClientID
	txn.ToClientID = toClient
	txn.Value = value
	txn.TransactionData = msg
	txn.Fee = fee
	txn.TransactionType = transaction.TxnTypeSmartContract
	if _, err := txn.Sign(w.SignatureScheme); err != nil {
		t.Fatal(err)
	}

	type fields struct {
		SignatureScheme encryption.SignatureScheme
		PublicKeyBytes  []byte
		PublicKey       string
		ClientID        string
		Balance         state.Balance
	}
	type args struct {
		toClient string
		value    state.Balance
		msg      string
		fee      state.Balance
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *transaction.Transaction
	}{
		{
			name: "OK",
			fields: fields{
				SignatureScheme: w.SignatureScheme,
				PublicKeyBytes:  w.PublicKeyBytes,
				PublicKey:       w.PublicKey,
				ClientID:        w.ClientID,
			},
			args: args{
				toClient: toClient,
				msg:      msg,
				value:    value,
				fee:      fee,
			},
			want: &transaction.Transaction{
				HashIDField:           txn.HashIDField,
				CollectionMemberField: txn.CollectionMemberField,
				VersionField:          txn.VersionField,
				ClientID:              txn.ClientID,
				PublicKey:             txn.PublicKey,
				ToClientID:            txn.ToClientID,
				ChainID:               txn.ChainID,
				TransactionData:       txn.TransactionData,
				Value:                 txn.Value,
				Signature:             txn.Signature,
				CreationDate:          txn.CreationDate,
				Fee:                   txn.Fee,
				TransactionType:       txn.TransactionType,
				TransactionOutput:     txn.TransactionOutput,
				OutputHash:            txn.OutputHash,
				Status:                txn.Status,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Wallet{
				SignatureScheme: tt.fields.SignatureScheme,
				PublicKeyBytes:  tt.fields.PublicKeyBytes,
				PublicKey:       tt.fields.PublicKey,
				ClientID:        tt.fields.ClientID,
				Balance:         tt.fields.Balance,
			}
			if got := w.CreateSCTransaction(tt.args.toClient, tt.args.value, tt.args.msg, tt.args.fee); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateSCTransaction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWallet_CreateRandomDataTransaction(t *testing.T) {
	var (
		w                 = Wallet{}
		fee state.Balance = 2
	)
	if err := w.Initialize("ed25519"); err != nil {
		t.Fatal(err)
	}

	config.DevConfiguration.IsFeeEnabled = true

	txn := transactionMetadataProvider.Instance().(*transaction.Transaction)
	txn.ClientID = w.ClientID
	txn.Fee = fee
	txn.TransactionType = transaction.TxnTypeData

	if _, err := txn.Sign(w.SignatureScheme); err != nil {
		t.Fatal(err)
	}

	type fields struct {
		SignatureScheme encryption.SignatureScheme
		PublicKeyBytes  []byte
		PublicKey       string
		ClientID        string
		Balance         state.Balance
	}
	type args struct {
		fee state.Balance
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *transaction.Transaction
	}{
		{
			name: "OK",
			fields: fields{
				SignatureScheme: w.SignatureScheme,
				PublicKeyBytes:  w.PublicKeyBytes,
				PublicKey:       w.PublicKey,
				ClientID:        w.ClientID,
			},
			args: args{
				fee: fee,
			},
			want: &transaction.Transaction{
				HashIDField:           txn.HashIDField,
				CollectionMemberField: txn.CollectionMemberField,
				VersionField:          txn.VersionField,
				ClientID:              txn.ClientID,
				PublicKey:             txn.PublicKey,
				ToClientID:            txn.ToClientID,
				ChainID:               txn.ChainID,
				TransactionData:       txn.TransactionData,
				Value:                 txn.Value,
				Signature:             txn.Signature,
				CreationDate:          txn.CreationDate,
				Fee:                   txn.Fee,
				TransactionType:       txn.TransactionType,
				TransactionOutput:     txn.TransactionOutput,
				OutputHash:            txn.OutputHash,
				Status:                txn.Status,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Wallet{
				SignatureScheme: tt.fields.SignatureScheme,
				PublicKeyBytes:  tt.fields.PublicKeyBytes,
				PublicKey:       tt.fields.PublicKey,
				ClientID:        tt.fields.ClientID,
				Balance:         tt.fields.Balance,
			}
			got := w.CreateRandomDataTransaction(tt.args.fee)
			got.TransactionData = ""
			tt.want.TransactionData = ""
			got.Hash = ""
			tt.want.Hash = ""
			got.Signature = ""
			tt.want.Signature = ""
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWallet_CreateDataTransaction(t *testing.T) {
	var (
		w                 = Wallet{}
		msg               = "msg"
		fee state.Balance = 2
	)
	if err := w.Initialize("ed25519"); err != nil {
		t.Fatal(err)
	}

	config.DevConfiguration.IsFeeEnabled = true

	txn := transactionMetadataProvider.Instance().(*transaction.Transaction)
	txn.ClientID = w.ClientID
	txn.TransactionData = msg
	txn.Fee = fee
	txn.TransactionType = transaction.TxnTypeData

	if _, err := txn.Sign(w.SignatureScheme); err != nil {
		t.Fatal(err)
	}

	type fields struct {
		SignatureScheme encryption.SignatureScheme
		PublicKeyBytes  []byte
		PublicKey       string
		ClientID        string
		Balance         state.Balance
	}
	type args struct {
		msg string
		fee state.Balance
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *transaction.Transaction
	}{
		{
			name: "OK",
			fields: fields{
				SignatureScheme: w.SignatureScheme,
				PublicKeyBytes:  w.PublicKeyBytes,
				PublicKey:       w.PublicKey,
				ClientID:        w.ClientID,
			},
			args: args{
				msg: msg,
				fee: fee,
			},
			want: &transaction.Transaction{
				HashIDField:           txn.HashIDField,
				CollectionMemberField: txn.CollectionMemberField,
				VersionField:          txn.VersionField,
				ClientID:              txn.ClientID,
				PublicKey:             txn.PublicKey,
				ToClientID:            txn.ToClientID,
				ChainID:               txn.ChainID,
				TransactionData:       txn.TransactionData,
				Value:                 txn.Value,
				Signature:             txn.Signature,
				CreationDate:          txn.CreationDate,
				Fee:                   txn.Fee,
				TransactionType:       txn.TransactionType,
				TransactionOutput:     txn.TransactionOutput,
				OutputHash:            txn.OutputHash,
				Status:                txn.Status,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Wallet{
				SignatureScheme: tt.fields.SignatureScheme,
				PublicKeyBytes:  tt.fields.PublicKeyBytes,
				PublicKey:       tt.fields.PublicKey,
				ClientID:        tt.fields.ClientID,
				Balance:         tt.fields.Balance,
			}
			got := w.CreateDataTransaction(tt.args.msg, tt.args.fee)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWallet_Register(t *testing.T) {
	w := Wallet{}
	if err := w.Initialize("ed25519"); err != nil {
		t.Fatal(err)
	}

	type fields struct {
		SignatureScheme encryption.SignatureScheme
		PublicKeyBytes  []byte
		PublicKey       string
		ClientID        string
		Balance         state.Balance
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
			name: "OK",
			fields: fields{
				SignatureScheme: w.SignatureScheme,
				PublicKeyBytes:  w.PublicKeyBytes,
				PublicKey:       w.PublicKey,
				ClientID:        w.ClientID,
			},
			args:    args{ctx: context.TODO()},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Wallet{
				SignatureScheme: tt.fields.SignatureScheme,
				PublicKeyBytes:  tt.fields.PublicKeyBytes,
				PublicKey:       tt.fields.PublicKey,
				ClientID:        tt.fields.ClientID,
				Balance:         tt.fields.Balance,
			}
			if err := w.Register(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
