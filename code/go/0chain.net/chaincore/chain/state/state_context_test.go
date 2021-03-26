package state

import (
	"0chain.net/chaincore/block"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/logging"
	"0chain.net/core/util"
	mocks "0chain.net/mocks/core/datastore"
	"encoding/hex"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func init() {
	block.SetupEntity(&mocks.Store{})

	logging.InitLogging("testing")
}

func TestNewStateContext(t *testing.T) {
	b := block.NewBlock("", 1)
	s := util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1)
	csd := state.Deserializer{}
	txn := transaction.Transaction{}

	type args struct {
		b                             *block.Block
		s                             util.MerklePatriciaTrieI
		csd                           state.DeserializerI
		t                             *transaction.Transaction
		getSharderFunc                func(*block.Block) []string
		getLastestFinalizedMagicBlock func() *block.Block
		getChainSignature             func() encryption.SignatureScheme
	}
	tests := []struct {
		name         string
		args         args
		wantBalances *StateContext
	}{
		{
			name: "OK",
			args: args{
				b:   b,
				s:   s,
				csd: &csd,
				t:   &txn,
			},
			wantBalances: &StateContext{
				block:                   b,
				state:                   s,
				clientStateDeserializer: &csd,
				txn:                     &txn,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBalances := NewStateContext(tt.args.b, tt.args.s, tt.args.csd, tt.args.t, tt.args.getSharderFunc, tt.args.getLastestFinalizedMagicBlock, tt.args.getChainSignature)
			assert.Equal(t, tt.wantBalances, gotBalances)
		})
	}
}

func Test_Simple_Setters_And_Getters(t *testing.T) {
	b := block.NewBlock("", 1)
	s := util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1)
	csd := state.Deserializer{}
	txn := transaction.Transaction{}
	sc := NewStateContext(b, s, &csd, &txn, nil, nil, nil)

	assert.Equal(t, b, sc.GetBlock())

	mb := block.NewMagicBlock()
	sc.SetMagicBlock(mb)
	assert.Equal(t, mb, sc.block.MagicBlock)

	assert.Equal(t, s, sc.GetState())

	assert.Equal(t, &txn, sc.GetTransaction())

	st := &state.SignedTransfer{}
	sc.AddSignedTransfer(st)
	assert.Equal(t, sc.signedTransfers, []*state.SignedTransfer{st})

	sc.transfers = []*state.Transfer{
		state.NewTransfer("from client id", "to client id", 5),
	}
	assert.Equal(t, sc.transfers, sc.GetTransfers())

	assert.Equal(t, sc.signedTransfers, sc.GetSignedTransfers())

	sc.mints = []*state.Mint{
		{
			Minter: "minter",
		},
	}
	assert.Equal(t, sc.mints, sc.GetMints())

	sh := []string{"sharder"}
	sc.getSharders = func(*block.Block) []string {
		return sh
	}
	assert.Equal(t, sh, sc.GetBlockSharders(nil))

	b = block.NewBlock("", 1)
	sc.getLastestFinalizedMagicBlock = func() *block.Block {
		return b
	}
	assert.Equal(t, b, sc.GetLastestFinalizedMagicBlock())

	scheme := encryption.NewED25519Scheme()
	sc.getSignature = func() encryption.SignatureScheme {
		return scheme
	}
	assert.Equal(t, scheme, sc.GetSignatureScheme())
}

func TestStateContext_AddTransfer(t *testing.T) {
	toClientID := "to client id"

	type fields struct {
		block                         *block.Block
		state                         util.MerklePatriciaTrieI
		txn                           *transaction.Transaction
		transfers                     []*state.Transfer
		signedTransfers               []*state.SignedTransfer
		mints                         []*state.Mint
		clientStateDeserializer       state.DeserializerI
		getSharders                   func(*block.Block) []string
		getLastestFinalizedMagicBlock func() *block.Block
		getSignature                  func() encryption.SignatureScheme
	}
	type args struct {
		t *state.Transfer
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
				txn: &transaction.Transaction{},
			},
			args: args{
				t: &state.Transfer{
					Receiver: toClientID,
				},
			},
		},
		{
			name: "ERR",
			fields: fields{
				txn: &transaction.Transaction{
					ClientID:   toClientID,
					ToClientID: toClientID,
				},
			},
			args: args{
				t: &state.Transfer{
					Receiver: "to client id",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := &StateContext{
				block:                         tt.fields.block,
				state:                         tt.fields.state,
				txn:                           tt.fields.txn,
				transfers:                     tt.fields.transfers,
				signedTransfers:               tt.fields.signedTransfers,
				mints:                         tt.fields.mints,
				clientStateDeserializer:       tt.fields.clientStateDeserializer,
				getSharders:                   tt.fields.getSharders,
				getLastestFinalizedMagicBlock: tt.fields.getLastestFinalizedMagicBlock,
				getSignature:                  tt.fields.getSignature,
			}
			if err := sc.AddTransfer(tt.args.t); (err != nil) != tt.wantErr {
				t.Errorf("AddTransfer() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStateContext_AddMint(t *testing.T) {
	id := approvedMinters[0]

	type fields struct {
		block                         *block.Block
		state                         util.MerklePatriciaTrieI
		txn                           *transaction.Transaction
		transfers                     []*state.Transfer
		signedTransfers               []*state.SignedTransfer
		mints                         []*state.Mint
		clientStateDeserializer       state.DeserializerI
		getSharders                   func(*block.Block) []string
		getLastestFinalizedMagicBlock func() *block.Block
		getSignature                  func() encryption.SignatureScheme
	}
	type args struct {
		m *state.Mint
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
				txn: &transaction.Transaction{
					ToClientID: id,
				},
			},
			args: args{
				m: &state.Mint{
					Minter: id,
				},
			},
			wantErr: false,
		},
		{
			name: "ERR",
			fields: fields{
				txn: &transaction.Transaction{
					ToClientID: "unknown id",
				},
			},
			args: args{
				m: &state.Mint{
					Minter: id,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := &StateContext{
				block:                         tt.fields.block,
				state:                         tt.fields.state,
				txn:                           tt.fields.txn,
				transfers:                     tt.fields.transfers,
				signedTransfers:               tt.fields.signedTransfers,
				mints:                         tt.fields.mints,
				clientStateDeserializer:       tt.fields.clientStateDeserializer,
				getSharders:                   tt.fields.getSharders,
				getLastestFinalizedMagicBlock: tt.fields.getLastestFinalizedMagicBlock,
				getSignature:                  tt.fields.getSignature,
			}
			if err := sc.AddMint(tt.args.m); (err != nil) != tt.wantErr {
				t.Errorf("AddMint() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStateContext_Validate(t *testing.T) {
	txn := transaction.Transaction{
		ToClientID: "to client id",
		ClientID:   "client id",
		Value:      1,
	}

	scheme := encryption.NewED25519Scheme()
	if err := scheme.GenerateKeys(); err != nil {
		t.Fatal(err)
	}
	pbKBytes, err := hex.DecodeString(scheme.GetPublicKey())
	if err != nil {
		t.Fatal(err)
	}

	st := &state.SignedTransfer{
		Transfer: state.Transfer{
			Sender: encryption.Hash(pbKBytes),
			Amount: 1,
		},
		SchemeName: "ed25519",
		PublicKey:  scheme.GetPublicKey(),
	}
	if err := st.Sign(scheme); err != nil {
		t.Fatal(err)
	}

	type fields struct {
		block                         *block.Block
		state                         util.MerklePatriciaTrieI
		txn                           *transaction.Transaction
		transfers                     []*state.Transfer
		signedTransfers               []*state.SignedTransfer
		mints                         []*state.Mint
		clientStateDeserializer       state.DeserializerI
		getSharders                   func(*block.Block) []string
		getLastestFinalizedMagicBlock func() *block.Block
		getSignature                  func() encryption.SignatureScheme
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Invalid_Transfer_ERR",
			fields: fields{
				transfers: []*state.Transfer{
					{
						Sender: "unknown id",
					},
				},
				txn: &txn,
			},
			wantErr: true,
		},
		{
			name: "Invalid_Transfer_Negative_Amount_ERR",
			fields: fields{
				transfers: []*state.Transfer{
					{
						Receiver: txn.ToClientID,
						Amount:   -1,
					},
				},
				txn: &txn,
			},
			wantErr: true,
		},
		{
			name: "Invalid_Transfer_Amount_Greater_Than_Total_Value_ERR",
			fields: fields{
				transfers: []*state.Transfer{
					{
						Receiver: txn.ClientID,
						Amount:   2,
					},
				},
				txn: &txn,
			},
			wantErr: true,
		},
		{
			name: "Verifying_Signature_ERR",
			fields: fields{
				transfers: []*state.Transfer{
					{
						Sender: txn.ClientID,
						Amount: 1,
					},
				},
				txn: &txn,
				signedTransfers: []*state.SignedTransfer{
					{
						Transfer:   state.Transfer{},
						SchemeName: "ed25519",
						PublicKey:  "",
						Sig:        "",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Negative_Signed_Transfer_Amount_ERR",
			fields: fields{
				transfers: []*state.Transfer{
					{
						Receiver: txn.ClientID,
						Amount:   1,
					},
				},
				txn: &txn,
				signedTransfers: []*state.SignedTransfer{
					func() *state.SignedTransfer {
						st := *st
						st.Amount = 0

						return &st
					}(),
				},
			},
			wantErr: true,
		},
		{
			name: "OK",
			fields: fields{
				transfers: []*state.Transfer{
					{
						Sender: txn.ClientID,
						Amount: 1,
					},
				},
				txn: &txn,
				signedTransfers: []*state.SignedTransfer{
					st,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := &StateContext{
				block:                         tt.fields.block,
				state:                         tt.fields.state,
				txn:                           tt.fields.txn,
				transfers:                     tt.fields.transfers,
				signedTransfers:               tt.fields.signedTransfers,
				mints:                         tt.fields.mints,
				clientStateDeserializer:       tt.fields.clientStateDeserializer,
				getSharders:                   tt.fields.getSharders,
				getLastestFinalizedMagicBlock: tt.fields.getLastestFinalizedMagicBlock,
				getSignature:                  tt.fields.getSignature,
			}
			if err := sc.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStateContext_GetClientBalance(t *testing.T) {
	v := state.State{}

	st := util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1)
	path := encryption.RawHash(v.Encode())
	_, err := st.Insert(path, &v)
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		block                         *block.Block
		state                         util.MerklePatriciaTrieI
		txn                           *transaction.Transaction
		transfers                     []*state.Transfer
		signedTransfers               []*state.SignedTransfer
		mints                         []*state.Mint
		clientStateDeserializer       state.DeserializerI
		getSharders                   func(*block.Block) []string
		getLastestFinalizedMagicBlock func() *block.Block
		getSignature                  func() encryption.SignatureScheme
	}
	type args struct {
		clientID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    state.Balance
		wantErr bool
	}{
		{
			name: "ERR",
			fields: fields{
				state: util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1),
			},
			wantErr: true,
		},
		{
			name: "OK",
			fields: fields{
				state:                   st,
				clientStateDeserializer: &state.Deserializer{},
			},
			args: args{clientID: string(path)},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := &StateContext{
				block:                         tt.fields.block,
				state:                         tt.fields.state,
				txn:                           tt.fields.txn,
				transfers:                     tt.fields.transfers,
				signedTransfers:               tt.fields.signedTransfers,
				mints:                         tt.fields.mints,
				clientStateDeserializer:       tt.fields.clientStateDeserializer,
				getSharders:                   tt.fields.getSharders,
				getLastestFinalizedMagicBlock: tt.fields.getLastestFinalizedMagicBlock,
				getSignature:                  tt.fields.getSignature,
			}
			got, err := sc.GetClientBalance(tt.args.clientID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetClientBalance() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetClientBalance() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStateContext_GetTrieNode(t *testing.T) {
	v := state.State{Balance: 5}

	st := util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1)
	path := encryption.Hash(v.Encode())
	_, err := st.Insert([]byte(path), &v)
	if err != nil {
		t.Fatal(err)
	}

	nv, err := st.GetNodeValue([]byte(path))
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		block                         *block.Block
		state                         util.MerklePatriciaTrieI
		txn                           *transaction.Transaction
		transfers                     []*state.Transfer
		signedTransfers               []*state.SignedTransfer
		mints                         []*state.Mint
		clientStateDeserializer       state.DeserializerI
		getSharders                   func(*block.Block) []string
		getLastestFinalizedMagicBlock func() *block.Block
		getSignature                  func() encryption.SignatureScheme
	}
	type args struct {
		key datastore.Key
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    util.Serializable
		wantErr bool
	}{
		{
			name:    "ERR",
			args:    args{key: encryption.Hash("s")},
			wantErr: true,
		},
		{
			name: "OK",
			fields: fields{
				state: st,
			},
			args: args{key: datastore.Key(v.Encode())},
			want: nv,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := &StateContext{
				block:                         tt.fields.block,
				state:                         tt.fields.state,
				txn:                           tt.fields.txn,
				transfers:                     tt.fields.transfers,
				signedTransfers:               tt.fields.signedTransfers,
				mints:                         tt.fields.mints,
				clientStateDeserializer:       tt.fields.clientStateDeserializer,
				getSharders:                   tt.fields.getSharders,
				getLastestFinalizedMagicBlock: tt.fields.getLastestFinalizedMagicBlock,
				getSignature:                  tt.fields.getSignature,
			}
			got, err := sc.GetTrieNode(tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTrieNode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetTrieNode() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStateContext_DeleteTrieNode(t *testing.T) {
	v := state.State{Balance: 5}

	st := util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1)
	path := encryption.Hash(v.Encode())
	_, err := st.Insert([]byte(path), &v)
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		block                         *block.Block
		state                         util.MerklePatriciaTrieI
		txn                           *transaction.Transaction
		transfers                     []*state.Transfer
		signedTransfers               []*state.SignedTransfer
		mints                         []*state.Mint
		clientStateDeserializer       state.DeserializerI
		getSharders                   func(*block.Block) []string
		getLastestFinalizedMagicBlock func() *block.Block
		getSignature                  func() encryption.SignatureScheme
	}
	type args struct {
		key datastore.Key
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    datastore.Key
		wantErr bool
	}{
		{
			name:    "ERR",
			args:    args{key: encryption.Hash("s")},
			wantErr: true,
		},
		{
			name: "OK",
			fields: fields{
				state: st,
			},
			args:    args{key: datastore.Key(v.Encode())},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := &StateContext{
				block:                         tt.fields.block,
				state:                         tt.fields.state,
				txn:                           tt.fields.txn,
				transfers:                     tt.fields.transfers,
				signedTransfers:               tt.fields.signedTransfers,
				mints:                         tt.fields.mints,
				clientStateDeserializer:       tt.fields.clientStateDeserializer,
				getSharders:                   tt.fields.getSharders,
				getLastestFinalizedMagicBlock: tt.fields.getLastestFinalizedMagicBlock,
				getSignature:                  tt.fields.getSignature,
			}
			got, err := sc.DeleteTrieNode(tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteTrieNode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DeleteTrieNode() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStateContext_InsertTrieNode(t *testing.T) {
	v := state.State{Balance: 5}
	path := v.Encode()
	ln := util.NewLeafNode(util.Path(encryption.Hash(path)), 1, &v)

	st := util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1)

	type fields struct {
		block                         *block.Block
		state                         util.MerklePatriciaTrieI
		txn                           *transaction.Transaction
		transfers                     []*state.Transfer
		signedTransfers               []*state.SignedTransfer
		mints                         []*state.Mint
		clientStateDeserializer       state.DeserializerI
		getSharders                   func(*block.Block) []string
		getLastestFinalizedMagicBlock func() *block.Block
		getSignature                  func() encryption.SignatureScheme
	}
	type args struct {
		key  datastore.Key
		node util.Serializable
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    datastore.Key
		wantErr bool
	}{
		{
			name:    "ERR",
			args:    args{key: encryption.Hash("s")},
			wantErr: true,
		},
		{
			name: "OK",
			fields: fields{
				state: st,
			},
			args:    args{key: datastore.Key(path), node: &v},
			want:    datastore.Key(ln.GetHashBytes()),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := &StateContext{
				block:                         tt.fields.block,
				state:                         tt.fields.state,
				txn:                           tt.fields.txn,
				transfers:                     tt.fields.transfers,
				signedTransfers:               tt.fields.signedTransfers,
				mints:                         tt.fields.mints,
				clientStateDeserializer:       tt.fields.clientStateDeserializer,
				getSharders:                   tt.fields.getSharders,
				getLastestFinalizedMagicBlock: tt.fields.getLastestFinalizedMagicBlock,
				getSignature:                  tt.fields.getSignature,
			}
			got, err := sc.InsertTrieNode(tt.args.key, tt.args.node)
			if (err != nil) != tt.wantErr {
				t.Errorf("InsertTrieNode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("InsertTrieNode() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStateContext_SetStateContext(t *testing.T) {
	b := block.NewBlock("", 1)
	txn := transaction.Transaction{
		HashIDField: datastore.HashIDField{
			Hash: encryption.Hash("data"),
		},
	}

	wantS := state.State{}
	wantS.SetRound(b.Round)
	if err := wantS.SetTxnHash(txn.Hash); err != nil {
		t.Fatal(err)
	}

	type fields struct {
		block                         *block.Block
		state                         util.MerklePatriciaTrieI
		txn                           *transaction.Transaction
		transfers                     []*state.Transfer
		signedTransfers               []*state.SignedTransfer
		mints                         []*state.Mint
		clientStateDeserializer       state.DeserializerI
		getSharders                   func(*block.Block) []string
		getLastestFinalizedMagicBlock func() *block.Block
		getSignature                  func() encryption.SignatureScheme
	}
	type args struct {
		s *state.State
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantS   *state.State
		wantErr bool
	}{
		{
			name: "OK",
			fields: fields{
				block: b,
				txn:   &txn,
			},
			args:    args{s: &state.State{}},
			wantS:   &wantS,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := &StateContext{
				block:                         tt.fields.block,
				state:                         tt.fields.state,
				txn:                           tt.fields.txn,
				transfers:                     tt.fields.transfers,
				signedTransfers:               tt.fields.signedTransfers,
				mints:                         tt.fields.mints,
				clientStateDeserializer:       tt.fields.clientStateDeserializer,
				getSharders:                   tt.fields.getSharders,
				getLastestFinalizedMagicBlock: tt.fields.getLastestFinalizedMagicBlock,
				getSignature:                  tt.fields.getSignature,
			}
			if err := sc.SetStateContext(tt.args.s); (err != nil) != tt.wantErr {
				t.Errorf("SetStateContext() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, tt.wantS, tt.args.s)
		})
	}
}
