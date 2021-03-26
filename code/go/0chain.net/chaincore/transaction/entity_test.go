package transaction

import (
	"0chain.net/chaincore/client"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/logging"
	"0chain.net/core/memorystore"
	"0chain.net/core/util"
	mocks "0chain.net/mocks/core/datastore"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/stretchr/testify/mock"
	"reflect"
	"strconv"
	"testing"
)

func init() {
	logging.InitLogging("development")

	clientEntityMetadata := datastore.MetadataProvider()
	clientEntityMetadata.Name = "client"
	clientEntityMetadata.Provider = client.Provider
	clientEntityMetadata.Store = memorystore.GetStorageProvider()
	datastore.RegisterEntityMetadata("client", clientEntityMetadata)

	transactionEntityMetadata = datastore.MetadataProvider()
	transactionEntityMetadata.Name = "txn"
	transactionEntityMetadata.DB = "txndb"
	transactionEntityMetadata.Provider = Provider
	transactionEntityMetadata.Store = memorystore.GetStorageProvider()
	datastore.RegisterEntityMetadata("txn", transactionEntityMetadata)

	SetupTxnSummaryEntity(memorystore.GetStorageProvider())

	SetupTransactionDB()
}

func makeTestTxn() Transaction {
	return Transaction{
		CreationDate:    common.Now(),
		ClientID:        "client id",
		ToClientID:      "to client id",
		Value:           123,
		TransactionData: "data",
	}
}

func TestTransaction_ValidateFee(t *testing.T) {
	minFee := state.Balance(5)
	SetTxnFee(minFee)

	type (
		fields struct {
			HashIDField           datastore.HashIDField
			CollectionMemberField datastore.CollectionMemberField
			VersionField          datastore.VersionField
			ClientID              datastore.Key
			PublicKey             string
			ToClientID            datastore.Key
			ChainID               datastore.Key
			TransactionData       string
			Value                 state.Balance
			Signature             string
			CreationDate          common.Timestamp
			Fee                   state.Balance
			TransactionType       int
			TransactionOutput     string
			OutputHash            string
			Status                int
		}
		test struct {
			name    string
			fields  fields
			wantErr bool
		}
	)

	tests := []test{
		{
			name: "Fee_Lower_Than_Min_ERR",
			fields: fields{
				Fee: minFee - 1,
				TransactionData: func() string {
					sctd := smartContractTransactionData{
						FunctionName: "unknown name",
						InputData: func() json.RawMessage {
							data := map[string]string{
								"key": "value",
							}
							blob, err := json.Marshal(data)
							if err != nil {
								t.Fatal(err)
							}

							return blob
						}(),
					}

					blob, err := json.Marshal(sctd)
					if err != nil {
						t.Error(err)
					}

					return string(blob)
				}(),
			},
			wantErr: true,
		},
		{
			name: "OK",
			fields: fields{
				Fee:             minFee + 1,
				TransactionData: "",
			},
			wantErr: false,
		},
	}

	for fName, ok := range exemptedSCFunctions {
		if !ok {
			continue
		}

		sctd := smartContractTransactionData{
			FunctionName: fName,
			InputData: func() json.RawMessage {
				data := map[string]string{
					"key": "value",
				}
				blob, err := json.Marshal(data)
				if err != nil {
					t.Fatal(err)
				}

				return blob
			}(),
		}

		blob, err := json.Marshal(sctd)
		if err != nil {
			t.Error(err)
		}

		test := test{
			name: fName + "_OK",
			fields: fields{
				TransactionData: string(blob),
				Fee:             minFee + 1,
			},
			wantErr: false,
		}
		tests = append(tests, test)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t1 *testing.T) {
			txn := &Transaction{
				HashIDField:           tt.fields.HashIDField,
				CollectionMemberField: tt.fields.CollectionMemberField,
				VersionField:          tt.fields.VersionField,
				ClientID:              tt.fields.ClientID,
				PublicKey:             tt.fields.PublicKey,
				ToClientID:            tt.fields.ToClientID,
				ChainID:               tt.fields.ChainID,
				TransactionData:       tt.fields.TransactionData,
				Value:                 tt.fields.Value,
				Signature:             tt.fields.Signature,
				CreationDate:          tt.fields.CreationDate,
				Fee:                   tt.fields.Fee,
				TransactionType:       tt.fields.TransactionType,
				TransactionOutput:     tt.fields.TransactionOutput,
				OutputHash:            tt.fields.OutputHash,
				Status:                tt.fields.Status,
			}
			if err := txn.ValidateFee(); (err != nil) != tt.wantErr {
				t1.Errorf("ValidateFee() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTransaction_ComputeClientID(t1 *testing.T) {
	pbK, _, err := encryption.GenerateKeys()
	if err != nil {
		t1.Fatal(err)
	}
	pbKByt, err := hex.DecodeString(pbK)
	if err != nil {
		t1.Fatal(err)
	}

	type fields struct {
		HashIDField           datastore.HashIDField
		CollectionMemberField datastore.CollectionMemberField
		VersionField          datastore.VersionField
		ClientID              datastore.Key
		PublicKey             string
		ToClientID            datastore.Key
		ChainID               datastore.Key
		TransactionData       string
		Value                 state.Balance
		Signature             string
		CreationDate          common.Timestamp
		Fee                   state.Balance
		TransactionType       int
		TransactionOutput     string
		OutputHash            string
		Status                int
	}
	tests := []struct {
		name   string
		fields fields
		wantID datastore.Key
	}{
		{
			name:   "OK",
			fields: fields{PublicKey: pbK},
			wantID: encryption.Hash(pbKByt),
		},
		{
			name:   "OK",
			wantID: "",
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Transaction{
				HashIDField:           tt.fields.HashIDField,
				CollectionMemberField: tt.fields.CollectionMemberField,
				VersionField:          tt.fields.VersionField,
				ClientID:              tt.fields.ClientID,
				PublicKey:             tt.fields.PublicKey,
				ToClientID:            tt.fields.ToClientID,
				ChainID:               tt.fields.ChainID,
				TransactionData:       tt.fields.TransactionData,
				Value:                 tt.fields.Value,
				Signature:             tt.fields.Signature,
				CreationDate:          tt.fields.CreationDate,
				Fee:                   tt.fields.Fee,
				TransactionType:       tt.fields.TransactionType,
				TransactionOutput:     tt.fields.TransactionOutput,
				OutputHash:            tt.fields.OutputHash,
				Status:                tt.fields.Status,
			}

			t.ComputeClientID()

			if !reflect.DeepEqual(t.ClientID, tt.wantID) {
				t1.Errorf("ComputeClientID() got = %v, want = %v", t.ClientID, tt.wantID)
			}
		})
	}
}

func TestTransaction_ValidateWrtTimeForBlock(t1 *testing.T) {
	pbK, prK, err := encryption.GenerateKeys()
	if err != nil {
		t1.Fatal(err)
	}
	chainID := "chain id"
	config.SetServerChainID(chainID)

	timeout := int64(5)
	SetTxnTimeout(timeout)

	ts := common.Now()

	client.SetClientSignatureScheme("ed25519")

	type fields struct {
		HashIDField           datastore.HashIDField
		CollectionMemberField datastore.CollectionMemberField
		VersionField          datastore.VersionField
		ClientID              datastore.Key
		PublicKey             string
		ToClientID            datastore.Key
		ChainID               datastore.Key
		TransactionData       string
		Value                 state.Balance
		Signature             string
		CreationDate          common.Timestamp
		Fee                   state.Balance
		TransactionType       int
		TransactionOutput     string
		OutputHash            string
		Status                int
	}
	type args struct {
		ctx               context.Context
		ts                common.Timestamp
		validateSignature bool
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		wantErr   bool
		enableFee bool
	}{
		{
			name:    "Negative_Value_ERR",
			fields:  fields{Value: -1},
			wantErr: true,
		},
		{
			name: "To_Client_ID_Is_Not_A_Hash_ERR",
			fields: fields{
				Value:      1,
				ToClientID: "!",
			},
			wantErr: true,
		},
		{
			name: "Incorrect_Fee_ERR",
			fields: fields{
				Value:      1,
				ToClientID: pbK,
				Fee:        -1,
			},
			enableFee: true,
			wantErr:   true,
		},
		{
			name: "Invalid_Chain_ERR",
			fields: fields{
				Value:      1,
				ToClientID: "",
				Fee:        1,
			},
			enableFee: true,
			wantErr:   true,
		},
		{
			name: "Empty_Hash_ERR",
			fields: fields{
				Value:      1,
				ChainID:    chainID,
				ToClientID: pbK,
				HashIDField: datastore.HashIDField{
					Hash: "",
				},
				Fee: 1,
			},
			enableFee: true,
			wantErr:   true,
		},
		{
			name: "Tolerance_Fail_Check_ERR",
			fields: func() fields {
				txn := Transaction{
					Value:        1,
					ChainID:      chainID,
					ToClientID:   pbK,
					CreationDate: ts - common.Timestamp(timeout) - 1,
					Fee:          1,
				}
				txn.Hash = txn.ComputeHash()

				return fields{
					Value:        txn.Value,
					ChainID:      txn.ChainID,
					ToClientID:   txn.ToClientID,
					CreationDate: txn.CreationDate,
					HashIDField: datastore.HashIDField{
						Hash: txn.Hash,
					},
					Fee: txn.Fee,
				}
			}(),
			args:      args{ts: ts},
			enableFee: true,
			wantErr:   true,
		},
		{
			name: "ClientID_Equal_ToClientID_ERR",
			fields: func() fields {
				txn := Transaction{
					Value:        1,
					ChainID:      chainID,
					ClientID:     pbK,
					ToClientID:   pbK,
					CreationDate: ts,
					Fee:          1,
				}
				txn.Hash = txn.ComputeHash()

				return fields{
					Value:        txn.Value,
					ChainID:      txn.ChainID,
					ClientID:     txn.ClientID,
					ToClientID:   txn.ToClientID,
					CreationDate: txn.CreationDate,
					HashIDField: datastore.HashIDField{
						Hash: txn.Hash,
					},
					Fee: txn.Fee,
				}
			}(),
			args:      args{ts: ts},
			enableFee: true,
			wantErr:   true,
		},
		{
			name: "Hash_Verifying_ERR",
			fields: func() fields {
				txn := Transaction{
					Value:        1,
					ChainID:      chainID,
					ToClientID:   pbK,
					CreationDate: ts,
					Fee:          1,
				}
				txn.Hash = "hash"

				return fields{
					Value:        txn.Value,
					ChainID:      txn.ChainID,
					ToClientID:   txn.ToClientID,
					CreationDate: txn.CreationDate,
					HashIDField: datastore.HashIDField{
						Hash: txn.Hash,
					},
					Fee: txn.Fee,
				}
			}(),
			args:      args{ts: ts},
			enableFee: true,
			wantErr:   true,
		},
		{
			name: "Signature_Verifying_ERR",
			fields: func() fields {
				txn := Transaction{
					Value:        1,
					ChainID:      chainID,
					ToClientID:   pbK,
					CreationDate: ts,
					PublicKey:    pbK,
					Fee:          1,
				}
				txn.Hash = txn.ComputeHash()

				return fields{
					Value:        txn.Value,
					ChainID:      txn.ChainID,
					ToClientID:   txn.ToClientID,
					CreationDate: txn.CreationDate,
					HashIDField: datastore.HashIDField{
						Hash: txn.Hash,
					},
					PublicKey: txn.PublicKey,
					Fee:       txn.Fee,
				}
			}(),
			args:      args{ts: ts, validateSignature: true},
			enableFee: true,
			wantErr:   true,
		},
		{
			name: "OutputHash_Verifying_ERR",
			fields: func() fields {
				txn := Transaction{
					Value:        1,
					ChainID:      chainID,
					ToClientID:   pbK,
					CreationDate: ts,
					PublicKey:    pbK,
					Fee:          1,
				}
				txn.Hash = txn.ComputeHash()

				sign, err := encryption.Sign(prK, txn.Hash)
				if err != nil {
					t1.Fatal(err)
				}
				txn.Signature = sign

				txn.OutputHash = "wrong output hash"

				return fields{
					Value:        txn.Value,
					ChainID:      txn.ChainID,
					ToClientID:   txn.ToClientID,
					CreationDate: txn.CreationDate,
					HashIDField: datastore.HashIDField{
						Hash: txn.Hash,
					},
					PublicKey:  txn.PublicKey,
					Signature:  txn.Signature,
					OutputHash: txn.OutputHash,
					Fee:        txn.Fee,
				}
			}(),
			args:      args{ts: ts, validateSignature: true},
			enableFee: true,
			wantErr:   true,
		},
		{
			name: "OK",
			fields: func() fields {
				txn := Transaction{
					Value:        1,
					ChainID:      chainID,
					ToClientID:   pbK,
					CreationDate: ts,
					PublicKey:    pbK,
					Fee:          1,
				}
				txn.Hash = txn.ComputeHash()

				sign, err := encryption.Sign(prK, txn.Hash)
				if err != nil {
					t1.Fatal(err)
				}
				txn.Signature = sign

				txn.OutputHash = txn.ComputeOutputHash()

				return fields{
					Value:        txn.Value,
					ChainID:      txn.ChainID,
					ToClientID:   txn.ToClientID,
					CreationDate: txn.CreationDate,
					HashIDField: datastore.HashIDField{
						Hash: txn.Hash,
					},
					PublicKey:  txn.PublicKey,
					Signature:  txn.Signature,
					OutputHash: txn.OutputHash,
					Fee:        txn.Fee,
				}
			}(),
			args:      args{ts: ts, validateSignature: true},
			enableFee: true,
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Transaction{
				HashIDField:           tt.fields.HashIDField,
				CollectionMemberField: tt.fields.CollectionMemberField,
				VersionField:          tt.fields.VersionField,
				ClientID:              tt.fields.ClientID,
				PublicKey:             tt.fields.PublicKey,
				ToClientID:            tt.fields.ToClientID,
				ChainID:               tt.fields.ChainID,
				TransactionData:       tt.fields.TransactionData,
				Value:                 tt.fields.Value,
				Signature:             tt.fields.Signature,
				CreationDate:          tt.fields.CreationDate,
				Fee:                   tt.fields.Fee,
				TransactionType:       tt.fields.TransactionType,
				TransactionOutput:     tt.fields.TransactionOutput,
				OutputHash:            tt.fields.OutputHash,
				Status:                tt.fields.Status,
			}
			config.DevConfiguration.IsFeeEnabled = tt.enableFee

			if err := t.ValidateWrtTimeForBlock(tt.args.ctx, tt.args.ts, tt.args.validateSignature); (err != nil) != tt.wantErr {
				t1.Errorf("ValidateWrtTimeForBlock() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTransaction_ValidateWrtTime(t1 *testing.T) {
	pbK, prK, err := encryption.GenerateKeys()
	if err != nil {
		t1.Fatal(err)
	}
	chainID := "chain id"
	config.SetServerChainID(chainID)

	timeout := int64(5)
	SetTxnTimeout(timeout)

	ts := common.Now()

	client.SetClientSignatureScheme("ed25519")

	type fields struct {
		HashIDField           datastore.HashIDField
		CollectionMemberField datastore.CollectionMemberField
		VersionField          datastore.VersionField
		ClientID              datastore.Key
		PublicKey             string
		ToClientID            datastore.Key
		ChainID               datastore.Key
		TransactionData       string
		Value                 state.Balance
		Signature             string
		CreationDate          common.Timestamp
		Fee                   state.Balance
		TransactionType       int
		TransactionOutput     string
		OutputHash            string
		Status                int
	}
	type args struct {
		ctx context.Context
		ts  common.Timestamp
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		wantErr   bool
		enableFee bool
	}{
		{
			name: "OK",
			fields: func() fields {
				txn := Transaction{
					Value:        1,
					ChainID:      chainID,
					ToClientID:   pbK,
					CreationDate: ts,
					PublicKey:    pbK,
					Fee:          1,
				}
				txn.Hash = txn.ComputeHash()

				sign, err := encryption.Sign(prK, txn.Hash)
				if err != nil {
					t1.Fatal(err)
				}
				txn.Signature = sign

				txn.OutputHash = txn.ComputeOutputHash()

				return fields{
					Value:        txn.Value,
					ChainID:      txn.ChainID,
					ToClientID:   txn.ToClientID,
					CreationDate: txn.CreationDate,
					HashIDField: datastore.HashIDField{
						Hash: txn.Hash,
					},
					PublicKey:  txn.PublicKey,
					Signature:  txn.Signature,
					OutputHash: txn.OutputHash,
					Fee:        txn.Fee,
				}
			}(),
			args:      args{ts: ts},
			enableFee: true,
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Transaction{
				HashIDField:           tt.fields.HashIDField,
				CollectionMemberField: tt.fields.CollectionMemberField,
				VersionField:          tt.fields.VersionField,
				ClientID:              tt.fields.ClientID,
				PublicKey:             tt.fields.PublicKey,
				ToClientID:            tt.fields.ToClientID,
				ChainID:               tt.fields.ChainID,
				TransactionData:       tt.fields.TransactionData,
				Value:                 tt.fields.Value,
				Signature:             tt.fields.Signature,
				CreationDate:          tt.fields.CreationDate,
				Fee:                   tt.fields.Fee,
				TransactionType:       tt.fields.TransactionType,
				TransactionOutput:     tt.fields.TransactionOutput,
				OutputHash:            tt.fields.OutputHash,
				Status:                tt.fields.Status,
			}
			config.DevConfiguration.IsFeeEnabled = tt.enableFee

			if err := t.ValidateWrtTime(tt.args.ctx, tt.args.ts); (err != nil) != tt.wantErr {
				t1.Errorf("ValidateWrtTime() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTransaction_Validate(t1 *testing.T) {
	pbK, prK, err := encryption.GenerateKeys()
	if err != nil {
		t1.Fatal(err)
	}
	chainID := "chain id"
	config.SetServerChainID(chainID)

	timeout := int64(5)
	SetTxnTimeout(timeout)

	ts := common.Now()

	client.SetClientSignatureScheme("ed25519")

	type fields struct {
		HashIDField           datastore.HashIDField
		CollectionMemberField datastore.CollectionMemberField
		VersionField          datastore.VersionField
		ClientID              datastore.Key
		PublicKey             string
		ToClientID            datastore.Key
		ChainID               datastore.Key
		TransactionData       string
		Value                 state.Balance
		Signature             string
		CreationDate          common.Timestamp
		Fee                   state.Balance
		TransactionType       int
		TransactionOutput     string
		OutputHash            string
		Status                int
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		enableFee bool
		wantErr   bool
	}{
		{
			name: "OK",
			fields: func() fields {
				txn := Transaction{
					Value:        1,
					ChainID:      chainID,
					ToClientID:   pbK,
					CreationDate: ts,
					PublicKey:    pbK,
					Fee:          1,
				}
				txn.Hash = txn.ComputeHash()

				sign, err := encryption.Sign(prK, txn.Hash)
				if err != nil {
					t1.Fatal(err)
				}
				txn.Signature = sign

				txn.OutputHash = txn.ComputeOutputHash()

				return fields{
					Value:        txn.Value,
					ChainID:      txn.ChainID,
					ToClientID:   txn.ToClientID,
					CreationDate: txn.CreationDate,
					HashIDField: datastore.HashIDField{
						Hash: txn.Hash,
					},
					PublicKey:  txn.PublicKey,
					Signature:  txn.Signature,
					OutputHash: txn.OutputHash,
					Fee:        txn.Fee,
				}
			}(),
			enableFee: true,
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Transaction{
				HashIDField:           tt.fields.HashIDField,
				CollectionMemberField: tt.fields.CollectionMemberField,
				VersionField:          tt.fields.VersionField,
				ClientID:              tt.fields.ClientID,
				PublicKey:             tt.fields.PublicKey,
				ToClientID:            tt.fields.ToClientID,
				ChainID:               tt.fields.ChainID,
				TransactionData:       tt.fields.TransactionData,
				Value:                 tt.fields.Value,
				Signature:             tt.fields.Signature,
				CreationDate:          tt.fields.CreationDate,
				Fee:                   tt.fields.Fee,
				TransactionType:       tt.fields.TransactionType,
				TransactionOutput:     tt.fields.TransactionOutput,
				OutputHash:            tt.fields.OutputHash,
				Status:                tt.fields.Status,
			}
			config.DevConfiguration.IsFeeEnabled = tt.enableFee

			if err := t.Validate(tt.args.ctx); (err != nil) != tt.wantErr {
				t1.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTransaction_GetScore(t1 *testing.T) {
	fee := state.Balance(1)

	type fields struct {
		HashIDField           datastore.HashIDField
		CollectionMemberField datastore.CollectionMemberField
		VersionField          datastore.VersionField
		ClientID              datastore.Key
		PublicKey             string
		ToClientID            datastore.Key
		ChainID               datastore.Key
		TransactionData       string
		Value                 state.Balance
		Signature             string
		CreationDate          common.Timestamp
		Fee                   state.Balance
		TransactionType       int
		TransactionOutput     string
		OutputHash            string
		Status                int
	}
	tests := []struct {
		name      string
		fields    fields
		want      int64
		enableFee bool
	}{
		{
			name: "Fee_Enabled_OK",
			fields: fields{
				Fee: fee,
			},
			enableFee: true,
			want:      int64(fee),
		},
		{
			name:      "Fee_Disabled_OK",
			enableFee: false,
			want:      0,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Transaction{
				HashIDField:           tt.fields.HashIDField,
				CollectionMemberField: tt.fields.CollectionMemberField,
				VersionField:          tt.fields.VersionField,
				ClientID:              tt.fields.ClientID,
				PublicKey:             tt.fields.PublicKey,
				ToClientID:            tt.fields.ToClientID,
				ChainID:               tt.fields.ChainID,
				TransactionData:       tt.fields.TransactionData,
				Value:                 tt.fields.Value,
				Signature:             tt.fields.Signature,
				CreationDate:          tt.fields.CreationDate,
				Fee:                   tt.fields.Fee,
				TransactionType:       tt.fields.TransactionType,
				TransactionOutput:     tt.fields.TransactionOutput,
				OutputHash:            tt.fields.OutputHash,
				Status:                tt.fields.Status,
			}
			config.DevConfiguration.IsFeeEnabled = tt.enableFee

			if got := t.GetScore(); got != tt.want {
				t1.Errorf("GetScore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTransaction_Read(t1 *testing.T) {
	store := mocks.Store{}
	store.On("Read", context.Context(nil), "", new(Transaction)).Return(
		func(_ context.Context, _ string, _ datastore.Entity) error {
			return nil
		},
	)

	transactionEntityMetadata.Store = &store

	type fields struct {
		HashIDField           datastore.HashIDField
		CollectionMemberField datastore.CollectionMemberField
		VersionField          datastore.VersionField
		ClientID              datastore.Key
		PublicKey             string
		ToClientID            datastore.Key
		ChainID               datastore.Key
		TransactionData       string
		Value                 state.Balance
		Signature             string
		CreationDate          common.Timestamp
		Fee                   state.Balance
		TransactionType       int
		TransactionOutput     string
		OutputHash            string
		Status                int
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
			t := &Transaction{
				HashIDField:           tt.fields.HashIDField,
				CollectionMemberField: tt.fields.CollectionMemberField,
				VersionField:          tt.fields.VersionField,
				ClientID:              tt.fields.ClientID,
				PublicKey:             tt.fields.PublicKey,
				ToClientID:            tt.fields.ToClientID,
				ChainID:               tt.fields.ChainID,
				TransactionData:       tt.fields.TransactionData,
				Value:                 tt.fields.Value,
				Signature:             tt.fields.Signature,
				CreationDate:          tt.fields.CreationDate,
				Fee:                   tt.fields.Fee,
				TransactionType:       tt.fields.TransactionType,
				TransactionOutput:     tt.fields.TransactionOutput,
				OutputHash:            tt.fields.OutputHash,
				Status:                tt.fields.Status,
			}
			if err := t.Read(tt.args.ctx, tt.args.key); (err != nil) != tt.wantErr {
				t1.Errorf("Read() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTransaction_Write(t1 *testing.T) {
	store := mocks.Store{}
	store.On("Write", context.Context(nil), new(Transaction)).Return(
		func(_ context.Context, _ datastore.Entity) error {
			return nil
		},
	)

	transactionEntityMetadata.Store = &store

	type fields struct {
		HashIDField           datastore.HashIDField
		CollectionMemberField datastore.CollectionMemberField
		VersionField          datastore.VersionField
		ClientID              datastore.Key
		PublicKey             string
		ToClientID            datastore.Key
		ChainID               datastore.Key
		TransactionData       string
		Value                 state.Balance
		Signature             string
		CreationDate          common.Timestamp
		Fee                   state.Balance
		TransactionType       int
		TransactionOutput     string
		OutputHash            string
		Status                int
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
			t := &Transaction{
				HashIDField:           tt.fields.HashIDField,
				CollectionMemberField: tt.fields.CollectionMemberField,
				VersionField:          tt.fields.VersionField,
				ClientID:              tt.fields.ClientID,
				PublicKey:             tt.fields.PublicKey,
				ToClientID:            tt.fields.ToClientID,
				ChainID:               tt.fields.ChainID,
				TransactionData:       tt.fields.TransactionData,
				Value:                 tt.fields.Value,
				Signature:             tt.fields.Signature,
				CreationDate:          tt.fields.CreationDate,
				Fee:                   tt.fields.Fee,
				TransactionType:       tt.fields.TransactionType,
				TransactionOutput:     tt.fields.TransactionOutput,
				OutputHash:            tt.fields.OutputHash,
				Status:                tt.fields.Status,
			}
			if err := t.Write(tt.args.ctx); (err != nil) != tt.wantErr {
				t1.Errorf("Write() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTransaction_Delete(t1 *testing.T) {
	store := mocks.Store{}
	store.On("Delete", context.Context(nil), new(Transaction)).Return(
		func(_ context.Context, _ datastore.Entity) error {
			return nil
		},
	)

	transactionEntityMetadata.Store = &store

	type fields struct {
		HashIDField           datastore.HashIDField
		CollectionMemberField datastore.CollectionMemberField
		VersionField          datastore.VersionField
		ClientID              datastore.Key
		PublicKey             string
		ToClientID            datastore.Key
		ChainID               datastore.Key
		TransactionData       string
		Value                 state.Balance
		Signature             string
		CreationDate          common.Timestamp
		Fee                   state.Balance
		TransactionType       int
		TransactionOutput     string
		OutputHash            string
		Status                int
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
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Transaction{
				HashIDField:           tt.fields.HashIDField,
				CollectionMemberField: tt.fields.CollectionMemberField,
				VersionField:          tt.fields.VersionField,
				ClientID:              tt.fields.ClientID,
				PublicKey:             tt.fields.PublicKey,
				ToClientID:            tt.fields.ToClientID,
				ChainID:               tt.fields.ChainID,
				TransactionData:       tt.fields.TransactionData,
				Value:                 tt.fields.Value,
				Signature:             tt.fields.Signature,
				CreationDate:          tt.fields.CreationDate,
				Fee:                   tt.fields.Fee,
				TransactionType:       tt.fields.TransactionType,
				TransactionOutput:     tt.fields.TransactionOutput,
				OutputHash:            tt.fields.OutputHash,
				Status:                tt.fields.Status,
			}
			if err := t.Delete(tt.args.ctx); (err != nil) != tt.wantErr {
				t1.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTransaction_GetCollectionName(t1 *testing.T) {
	txnEntityCollection = &datastore.EntityCollection{CollectionName: "collection.txn"}

	type fields struct {
		HashIDField           datastore.HashIDField
		CollectionMemberField datastore.CollectionMemberField
		VersionField          datastore.VersionField
		ClientID              datastore.Key
		PublicKey             string
		ToClientID            datastore.Key
		ChainID               datastore.Key
		TransactionData       string
		Value                 state.Balance
		Signature             string
		CreationDate          common.Timestamp
		Fee                   state.Balance
		TransactionType       int
		TransactionOutput     string
		OutputHash            string
		Status                int
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "OK",
			want: txnEntityCollection.CollectionName,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Transaction{
				HashIDField:           tt.fields.HashIDField,
				CollectionMemberField: tt.fields.CollectionMemberField,
				VersionField:          tt.fields.VersionField,
				ClientID:              tt.fields.ClientID,
				PublicKey:             tt.fields.PublicKey,
				ToClientID:            tt.fields.ToClientID,
				ChainID:               tt.fields.ChainID,
				TransactionData:       tt.fields.TransactionData,
				Value:                 tt.fields.Value,
				Signature:             tt.fields.Signature,
				CreationDate:          tt.fields.CreationDate,
				Fee:                   tt.fields.Fee,
				TransactionType:       tt.fields.TransactionType,
				TransactionOutput:     tt.fields.TransactionOutput,
				OutputHash:            tt.fields.OutputHash,
				Status:                tt.fields.Status,
			}
			if got := t.GetCollectionName(); got != tt.want {
				t1.Errorf("GetCollectionName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTransaction_GetHash(t1 *testing.T) {
	hash := encryption.Hash("data")

	type fields struct {
		HashIDField           datastore.HashIDField
		CollectionMemberField datastore.CollectionMemberField
		VersionField          datastore.VersionField
		ClientID              datastore.Key
		PublicKey             string
		ToClientID            datastore.Key
		ChainID               datastore.Key
		TransactionData       string
		Value                 state.Balance
		Signature             string
		CreationDate          common.Timestamp
		Fee                   state.Balance
		TransactionType       int
		TransactionOutput     string
		OutputHash            string
		Status                int
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name:   "OK",
			fields: fields{HashIDField: datastore.HashIDField{Hash: hash}},
			want:   hash,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Transaction{
				HashIDField:           tt.fields.HashIDField,
				CollectionMemberField: tt.fields.CollectionMemberField,
				VersionField:          tt.fields.VersionField,
				ClientID:              tt.fields.ClientID,
				PublicKey:             tt.fields.PublicKey,
				ToClientID:            tt.fields.ToClientID,
				ChainID:               tt.fields.ChainID,
				TransactionData:       tt.fields.TransactionData,
				Value:                 tt.fields.Value,
				Signature:             tt.fields.Signature,
				CreationDate:          tt.fields.CreationDate,
				Fee:                   tt.fields.Fee,
				TransactionType:       tt.fields.TransactionType,
				TransactionOutput:     tt.fields.TransactionOutput,
				OutputHash:            tt.fields.OutputHash,
				Status:                tt.fields.Status,
			}
			if got := t.GetHash(); got != tt.want {
				t1.Errorf("GetHash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTransaction_GetHashBytes(t1 *testing.T) {
	hash := encryption.Hash("data")

	type fields struct {
		HashIDField           datastore.HashIDField
		CollectionMemberField datastore.CollectionMemberField
		VersionField          datastore.VersionField
		ClientID              datastore.Key
		PublicKey             string
		ToClientID            datastore.Key
		ChainID               datastore.Key
		TransactionData       string
		Value                 state.Balance
		Signature             string
		CreationDate          common.Timestamp
		Fee                   state.Balance
		TransactionType       int
		TransactionOutput     string
		OutputHash            string
		Status                int
	}
	tests := []struct {
		name   string
		fields fields
		want   []byte
	}{
		{
			name:   "OK",
			fields: fields{HashIDField: datastore.HashIDField{Hash: hash}},
			want:   util.HashStringToBytes(hash),
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Transaction{
				HashIDField:           tt.fields.HashIDField,
				CollectionMemberField: tt.fields.CollectionMemberField,
				VersionField:          tt.fields.VersionField,
				ClientID:              tt.fields.ClientID,
				PublicKey:             tt.fields.PublicKey,
				ToClientID:            tt.fields.ToClientID,
				ChainID:               tt.fields.ChainID,
				TransactionData:       tt.fields.TransactionData,
				Value:                 tt.fields.Value,
				Signature:             tt.fields.Signature,
				CreationDate:          tt.fields.CreationDate,
				Fee:                   tt.fields.Fee,
				TransactionType:       tt.fields.TransactionType,
				TransactionOutput:     tt.fields.TransactionOutput,
				OutputHash:            tt.fields.OutputHash,
				Status:                tt.fields.Status,
			}
			if got := t.GetHashBytes(); !reflect.DeepEqual(got, tt.want) {
				t1.Errorf("GetHashBytes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTransaction_HashData(t1 *testing.T) {
	t := makeTestTxn()
	hashdata := common.TimeToString(t.CreationDate) + ":" + t.ClientID + ":" + t.ToClientID + ":" +
		strconv.FormatInt(int64(t.Value), 10) + ":" + encryption.Hash(t.TransactionData)

	type fields struct {
		HashIDField           datastore.HashIDField
		CollectionMemberField datastore.CollectionMemberField
		VersionField          datastore.VersionField
		ClientID              datastore.Key
		PublicKey             string
		ToClientID            datastore.Key
		ChainID               datastore.Key
		TransactionData       string
		Value                 state.Balance
		Signature             string
		CreationDate          common.Timestamp
		Fee                   state.Balance
		TransactionType       int
		TransactionOutput     string
		OutputHash            string
		Status                int
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "OK",
			fields: fields{
				HashIDField:           t.HashIDField,
				CollectionMemberField: t.CollectionMemberField,
				VersionField:          t.VersionField,
				ClientID:              t.ClientID,
				PublicKey:             t.PublicKey,
				ToClientID:            t.ToClientID,
				ChainID:               t.ChainID,
				TransactionData:       t.TransactionData,
				Value:                 t.Value,
				Signature:             t.Signature,
				CreationDate:          t.CreationDate,
				Fee:                   t.Fee,
				TransactionType:       t.TransactionType,
				TransactionOutput:     t.TransactionOutput,
				OutputHash:            t.OutputHash,
				Status:                t.Status,
			},
			want: hashdata,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Transaction{
				HashIDField:           tt.fields.HashIDField,
				CollectionMemberField: tt.fields.CollectionMemberField,
				VersionField:          tt.fields.VersionField,
				ClientID:              tt.fields.ClientID,
				PublicKey:             tt.fields.PublicKey,
				ToClientID:            tt.fields.ToClientID,
				ChainID:               tt.fields.ChainID,
				TransactionData:       tt.fields.TransactionData,
				Value:                 tt.fields.Value,
				Signature:             tt.fields.Signature,
				CreationDate:          tt.fields.CreationDate,
				Fee:                   tt.fields.Fee,
				TransactionType:       tt.fields.TransactionType,
				TransactionOutput:     tt.fields.TransactionOutput,
				OutputHash:            tt.fields.OutputHash,
				Status:                tt.fields.Status,
			}
			if got := t.HashData(); got != tt.want {
				t1.Errorf("HashData() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTransaction_ComputeHash(t1 *testing.T) {
	t := makeTestTxn()
	hashdata := common.TimeToString(t.CreationDate) + ":" + t.ClientID + ":" + t.ToClientID + ":" +
		strconv.FormatInt(int64(t.Value), 10) + ":" + encryption.Hash(t.TransactionData)

	type fields struct {
		HashIDField           datastore.HashIDField
		CollectionMemberField datastore.CollectionMemberField
		VersionField          datastore.VersionField
		ClientID              datastore.Key
		PublicKey             string
		ToClientID            datastore.Key
		ChainID               datastore.Key
		TransactionData       string
		Value                 state.Balance
		Signature             string
		CreationDate          common.Timestamp
		Fee                   state.Balance
		TransactionType       int
		TransactionOutput     string
		OutputHash            string
		Status                int
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "OK",
			fields: fields{
				HashIDField:           t.HashIDField,
				CollectionMemberField: t.CollectionMemberField,
				VersionField:          t.VersionField,
				ClientID:              t.ClientID,
				PublicKey:             t.PublicKey,
				ToClientID:            t.ToClientID,
				ChainID:               t.ChainID,
				TransactionData:       t.TransactionData,
				Value:                 t.Value,
				Signature:             t.Signature,
				CreationDate:          t.CreationDate,
				Fee:                   t.Fee,
				TransactionType:       t.TransactionType,
				TransactionOutput:     t.TransactionOutput,
				OutputHash:            t.OutputHash,
				Status:                t.Status,
			},
			want: encryption.Hash(hashdata),
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Transaction{
				HashIDField:           tt.fields.HashIDField,
				CollectionMemberField: tt.fields.CollectionMemberField,
				VersionField:          tt.fields.VersionField,
				ClientID:              tt.fields.ClientID,
				PublicKey:             tt.fields.PublicKey,
				ToClientID:            tt.fields.ToClientID,
				ChainID:               tt.fields.ChainID,
				TransactionData:       tt.fields.TransactionData,
				Value:                 tt.fields.Value,
				Signature:             tt.fields.Signature,
				CreationDate:          tt.fields.CreationDate,
				Fee:                   tt.fields.Fee,
				TransactionType:       tt.fields.TransactionType,
				TransactionOutput:     tt.fields.TransactionOutput,
				OutputHash:            tt.fields.OutputHash,
				Status:                tt.fields.Status,
			}
			if got := t.ComputeHash(); got != tt.want {
				t1.Errorf("ComputeHash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTransaction_VerifyHash(t1 *testing.T) {
	t := makeTestTxn()
	t.Hash = t.ComputeHash()

	type fields struct {
		HashIDField           datastore.HashIDField
		CollectionMemberField datastore.CollectionMemberField
		VersionField          datastore.VersionField
		ClientID              datastore.Key
		PublicKey             string
		ToClientID            datastore.Key
		ChainID               datastore.Key
		TransactionData       string
		Value                 state.Balance
		Signature             string
		CreationDate          common.Timestamp
		Fee                   state.Balance
		TransactionType       int
		TransactionOutput     string
		OutputHash            string
		Status                int
	}
	type args struct {
		in0 context.Context
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
				HashIDField:           t.HashIDField,
				CollectionMemberField: t.CollectionMemberField,
				VersionField:          t.VersionField,
				ClientID:              t.ClientID,
				PublicKey:             t.PublicKey,
				ToClientID:            t.ToClientID,
				ChainID:               t.ChainID,
				TransactionData:       t.TransactionData,
				Value:                 t.Value,
				Signature:             t.Signature,
				CreationDate:          t.CreationDate,
				Fee:                   t.Fee,
				TransactionType:       t.TransactionType,
				TransactionOutput:     t.TransactionOutput,
				OutputHash:            t.OutputHash,
				Status:                t.Status,
			},
			wantErr: false,
		},
		{
			name: "ERR",
			fields: fields{
				HashIDField:           datastore.HashIDField{},
				CollectionMemberField: t.CollectionMemberField,
				VersionField:          t.VersionField,
				ClientID:              t.ClientID,
				PublicKey:             t.PublicKey,
				ToClientID:            t.ToClientID,
				ChainID:               t.ChainID,
				TransactionData:       t.TransactionData,
				Value:                 t.Value,
				Signature:             t.Signature,
				CreationDate:          t.CreationDate,
				Fee:                   t.Fee,
				TransactionType:       t.TransactionType,
				TransactionOutput:     t.TransactionOutput,
				OutputHash:            t.OutputHash,
				Status:                t.Status,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Transaction{
				HashIDField:           tt.fields.HashIDField,
				CollectionMemberField: tt.fields.CollectionMemberField,
				VersionField:          tt.fields.VersionField,
				ClientID:              tt.fields.ClientID,
				PublicKey:             tt.fields.PublicKey,
				ToClientID:            tt.fields.ToClientID,
				ChainID:               tt.fields.ChainID,
				TransactionData:       tt.fields.TransactionData,
				Value:                 tt.fields.Value,
				Signature:             tt.fields.Signature,
				CreationDate:          tt.fields.CreationDate,
				Fee:                   tt.fields.Fee,
				TransactionType:       tt.fields.TransactionType,
				TransactionOutput:     tt.fields.TransactionOutput,
				OutputHash:            tt.fields.OutputHash,
				Status:                tt.fields.Status,
			}
			if err := t.VerifyHash(tt.args.in0); (err != nil) != tt.wantErr {
				t1.Errorf("VerifyHash() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTransaction_VerifySignature(t1 *testing.T) {
	var (
		t   = makeTestTxn()
		prK string
		err error
	)
	if t.PublicKey, prK, err = encryption.GenerateKeys(); err != nil {
		t1.Fatal(err)
	}
	if t.Signature, err = encryption.Sign(prK, t.ComputeHash()); err != nil {
		t1.Fatal(err)
	}

	t.Hash = t.ComputeHash()

	store := mocks.Store{}
	store.On("Read", context.TODO(), "", client.NewClient()).Return(
		func(_ context.Context, _ string, _ datastore.Entity) error {
			return errors.New("")
		},
	)

	transactionSummaryEntityMetadata.Store = &store

	clientEntityMetadata := datastore.MetadataProvider()
	clientEntityMetadata.Store = &store
	client.SetEntityMetadata(clientEntityMetadata)

	type fields struct {
		HashIDField           datastore.HashIDField
		CollectionMemberField datastore.CollectionMemberField
		VersionField          datastore.VersionField
		ClientID              datastore.Key
		PublicKey             string
		ToClientID            datastore.Key
		ChainID               datastore.Key
		TransactionData       string
		Value                 state.Balance
		Signature             string
		CreationDate          common.Timestamp
		Fee                   state.Balance
		TransactionType       int
		TransactionOutput     string
		OutputHash            string
		Status                int
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name            string
		fields          fields
		args            args
		clientSigScheme string
		wantErr         bool
	}{
		{
			name:    "Get_Sign_Scheme_ERR",
			args:    args{ctx: context.TODO()},
			wantErr: true,
		},
		{
			name: "Wrong_Sign_ERR",
			fields: fields{
				HashIDField:           datastore.HashIDField{},
				CollectionMemberField: t.CollectionMemberField,
				VersionField:          t.VersionField,
				ClientID:              t.ClientID,
				PublicKey:             t.PublicKey,
				ToClientID:            t.ToClientID,
				ChainID:               t.ChainID,
				TransactionData:       t.TransactionData,
				Value:                 t.Value,
				Signature:             "",
				CreationDate:          t.CreationDate,
				Fee:                   t.Fee,
				TransactionType:       t.TransactionType,
				TransactionOutput:     t.TransactionOutput,
				OutputHash:            t.OutputHash,
				Status:                t.Status,
			},
			clientSigScheme: "ed25519",
			wantErr:         true,
		},
		{
			name: "Signing_ERR",
			fields: fields{
				HashIDField:           datastore.HashIDField{},
				CollectionMemberField: t.CollectionMemberField,
				VersionField:          t.VersionField,
				ClientID:              t.ClientID,
				PublicKey:             t.PublicKey,
				ToClientID:            t.ToClientID,
				ChainID:               t.ChainID,
				TransactionData:       t.TransactionData,
				Value:                 t.Value,
				Signature:             "",
				CreationDate:          t.CreationDate,
				Fee:                   t.Fee,
				TransactionType:       t.TransactionType,
				TransactionOutput:     t.TransactionOutput,
				OutputHash:            t.OutputHash,
				Status:                t.Status,
			},
			clientSigScheme: "bls0chain",
			wantErr:         true,
		},
		{
			name: "OK",
			fields: fields{
				HashIDField:           t.HashIDField,
				CollectionMemberField: t.CollectionMemberField,
				VersionField:          t.VersionField,
				ClientID:              t.ClientID,
				PublicKey:             t.PublicKey,
				ToClientID:            t.ToClientID,
				ChainID:               t.ChainID,
				TransactionData:       t.TransactionData,
				Value:                 t.Value,
				Signature:             t.Signature,
				CreationDate:          t.CreationDate,
				Fee:                   t.Fee,
				TransactionType:       t.TransactionType,
				TransactionOutput:     t.TransactionOutput,
				OutputHash:            t.OutputHash,
				Status:                t.Status,
			},
			clientSigScheme: "ed25519",
			wantErr:         false,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Transaction{
				HashIDField:           tt.fields.HashIDField,
				CollectionMemberField: tt.fields.CollectionMemberField,
				VersionField:          tt.fields.VersionField,
				ClientID:              tt.fields.ClientID,
				PublicKey:             tt.fields.PublicKey,
				ToClientID:            tt.fields.ToClientID,
				ChainID:               tt.fields.ChainID,
				TransactionData:       tt.fields.TransactionData,
				Value:                 tt.fields.Value,
				Signature:             tt.fields.Signature,
				CreationDate:          tt.fields.CreationDate,
				Fee:                   tt.fields.Fee,
				TransactionType:       tt.fields.TransactionType,
				TransactionOutput:     tt.fields.TransactionOutput,
				OutputHash:            tt.fields.OutputHash,
				Status:                tt.fields.Status,
			}
			client.SetClientSignatureScheme(tt.clientSigScheme)

			if err := t.VerifySignature(tt.args.ctx); (err != nil) != tt.wantErr {
				t1.Errorf("VerifySignature() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTransaction_Sign(t1 *testing.T) {
	var (
		t      = makeTestTxn()
		scheme = encryption.NewED25519Scheme()
		err    error
	)
	if err = scheme.GenerateKeys(); err != nil {
		t1.Fatal(err)
	}
	t.PublicKey = scheme.GetPublicKey()

	if t.Signature, err = scheme.Sign(t.ComputeHash()); err != nil {
		t1.Fatal(err)
	}

	client.SetClientSignatureScheme("ed25519")

	type fields struct {
		HashIDField           datastore.HashIDField
		CollectionMemberField datastore.CollectionMemberField
		VersionField          datastore.VersionField
		ClientID              datastore.Key
		PublicKey             string
		ToClientID            datastore.Key
		ChainID               datastore.Key
		TransactionData       string
		Value                 state.Balance
		Signature             string
		CreationDate          common.Timestamp
		Fee                   state.Balance
		TransactionType       int
		TransactionOutput     string
		OutputHash            string
		Status                int
	}
	type args struct {
		signatureScheme encryption.SignatureScheme
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "OK",
			fields: fields{
				HashIDField:           datastore.HashIDField{},
				CollectionMemberField: t.CollectionMemberField,
				VersionField:          t.VersionField,
				ClientID:              t.ClientID,
				PublicKey:             t.PublicKey,
				ToClientID:            t.ToClientID,
				ChainID:               t.ChainID,
				TransactionData:       t.TransactionData,
				Value:                 t.Value,
				Signature:             "",
				CreationDate:          t.CreationDate,
				Fee:                   t.Fee,
				TransactionType:       t.TransactionType,
				TransactionOutput:     t.TransactionOutput,
				OutputHash:            t.OutputHash,
				Status:                t.Status,
			},
			args: args{signatureScheme: scheme},
			want: t.Signature,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Transaction{
				HashIDField:           tt.fields.HashIDField,
				CollectionMemberField: tt.fields.CollectionMemberField,
				VersionField:          tt.fields.VersionField,
				ClientID:              tt.fields.ClientID,
				PublicKey:             tt.fields.PublicKey,
				ToClientID:            tt.fields.ToClientID,
				ChainID:               tt.fields.ChainID,
				TransactionData:       tt.fields.TransactionData,
				Value:                 tt.fields.Value,
				Signature:             tt.fields.Signature,
				CreationDate:          tt.fields.CreationDate,
				Fee:                   tt.fields.Fee,
				TransactionType:       tt.fields.TransactionType,
				TransactionOutput:     tt.fields.TransactionOutput,
				OutputHash:            tt.fields.OutputHash,
				Status:                tt.fields.Status,
			}
			got, err := t.Sign(tt.args.signatureScheme)
			if (err != nil) != tt.wantErr {
				t1.Errorf("Sign() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t1.Errorf("Sign() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTransaction_GetSummary(t1 *testing.T) {
	t := makeTestTxn()
	t.Hash = t.ComputeHash()

	type fields struct {
		HashIDField           datastore.HashIDField
		CollectionMemberField datastore.CollectionMemberField
		VersionField          datastore.VersionField
		ClientID              datastore.Key
		PublicKey             string
		ToClientID            datastore.Key
		ChainID               datastore.Key
		TransactionData       string
		Value                 state.Balance
		Signature             string
		CreationDate          common.Timestamp
		Fee                   state.Balance
		TransactionType       int
		TransactionOutput     string
		OutputHash            string
		Status                int
	}
	tests := []struct {
		name   string
		fields fields
		want   *TransactionSummary
	}{
		{
			name: "OK",
			fields: fields{
				HashIDField:           t.HashIDField,
				CollectionMemberField: t.CollectionMemberField,
				VersionField:          t.VersionField,
				ClientID:              t.ClientID,
				PublicKey:             t.PublicKey,
				ToClientID:            t.ToClientID,
				ChainID:               t.ChainID,
				TransactionData:       t.TransactionData,
				Value:                 t.Value,
				Signature:             t.Signature,
				CreationDate:          t.CreationDate,
				Fee:                   t.Fee,
				TransactionType:       t.TransactionType,
				TransactionOutput:     t.TransactionOutput,
				OutputHash:            t.OutputHash,
				Status:                t.Status,
			},
			want: func() *TransactionSummary {
				summary := datastore.GetEntityMetadata("txn_summary").Instance().(*TransactionSummary)
				summary.Hash = t.Hash
				return summary
			}(),
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Transaction{
				HashIDField:           tt.fields.HashIDField,
				CollectionMemberField: tt.fields.CollectionMemberField,
				VersionField:          tt.fields.VersionField,
				ClientID:              tt.fields.ClientID,
				PublicKey:             tt.fields.PublicKey,
				ToClientID:            tt.fields.ToClientID,
				ChainID:               tt.fields.ChainID,
				TransactionData:       tt.fields.TransactionData,
				Value:                 tt.fields.Value,
				Signature:             tt.fields.Signature,
				CreationDate:          tt.fields.CreationDate,
				Fee:                   tt.fields.Fee,
				TransactionType:       tt.fields.TransactionType,
				TransactionOutput:     tt.fields.TransactionOutput,
				OutputHash:            tt.fields.OutputHash,
				Status:                tt.fields.Status,
			}
			if got := t.GetSummary(); !reflect.DeepEqual(got, tt.want) {
				t1.Errorf("GetSummary() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTransaction_DebugTxn(t1 *testing.T) {
	type fields struct {
		HashIDField           datastore.HashIDField
		CollectionMemberField datastore.CollectionMemberField
		VersionField          datastore.VersionField
		ClientID              datastore.Key
		PublicKey             string
		ToClientID            datastore.Key
		ChainID               datastore.Key
		TransactionData       string
		Value                 state.Balance
		Signature             string
		CreationDate          common.Timestamp
		Fee                   state.Balance
		TransactionType       int
		TransactionOutput     string
		OutputHash            string
		Status                int
	}
	tests := []struct {
		name        string
		fields      fields
		development bool
		want        bool
	}{
		{
			name:        "Development_FALSE",
			development: true,
			want:        false,
		},
		{
			name:        "No_Debug_FALSE",
			development: false,
			want:        false,
		},
		{
			name:        "TRUE",
			fields:      fields{TransactionData: "debug"},
			development: false,
			want:        false,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Transaction{
				HashIDField:           tt.fields.HashIDField,
				CollectionMemberField: tt.fields.CollectionMemberField,
				VersionField:          tt.fields.VersionField,
				ClientID:              tt.fields.ClientID,
				PublicKey:             tt.fields.PublicKey,
				ToClientID:            tt.fields.ToClientID,
				ChainID:               tt.fields.ChainID,
				TransactionData:       tt.fields.TransactionData,
				Value:                 tt.fields.Value,
				Signature:             tt.fields.Signature,
				CreationDate:          tt.fields.CreationDate,
				Fee:                   tt.fields.Fee,
				TransactionType:       tt.fields.TransactionType,
				TransactionOutput:     tt.fields.TransactionOutput,
				OutputHash:            tt.fields.OutputHash,
				Status:                tt.fields.Status,
			}
			if tt.development {
				config.Configuration.DeploymentMode = config.DeploymentDevelopment
			} else {
				config.Configuration.DeploymentMode = config.DeploymentTestNet
			}

			if got := t.DebugTxn(); got != tt.want {
				t1.Errorf("DebugTxn() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTransaction_ComputeOutputHash(t1 *testing.T) {
	output := "output"

	type fields struct {
		HashIDField           datastore.HashIDField
		CollectionMemberField datastore.CollectionMemberField
		VersionField          datastore.VersionField
		ClientID              datastore.Key
		PublicKey             string
		ToClientID            datastore.Key
		ChainID               datastore.Key
		TransactionData       string
		Value                 state.Balance
		Signature             string
		CreationDate          common.Timestamp
		Fee                   state.Balance
		TransactionType       int
		TransactionOutput     string
		OutputHash            string
		Status                int
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name:   "Empty_Output_OK",
			fields: fields{TransactionOutput: ""},
			want:   encryption.EmptyHash,
		},
		{
			name:   "OK",
			fields: fields{TransactionOutput: output},
			want:   encryption.Hash(output),
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Transaction{
				HashIDField:           tt.fields.HashIDField,
				CollectionMemberField: tt.fields.CollectionMemberField,
				VersionField:          tt.fields.VersionField,
				ClientID:              tt.fields.ClientID,
				PublicKey:             tt.fields.PublicKey,
				ToClientID:            tt.fields.ToClientID,
				ChainID:               tt.fields.ChainID,
				TransactionData:       tt.fields.TransactionData,
				Value:                 tt.fields.Value,
				Signature:             tt.fields.Signature,
				CreationDate:          tt.fields.CreationDate,
				Fee:                   tt.fields.Fee,
				TransactionType:       tt.fields.TransactionType,
				TransactionOutput:     tt.fields.TransactionOutput,
				OutputHash:            tt.fields.OutputHash,
				Status:                tt.fields.Status,
			}
			if got := t.ComputeOutputHash(); got != tt.want {
				t1.Errorf("ComputeOutputHash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTransaction_VerifyOutputHash(t1 *testing.T) {
	t := makeTestTxn()
	t.TransactionOutput = "output"
	t.OutputHash = t.ComputeOutputHash()

	type fields struct {
		HashIDField           datastore.HashIDField
		CollectionMemberField datastore.CollectionMemberField
		VersionField          datastore.VersionField
		ClientID              datastore.Key
		PublicKey             string
		ToClientID            datastore.Key
		ChainID               datastore.Key
		TransactionData       string
		Value                 state.Balance
		Signature             string
		CreationDate          common.Timestamp
		Fee                   state.Balance
		TransactionType       int
		TransactionOutput     string
		OutputHash            string
		Status                int
	}
	type args struct {
		in0 context.Context
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
				HashIDField:           t.HashIDField,
				CollectionMemberField: t.CollectionMemberField,
				VersionField:          t.VersionField,
				ClientID:              t.ClientID,
				PublicKey:             t.PublicKey,
				ToClientID:            t.ToClientID,
				ChainID:               t.ChainID,
				TransactionData:       t.TransactionData,
				Value:                 t.Value,
				Signature:             t.Signature,
				CreationDate:          t.CreationDate,
				Fee:                   t.Fee,
				TransactionType:       t.TransactionType,
				TransactionOutput:     t.TransactionOutput,
				OutputHash:            t.OutputHash,
				Status:                t.Status,
			},
			wantErr: false,
		},
		{
			name: "ERR",
			fields: fields{
				HashIDField:           t.HashIDField,
				CollectionMemberField: t.CollectionMemberField,
				VersionField:          t.VersionField,
				ClientID:              t.ClientID,
				PublicKey:             t.PublicKey,
				ToClientID:            t.ToClientID,
				ChainID:               t.ChainID,
				TransactionData:       t.TransactionData,
				Value:                 t.Value,
				Signature:             t.Signature,
				CreationDate:          t.CreationDate,
				Fee:                   t.Fee,
				TransactionType:       t.TransactionType,
				TransactionOutput:     t.TransactionOutput,
				OutputHash:            "",
				Status:                t.Status,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Transaction{
				HashIDField:           tt.fields.HashIDField,
				CollectionMemberField: tt.fields.CollectionMemberField,
				VersionField:          tt.fields.VersionField,
				ClientID:              tt.fields.ClientID,
				PublicKey:             tt.fields.PublicKey,
				ToClientID:            tt.fields.ToClientID,
				ChainID:               tt.fields.ChainID,
				TransactionData:       tt.fields.TransactionData,
				Value:                 tt.fields.Value,
				Signature:             tt.fields.Signature,
				CreationDate:          tt.fields.CreationDate,
				Fee:                   tt.fields.Fee,
				TransactionType:       tt.fields.TransactionType,
				TransactionOutput:     tt.fields.TransactionOutput,
				OutputHash:            tt.fields.OutputHash,
				Status:                tt.fields.Status,
			}
			if err := t.VerifyOutputHash(tt.args.in0); (err != nil) != tt.wantErr {
				t1.Errorf("VerifyOutputHash() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetTransactionCount(t *testing.T) {
	transactionCount = 2

	tests := []struct {
		name string
		want uint64
	}{
		{
			name: "OK",
			want: transactionCount,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetTransactionCount(); got != tt.want {
				t.Errorf("GetTransactionCount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIncTransactionCount(t *testing.T) {
	tests := []struct {
		name string
		want uint64
	}{
		{
			name: "OK",
			want: transactionCount + 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IncTransactionCount(); got != tt.want {
				t.Errorf("IncTransactionCount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTransaction_ComputeProperties(t1 *testing.T) {
	chainID := "chain id"
	config.SetServerChainID(chainID)

	t, ok := Provider().(*Transaction)
	if !ok {
		t1.Error("expected transaction entity")
	}
	t.ChainID = ""

	want, ok := Provider().(*Transaction)
	if !ok {
		t1.Error("expected transaction entity")
	}
	want.EntityCollection = txnEntityCollection
	want.ChainID = chainID

	type fields struct {
		HashIDField           datastore.HashIDField
		CollectionMemberField datastore.CollectionMemberField
		VersionField          datastore.VersionField
		ClientID              datastore.Key
		PublicKey             string
		ToClientID            datastore.Key
		ChainID               datastore.Key
		TransactionData       string
		Value                 state.Balance
		Signature             string
		CreationDate          common.Timestamp
		Fee                   state.Balance
		TransactionType       int
		TransactionOutput     string
		OutputHash            string
		Status                int
	}
	tests := []struct {
		name   string
		fields fields
		want   *Transaction
	}{
		{
			name: "OK",
			fields: fields{
				HashIDField:           t.HashIDField,
				CollectionMemberField: t.CollectionMemberField,
				VersionField:          t.VersionField,
				ClientID:              t.ClientID,
				PublicKey:             t.PublicKey,
				ToClientID:            t.ToClientID,
				ChainID:               t.ChainID,
				TransactionData:       t.TransactionData,
				Value:                 t.Value,
				Signature:             t.Signature,
				CreationDate:          t.CreationDate,
				Fee:                   t.Fee,
				TransactionType:       t.TransactionType,
				TransactionOutput:     t.TransactionOutput,
				OutputHash:            t.OutputHash,
				Status:                t.Status,
			},
			want: want,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Transaction{
				HashIDField:           tt.fields.HashIDField,
				CollectionMemberField: tt.fields.CollectionMemberField,
				VersionField:          tt.fields.VersionField,
				ClientID:              tt.fields.ClientID,
				PublicKey:             tt.fields.PublicKey,
				ToClientID:            tt.fields.ToClientID,
				ChainID:               tt.fields.ChainID,
				TransactionData:       tt.fields.TransactionData,
				Value:                 tt.fields.Value,
				Signature:             tt.fields.Signature,
				CreationDate:          tt.fields.CreationDate,
				Fee:                   tt.fields.Fee,
				TransactionType:       tt.fields.TransactionType,
				TransactionOutput:     tt.fields.TransactionOutput,
				OutputHash:            tt.fields.OutputHash,
				Status:                tt.fields.Status,
			}

			t.ComputeProperties()
			if !reflect.DeepEqual(t, tt.want) {
				t1.Errorf("ComputeProperties() = %v, want = %v", t, tt.want)
			}
		})
	}
}

func TestTransaction_GetClient(t1 *testing.T) {
	store := mocks.Store{}
	store.On("Read", context.Context(nil), "", client.NewClient()).Return(
		func(_ context.Context, _ string, _ datastore.Entity) error {
			return nil
		},
	)
	store.On("Read", context.TODO(), "", client.NewClient()).Return(
		func(_ context.Context, _ string, _ datastore.Entity) error {
			return errors.New("")
		},
	)

	transactionSummaryEntityMetadata.Store = &store

	clientEntityMetadata := datastore.MetadataProvider()
	clientEntityMetadata.Store = &store
	client.SetEntityMetadata(clientEntityMetadata)

	type fields struct {
		HashIDField           datastore.HashIDField
		CollectionMemberField datastore.CollectionMemberField
		VersionField          datastore.VersionField
		ClientID              datastore.Key
		PublicKey             string
		ToClientID            datastore.Key
		ChainID               datastore.Key
		TransactionData       string
		Value                 state.Balance
		Signature             string
		CreationDate          common.Timestamp
		Fee                   state.Balance
		TransactionType       int
		TransactionOutput     string
		OutputHash            string
		Status                int
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *client.Client
		wantErr bool
	}{
		{
			name:    "ERR",
			args:    args{ctx: context.TODO()},
			wantErr: true,
		},
		{
			name:    "OK",
			want:    client.NewClient(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Transaction{
				HashIDField:           tt.fields.HashIDField,
				CollectionMemberField: tt.fields.CollectionMemberField,
				VersionField:          tt.fields.VersionField,
				ClientID:              tt.fields.ClientID,
				PublicKey:             tt.fields.PublicKey,
				ToClientID:            tt.fields.ToClientID,
				ChainID:               tt.fields.ChainID,
				TransactionData:       tt.fields.TransactionData,
				Value:                 tt.fields.Value,
				Signature:             tt.fields.Signature,
				CreationDate:          tt.fields.CreationDate,
				Fee:                   tt.fields.Fee,
				TransactionType:       tt.fields.TransactionType,
				TransactionOutput:     tt.fields.TransactionOutput,
				OutputHash:            tt.fields.OutputHash,
				Status:                tt.fields.Status,
			}
			got, err := t.GetClient(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t1.Errorf("GetClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t1.Errorf("GetClient() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTransaction_GetSignatureScheme(t1 *testing.T) {
	store := mocks.Store{}
	store.On("Read", context.TODO(), mock.AnythingOfType("string"), mock.AnythingOfType("*client.Client")).Return(
		func(_ context.Context, _ string, _ datastore.Entity) error {
			return errors.New("")
		},
	)

	pbK, _, err := encryption.GenerateKeys()
	if err != nil {
		t1.Fatal(err)
	}
	clientID := "client id"
	want := client.NewClient()
	want.ID = clientID
	want.SetPublicKey(pbK)

	clientEntityMetadata := datastore.MetadataProvider()
	clientEntityMetadata.Store = &store
	client.SetEntityMetadata(clientEntityMetadata)

	type fields struct {
		HashIDField           datastore.HashIDField
		CollectionMemberField datastore.CollectionMemberField
		VersionField          datastore.VersionField
		ClientID              datastore.Key
		PublicKey             string
		ToClientID            datastore.Key
		ChainID               datastore.Key
		TransactionData       string
		Value                 state.Balance
		Signature             string
		CreationDate          common.Timestamp
		Fee                   state.Balance
		TransactionType       int
		TransactionOutput     string
		OutputHash            string
		Status                int
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    encryption.SignatureScheme
		wantErr bool
	}{
		{
			name: "Empty_Public_Key_ERR",
			fields: fields{
				ClientID: "unknown client id",
			},
			args:    args{ctx: context.TODO()},
			wantErr: true,
		},
		{
			name: "OK",
			fields: fields{
				PublicKey: pbK,
				ClientID:  clientID,
			},
			want:    want.GetSignatureScheme(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Transaction{
				HashIDField:           tt.fields.HashIDField,
				CollectionMemberField: tt.fields.CollectionMemberField,
				VersionField:          tt.fields.VersionField,
				ClientID:              tt.fields.ClientID,
				PublicKey:             tt.fields.PublicKey,
				ToClientID:            tt.fields.ToClientID,
				ChainID:               tt.fields.ChainID,
				TransactionData:       tt.fields.TransactionData,
				Value:                 tt.fields.Value,
				Signature:             tt.fields.Signature,
				CreationDate:          tt.fields.CreationDate,
				Fee:                   tt.fields.Fee,
				TransactionType:       tt.fields.TransactionType,
				TransactionOutput:     tt.fields.TransactionOutput,
				OutputHash:            tt.fields.OutputHash,
				Status:                tt.fields.Status,
			}
			got, err := t.GetSignatureScheme(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t1.Errorf("GetSignatureScheme() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t1.Errorf("GetSignatureScheme() got = %v, want %v", got, tt.want)
			}
		})
	}
}
