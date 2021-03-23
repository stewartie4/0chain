package storagesc

import (
	"testing"

	"0chain.net/chaincore/chain/state"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
)

const (
	bId1 = "blobber_1"
	bId2 = "blobber_2"
	bId3 = "blobber_3"

	bPk1 = "blobber_pk_1"
	bPk2 = "blobber_pk_2"
	bPk3 = "blobber_pk_3"

	bUrl1 = "/blobber/1"
	bUrl2 = "/blobber/2"
	bUrl3 = "/blobber/3"
)

func testBlobber(id, url, pk string) *StorageNode {
	return &StorageNode{
		ID:              id,
		BaseURL:         url,
		Terms:           Terms{},
		Capacity:        0,
		Used:            0,
		LastHealthCheck: 0,
		PublicKey:       pk,
	}
}

func TestStorageSmartContract_insertBlobber(t *testing.T) {
	type fields struct {
		SmartContract *sci.SmartContract
	}
	type args struct {
		t        *transaction.Transaction
		conf     *scConfig
		blobber  *StorageNode
		all      *StorageNodes
		balances state.StateContextI
	}
	emptyBalances := newTestBalances(t, false)
	client1 := newClient(100*x10, emptyBalances)
	client2 := newClient(110*x10, emptyBalances)

	type test struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		before  func(test)
	}

	tests := []test{
		{
			name:   "error: blobber already exists",
			fields: fields{SmartContract: sci.NewSC(ADDRESS)},
			args: args{
				t:       newTransaction(client1.id, client2.id, 100, 0),
				conf:    &scConfig{},
				blobber: testBlobber(bId1, bUrl1, bPk1),
				all: &StorageNodes{Nodes: []*StorageNode{
					0: testBlobber(bId1, bUrl1, bPk1),
					1: testBlobber(bId1, bUrl1, bPk1),
					2: testBlobber(bId1, bUrl1, bPk1),
				}},
				balances: emptyBalances,
			},
			wantErr: true,
		},
		{
			name:   "error: invalid stake_pool settings",
			fields: fields{SmartContract: sci.NewSC(ADDRESS)},
			args: args{
				t:       newTransaction(client1.id, client2.id, 100, 0),
				conf:    &scConfig{},
				blobber: testBlobber(bId1, bUrl1, bPk1),
				all: &StorageNodes{Nodes: []*StorageNode{
					0: testBlobber(bId2, bUrl2, bPk2),
					1: testBlobber(bId3, bUrl3, bPk3),
				}},
				balances: emptyBalances,
			},
			wantErr: true,
		},
		{
			name:   "error: negative service charge",
			fields: fields{SmartContract: sci.NewSC(ADDRESS)},
			args: args{
				t:       newTransaction(client1.id, client2.id, 100, 0),
				conf:    &scConfig{},
				blobber: testBlobber(bId1, bUrl1, bPk1),
				all: &StorageNodes{Nodes: []*StorageNode{
					0: testBlobber(bId2, bUrl2, bPk2),
					1: testBlobber(bId3, bUrl3, bPk3),
				}},
				balances: emptyBalances,
			},
			before: func(t test) {
				t.args.blobber.StakePoolSettings.ServiceCharge = -1
			},
			wantErr: true,
		},
		{
			name:   "error: service_charge  is greater then max allowed",
			fields: fields{SmartContract: sci.NewSC(ADDRESS)},
			args: args{
				t:       newTransaction(client1.id, client2.id, 100, 0),
				conf:    &scConfig{},
				blobber: testBlobber(bId1, bUrl1, bPk1),
				all: &StorageNodes{Nodes: []*StorageNode{
					0: testBlobber(bId2, bUrl2, bPk2),
					1: testBlobber(bId3, bUrl3, bPk3),
				}},
				balances: emptyBalances,
			},
			before: func(t test) {
				t.args.blobber.StakePoolSettings.ServiceCharge = 10
				t.args.conf.MaxCharge = 5
			},
			wantErr: true,
		},
		{
			name:   "error:  num_delegates <= 0",
			fields: fields{SmartContract: sci.NewSC(ADDRESS)},
			args: args{
				t:       newTransaction(client1.id, client2.id, 100, 0),
				conf:    &scConfig{},
				blobber: testBlobber(bId1, bUrl1, bPk1),
				all: &StorageNodes{Nodes: []*StorageNode{
					0: testBlobber(bId2, bUrl2, bPk2),
					1: testBlobber(bId3, bUrl3, bPk3),
				}},
				balances: emptyBalances,
			},
			wantErr: true,
		},
		{
			name:   "error: ok",
			fields: fields{SmartContract: sci.NewSC(ADDRESS)},
			args: args{
				t:       newTransaction(client1.id, client2.id, 100, 0),
				conf:    &scConfig{},
				blobber: testBlobber(bId1, bUrl1, bPk1),
				all: &StorageNodes{Nodes: []*StorageNode{
					0: testBlobber(bId2, bUrl2, bPk2),
					1: testBlobber(bId3, bUrl3, bPk3),
				}},
				balances: emptyBalances,
			},
			before: func(t test) {
				t.args.blobber.StakePoolSettings.NumDelegates = 2
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		if tt.before != nil {
			tt.before(tt)
		}

		t.Run(tt.name, func(t *testing.T) {
			sc := &StorageSmartContract{
				SmartContract: tt.fields.SmartContract,
			}
			if err := sc.insertBlobber(tt.args.t, tt.args.conf, tt.args.blobber, tt.args.all, tt.args.balances); (err != nil) != tt.wantErr {
				t.Errorf("insertBlobber() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
