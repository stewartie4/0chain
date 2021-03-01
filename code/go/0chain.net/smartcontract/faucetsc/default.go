package faucetsc

import (
	"encoding/json"
	"fmt"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
)

var (
	testFaucedHashIdOk    = datastore.HashIDField{Hash: "test_fauced_ok"}
	testFaucedHashIdWrong = datastore.HashIDField{Hash: "test_fauced_wrong"}

	testCollectionMemberFieldOK = datastore.CollectionMemberField{
		EntityCollection: &datastore.EntityCollection{
			CollectionName:     "collection.company",
			CollectionSize:     10000,
			CollectionDuration: time.Hour,
		},
		CollectionScore: 777,
	}

	testCollectionMemberFieldWrong = datastore.CollectionMemberField{
		EntityCollection: &datastore.EntityCollection{
			CollectionName:     "wrong-collection.company",
			CollectionSize:     10000,
			CollectionDuration: time.Hour,
		},
		CollectionScore: 777,
	}

	testVersionField = datastore.VersionField{Version: "1.0"}
)

const (
	clientID1 = "client_1"
	clientID2 = "client_2"
)

const (
	client1PubKey = "74f8a3642b07b5a13636909531619246e24bdd2697e9d25e59a4f7e001f65b0ebc09c356728216ef0f2b12d80ed29ab536fe8af4b4a3e22f68a7aff2103ff610"
	client2PubKey = "56cb37686ed110ad2e5e8a3bb2baefb793e553192da0cefb6999e335a71dfc2383f3ceef8640597c948bc3568b0edb1c6c26b2ee2a3c01a806d9bf5cab832d09"
)

const (
	testTxnDataOK    = "Txn: Pay 42 from 74f8a3642b07b5a13636909531619246e24bdd2697e9d25e59a4f7e001f65b0ebc09c356728216ef0f2b12d80ed29ab536fe8af4b4a3e22f68a7aff2103ff610\n"
	testTxnDataWrong = "Txn: Pay 1 from 99f8a3642b07b5a13636909531619246e24bdd2697e9d25e59a4f7e001f65b0ebc09c356728216ef0f2b12d80ed29ab536fe8af4b4a3e22f68a7aff2103ff610\n"
)

const (
	PourAmount      = 10
	MaxPourAmount   = 20
	PeriodicLimit   = 30
	GlobalLimit     = 30
	IndividualReset = 0
	GlobalReset     = 0
)

const (
	globalNode1Ok = "global_node1"
	globalNode2Ok = "global_node2"
)

var (
	now = common.Now()

	txnOutOk = fmt.Sprintf(`{"name":"payFees","input":{"round":%v}}`, 1)
)

func txnDataOk() string {
	txnData := &limitRequest{
		PourAmount:      PourAmount,
		MaxPourAmount:   MaxPourAmount,
		PeriodicLimit:   PeriodicLimit,
		GlobalLimit:     GlobalLimit,
		IndividualReset: IndividualReset,
		GlobalReset:     GlobalReset,
	}
	txnDataOk, _ := json.Marshal(txnData)
	return string(txnDataOk)
}

func globalNode1() *GlobalNode {
	return &GlobalNode{
		ID:              globalNode1Ok,
		PourAmount:      PourAmount,
		MaxPourAmount:   MaxPourAmount,
		PeriodicLimit:   PeriodicLimit,
		GlobalLimit:     GlobalLimit,
		IndividualReset: IndividualReset,
		GlobalReset:     GlobalReset,
		Used:            0,
		StartTime:       common.ToTime(now),
	}
}

func globalNode2() *GlobalNode {
	return &GlobalNode{
		ID:              globalNode2Ok,
		PourAmount:      PourAmount,
		MaxPourAmount:   MaxPourAmount,
		PeriodicLimit:   PeriodicLimit,
		GlobalLimit:     GlobalLimit,
		IndividualReset: IndividualReset,
		GlobalReset:     GlobalReset,
		Used:            0,
		StartTime:       common.ToTime(now),
	}
}

func globalNode1WithReset() *GlobalNode {
	return &GlobalNode{
		ID:              globalNode1Ok,
		PourAmount:      PourAmount,
		MaxPourAmount:   MaxPourAmount,
		PeriodicLimit:   PeriodicLimit,
		GlobalLimit:     GlobalLimit,
		IndividualReset: 1000000000,
		GlobalReset:     1000000000,
		Used:            0,
		StartTime:       common.ToTime(now),
	}
}

func makeTestTx1Ok() *transaction.Transaction {
	t := &transaction.Transaction{
		ClientID:          clientID1,
		ToClientID:        clientID2,
		ChainID:           config.GetMainChainID(),
		TransactionData:   testTxnDataOK,
		TransactionOutput: txnOutOk,
		Value:             1,
		TransactionType:   transaction.TxnTypeSmartContract,
		CreationDate:      now,
	}
	t.ComputeOutputHash()
	var scheme = encryption.NewBLS0ChainScheme()
	scheme.GenerateKeys()
	t.PublicKey = scheme.GetPublicKey()
	t.Sign(scheme)
	return t
}

func makeTestTxWithOwner_OK_and_BadRequest() *transaction.Transaction {
	t := &transaction.Transaction{
		ClientID:          owner,
		ToClientID:        clientID2,
		ChainID:           config.GetMainChainID(),
		TransactionData:   testTxnDataOK,
		TransactionOutput: txnOutOk,
		Value:             1,
		TransactionType:   transaction.TxnTypeSmartContract,
	}
	t.ComputeOutputHash()
	var scheme = encryption.NewBLS0ChainScheme()
	scheme.GenerateKeys()
	t.PublicKey = scheme.GetPublicKey()
	t.Sign(scheme)
	return t
}

func makeTestTxWithOwner_OK_and_RequestOK() *transaction.Transaction {
	t := &transaction.Transaction{
		ClientID:          owner,
		ToClientID:        clientID2,
		ChainID:           config.GetMainChainID(),
		TransactionData:   txnDataOk(),
		TransactionOutput: txnOutOk,
		Value:             1,
		TransactionType:   transaction.TxnTypeSmartContract,
	}
	t.ComputeOutputHash()
	var scheme = encryption.NewBLS0ChainScheme()
	scheme.GenerateKeys()
	t.PublicKey = scheme.GetPublicKey()
	t.Sign(scheme)
	return t
}

func makeTestBlockOk() {
	b := new(block.Block)
	b.MagicBlock = block.NewMagicBlock()
	b.MagicBlock.Miners = node.NewPool(node.NodeTypeMiner)
	b.MagicBlock.Sharders = node.NewPool(node.NodeTypeSharder)

	for _, mn := range miners {
		b.MagicBlock.Miners.NodesMap[mn.miner.id] = new(node.Node)
	}
	for _, sh := range sharders {
		b.MagicBlock.Sharders.NodesMap[sh.sharder.id] = new(node.Node)
	}
}

func newTestEmptyBalances() *testBalances {
	t := &testBalances{
		balances: make(map[datastore.Key]state.Balance),
		tree:     make(map[datastore.Key]util.Serializable),
	}
	return t
}

func newTestEmptyBalancesWithValue(key datastore.Key) *testBalances {
	t := &testBalances{
		balances: make(map[datastore.Key]state.Balance),
		tree:     make(map[datastore.Key]util.Serializable),
	}
	t.InsertTrieNode(key, newEmptyUserNode())
	return t
}

func newTest100BalancesWithValue(key datastore.Key) *testBalances {
	t := &testBalances{
		balances: make(map[datastore.Key]state.Balance),
		tree:     make(map[datastore.Key]util.Serializable),
	}
	t.InsertTrieNode(key, newEmptyUserNodeWith100())
	return t
}

func newEmptyUserNode() *UserNode {
	return &UserNode{
		ID:        clientID1,
		StartTime: common.ToTime(now),
		Used:      state.Balance(0),
	}
}

func newEmptyUserNodeWith100() *UserNode {
	return &UserNode{
		ID:        clientID1,
		StartTime: common.ToTime(now),
		Used:      state.Balance(100),
	}
}

