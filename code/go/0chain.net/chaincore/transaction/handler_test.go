package transaction

import (
	"0chain.net/chaincore/client"
	"0chain.net/chaincore/config"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	mocks "0chain.net/mocks/core/datastore"
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func init() {
	config.Configuration.DeploymentMode = config.DeploymentDevelopment
}

func TestGetTransaction(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	type args struct {
		ctx context.Context
		r   *http.Request
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			name:    "ERR",
			args:    args{r: r},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetTransaction(tt.args.ctx, tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTransaction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetTransaction() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPutTransaction(t *testing.T) {
	pbK, prK, err := encryption.GenerateKeys()
	if err != nil {
		t.Fatal(err)
	}
	chainID := "chain id"
	config.SetServerChainID(chainID)

	timeout := int64(5)
	SetTxnTimeout(timeout)

	ts := common.Now()

	client.SetClientSignatureScheme("ed25519")

	clStore := mocks.Store{}
	clStore.On("Read", context.TODO(), mock.AnythingOfType("string"),
		mock.AnythingOfType("*client.Client")).Return(
		func(_ context.Context, _ string, entity datastore.Entity) error {
			cl := entity.(*client.Client)
			*cl = client.Client{
				PublicKey: pbK,
			}
			return nil
		},
	)
	clStore.On("Read", context.Context(nil), mock.AnythingOfType("string"),
		mock.AnythingOfType("*client.Client")).Return(
		func(_ context.Context, _ string, entity datastore.Entity) error {
			return errors.New("")
		},
	)

	clientEntityMetadata := datastore.MetadataProvider()
	clientEntityMetadata.Store = &clStore
	client.SetEntityMetadata(clientEntityMetadata)

	txnStore := mocks.Store{}
	txnStore.On("Write", context.TODO(), mock.AnythingOfType("*transaction.Transaction")).Return(
		func(_ context.Context, entity datastore.Entity) error {
			txn := entity.(*Transaction)
			if txn.Value == 1 {
				return errors.New("")
			}
			return nil
		},
	)

	transactionEntityMetadata.Store = &txnStore

	validTxn := Transaction{
		Value:           2,
		ChainID:         chainID,
		ToClientID:      pbK,
		CreationDate:    ts,
		PublicKey:       pbK,
		Fee:             1,
		TransactionData: "debug",
	}

	validTxn.ComputeProperties()
	validTxn.Hash = validTxn.ComputeHash()

	sign, err := encryption.Sign(prK, validTxn.Hash)
	if err != nil {
		t.Fatal(err)
	}
	validTxn.Signature = sign

	validTxn.OutputHash = validTxn.ComputeOutputHash()

	type args struct {
		ctx    context.Context
		entity datastore.Entity
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			name:    "Not_A_Transaction_Entity_ERR",
			args:    args{entity: &TransactionSummary{}},
			wantErr: true,
		},
		{
			name:    "Invalid_Transaction_ERR",
			args:    args{entity: &Transaction{}},
			wantErr: true,
		},
		{
			name: "GetClient_ERR",
			args: args{
				entity: func() datastore.Entity {
					txn := Transaction{
						Value:           1,
						ChainID:         chainID,
						ToClientID:      pbK,
						CreationDate:    ts,
						PublicKey:       pbK,
						Fee:             1,
						TransactionData: "debug",
					}

					txn.ComputeProperties()
					txn.Hash = txn.ComputeHash()

					sign, err := encryption.Sign(prK, txn.Hash)
					if err != nil {
						t.Fatal(err)
					}
					txn.Signature = sign

					txn.OutputHash = txn.ComputeOutputHash()

					return &txn
				}(),
			},
			wantErr: true,
		},
		{
			name: "Txn_Write_ERR",
			args: args{
				ctx: context.TODO(),
				entity: func() datastore.Entity {
					txn := Transaction{
						Value:           1, // set for error calls on store mock
						ChainID:         chainID,
						ToClientID:      pbK,
						CreationDate:    ts,
						PublicKey:       pbK,
						Fee:             1,
						TransactionData: "debug",
					}

					txn.ComputeProperties()
					txn.Hash = txn.ComputeHash()

					sign, err := encryption.Sign(prK, txn.Hash)
					if err != nil {
						t.Fatal(err)
					}
					txn.Signature = sign

					txn.OutputHash = txn.ComputeOutputHash()

					return &txn
				}(),
			},
			wantErr: true,
		},
		{
			name: "OK",
			args: args{
				ctx:    context.TODO(),
				entity: &validTxn,
			},
			want:    &validTxn,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PutTransaction(tt.args.ctx, tt.args.entity)
			if (err != nil) != tt.wantErr {
				t.Errorf("PutTransaction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !assert.Equal(t, tt.want, got) {
				t.Errorf("PutTransaction() got = %v, want %v", got, tt.want)
			}
		})
	}
}
