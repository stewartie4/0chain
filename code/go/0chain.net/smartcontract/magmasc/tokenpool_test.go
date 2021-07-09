package magmasc

import (
	"encoding/json"
	"reflect"
	"testing"

	chain "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	tx "0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
)

func Test_tokenPool_Decode(t *testing.T) {
	t.Parallel()

	pool := mockTokenPool()
	blob, err := json.Marshal(pool)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v | want: %v", err, nil)
	}

	tests := [2]struct {
		name  string
		blob  []byte
		want  *tokenPool
		error error
	}{
		{
			name:  "OK",
			blob:  blob,
			want:  pool,
			error: nil,
		},
		{
			name:  "Decode_ERR",
			blob:  []byte(":"), // invalid json
			want:  &tokenPool{},
			error: errDecodeData,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := &tokenPool{}
			if err = got.Decode(test.blob); !errIs(err, test.error) {
				t.Errorf("Decode() error: %v | want: %v", err, test.error)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("Decode() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_tokenPool_Encode(t *testing.T) {
	t.Parallel()

	pool := mockTokenPool()
	blob, err := json.Marshal(pool)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v | want: %v", err, nil)
	}

	tests := [1]struct {
		name string
		ackn *tokenPool
		want []byte
	}{
		{
			name: "OK",
			ackn: pool,
			want: blob,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.ackn.Encode(); !reflect.DeepEqual(got, test.want) {
				t.Errorf("Encode() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_tokenPool_create(t *testing.T) {
	t.Parallel()

	ackn, sci := mockAcknowledgment(), mockStateContextI()
	amount, txn := ackn.ProviderTerms.GetAmount(), sci.GetTransaction()

	txn.ClientID = ackn.ConsumerID
	txn.Value = int64(amount)

	resp := &tokenpool.TokenPoolTransferResponse{
		TxnHash:    txn.Hash,
		ToPool:     ackn.SessionID,
		Value:      amount,
		FromClient: ackn.ConsumerID,
		ToClient:   txn.ToClientID,
	}

	acknClientBalanceErr := mockAcknowledgment()
	acknClientBalanceErr.ConsumerID = ""

	acknInsufficientFundsErr := mockAcknowledgment()
	acknInsufficientFundsErr.ConsumerID = "insolvent_id"

	tests := [4]struct {
		name  string
		txn   *tx.Transaction
		ackn  *Acknowledgment
		pool  *tokenPool
		sci   chain.StateContextI
		want  string
		error bool
	}{
		{
			name:  "OK",
			txn:   txn,
			ackn:  ackn,
			pool:  &tokenPool{},
			sci:   sci,
			want:  string(resp.Encode()),
			error: false,
		},
		{
			name:  "Client_Balance_ERR",
			txn:   txn,
			ackn:  acknClientBalanceErr,
			pool:  &tokenPool{},
			sci:   sci,
			error: true,
		},
		{
			name:  "Insufficient_Funds_ERR",
			txn:   txn,
			ackn:  acknInsufficientFundsErr,
			pool:  &tokenPool{},
			sci:   sci,
			error: true,
		},
		{
			name:  "Add_Transfer_ERR",
			txn:   &tx.Transaction{ToClientID: "not_present_id"},
			ackn:  ackn,
			pool:  &tokenPool{},
			sci:   sci,
			error: true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := test.pool.create(test.txn, test.ackn, test.sci)
			if err == nil && got != test.want {
				t.Errorf("create() got: %v | want: %v", got, test.want)
				return
			}
			if (err != nil) != test.error {
				t.Errorf("create() error: %v | want: %v", err, test.error)
			}
		})
	}
}

func Test_tokenPool_spend(t *testing.T) {
	t.Parallel()

	sci, pool := mockStateContextI(), mockTokenPool()
	txn := sci.GetTransaction()

	resp := &tokenpool.TokenPoolTransferResponse{
		FromClient: txn.ToClientID,
		ToClient:   pool.PayeeID,
		FromPool:   pool.ID,
		Value:      100,
	}

	tests := [3]struct {
		name   string
		txn    *tx.Transaction
		amount state.Balance
		pool   *tokenPool
		sci    chain.StateContextI
		want   string
		error  bool
	}{
		{
			name:   "OK",
			txn:    txn,
			amount: resp.Value,
			pool:   pool,
			sci:    sci,
			want:   string(resp.Encode()),
			error:  false,
		},
		{
			name:   "Insufficient_Balance_ERR",
			txn:    txn,
			amount: 1111,
			pool:   mockTokenPool(),
			sci:    sci,
			error:  true,
		},
		{
			name:   "Add_Transfer_ERR",
			txn:    txn,
			amount: -1,
			pool:   mockTokenPool(),
			sci:    sci,
			error:  true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := test.pool.spend(test.txn, test.amount, test.sci)
			if err == nil && got != test.want {
				t.Errorf("spend() got: %v | want: %v", got, test.want)
				return
			}
			if (err != nil) != test.error {
				t.Errorf("spend() error: %v | want: %v", err, test.error)
			}
		})
	}
}

func Test_tokenPool_transfer(t *testing.T) {
	t.Parallel()

	sci, pool := mockStateContextI(), mockTokenPool()
	resp := &tokenpool.TokenPoolTransferResponse{
		FromClient: pool.PayerID,
		ToClient:   pool.PayeeID,
		Value:      pool.Balance,
		FromPool:   pool.ID,
	}

	poolAddTransferErr := mockTokenPool()
	poolAddTransferErr.Balance = -1

	tests := [3]struct {
		name   string
		fromID datastore.Key
		toID   datastore.Key
		sci    chain.StateContextI
		pool   *tokenPool
		want   string
		error  bool
	}{
		{
			name:   "OK",
			fromID: pool.PayerID,
			toID:   pool.PayeeID,
			sci:    sci,
			pool:   pool,
			want:   string(resp.Encode()),
			error:  false,
		},
		{
			name:  "Empty_Pool_ERR",
			pool:  &tokenPool{},
			error: true,
		},
		{
			name:  "Add_Transfer_ERR",
			sci:   sci,
			pool:  poolAddTransferErr,
			error: true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := test.pool.transfer(test.fromID, test.toID, test.sci)
			if err == nil && got != test.want {
				t.Errorf("transfer() got: %v | want: %v", got, test.want)
				return
			}
			if (err != nil) != test.error {
				t.Errorf("transfer() error: %v | want: %v", err, test.error)
			}
		})
	}
}

func Test_tokenPool_uid(t *testing.T) {
	t.Parallel()

	const (
		parentUID    = "parent_uid"
		tokenPoolID  = "token_pool_id"
		tokenPoolUID = parentUID + ":tokenpool:" + tokenPoolID
	)

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		pool := tokenPool{}
		pool.ID = tokenPoolID

		if got := pool.uid(parentUID); got != tokenPoolUID {
			t.Errorf("uid() got: %v | want: %v", got, tokenPoolUID)
		}
	})
}
