package magmasc

import (
	"testing"

	chain "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	tx "0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
)

func Test_tokenPool_create(t *testing.T) {
	t.Parallel()

	ackn, sci := mockAcknowledgment(), mockStateContextI()
	amount := int64(ackn.ProviderTerms.GetVolume() * ackn.ProviderTerms.Price)
	txn := tx.Transaction{
		ClientID:   ackn.ConsumerID,
		ToClientID: ackn.ProviderID,
		Value:      amount,
	}
	resp := &tokenpool.TokenPoolTransferResponse{
		TxnHash:    txn.Hash,
		ToPool:     ackn.SessionID,
		Value:      state.Balance(amount),
		FromClient: ackn.ConsumerID,
		ToClient:   ackn.ProviderID,
	}

	acknClientBalanceErr := mockAcknowledgment()
	acknClientBalanceErr.ConsumerID = ""

	acknInsufficientFundsErr := mockAcknowledgment()
	acknInsufficientFundsErr.ConsumerID = "insolvent_id"

	acknAddTransferErr := mockAcknowledgment()
	acknAddTransferErr.ProviderID = "not_present_id"

	tests := [4]struct {
		name  string
		ackn  *Acknowledgment
		pool  *tokenPool
		sci   chain.StateContextI
		txn   *tx.Transaction
		want  string
		error bool
	}{
		{
			name:  "OK",
			ackn:  ackn,
			pool:  &tokenPool{},
			sci:   sci,
			txn:   &txn,
			want:  string(resp.Encode()),
			error: false,
		},
		{
			name:  "Client_Balance_ERR",
			ackn:  acknClientBalanceErr,
			pool:  &tokenPool{},
			sci:   sci,
			error: true,
		},
		{
			name:  "Insufficient_Funds_ERR",
			ackn:  acknInsufficientFundsErr,
			pool:  &tokenPool{},
			sci:   sci,
			error: true,
		},
		{
			name:  "Add_Transfer_ERR",
			ackn:  acknAddTransferErr,
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
	resp := &tokenpool.TokenPoolTransferResponse{
		FromClient: pool.ClientID,
		ToClient:   pool.DelegateID,
		FromPool:   pool.ID,
		Value:      100,
	}

	tests := [3]struct {
		name   string
		amount state.Balance
		pool   *tokenPool
		sci    chain.StateContextI
		want   string
		error  bool
	}{
		{
			name:   "OK",
			amount: resp.Value,
			pool:   pool,
			sci:    sci,
			want:   string(resp.Encode()),
			error:  false,
		},
		{
			name:   "Insufficient_Balance_ERR",
			amount: 1111,
			pool:   mockTokenPool(),
			sci:    sci,
			error:  true,
		},
		{
			name:   "Add_Transfer_ERR",
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

			got, err := test.pool.spend(test.amount, test.sci)
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
		FromClient: pool.ClientID,
		ToClient:   pool.DelegateID,
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
			fromID: pool.ClientID,
			toID:   pool.DelegateID,
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
