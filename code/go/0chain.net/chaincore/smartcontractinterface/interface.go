package smartcontractinterface

import (
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
	"context"
	"encoding/json"
	"net/url"
	"sync"

	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
)

const Seperator = ":"

type SmartContractRestHandler func(ctx context.Context, params url.Values, balances c_state.StateContextI) (interface{}, error)

type SmartContract struct {
	mu        sync.RWMutex
	state     util.MerklePatriciaTrieI
	bcContext BCContextI
	db        *util.PNodeDB

	ID                          string
	Name                        string
	RestHandlers                map[string]SmartContractRestHandler
	SmartContractExecutionStats map[string]interface{}
}

func NewSC(id, name string, db *util.PNodeDB) *SmartContract {
	result := &SmartContract{
		db:                          db,
		ID:                          id,
		Name:                        name,
		RestHandlers:                make(map[string]SmartContractRestHandler),
		SmartContractExecutionStats: make(map[string]interface{}),
	}
	return result
}

type SmartContractTransactionData struct {
	FunctionName string          `json:"name"`
	InputData    json.RawMessage `json:"input"`
}

type SmartContractInterface interface {
	Execute(t *transaction.Transaction, funcName string, input []byte, balances c_state.StateContextI) (string, error)
	SetSC(sc *SmartContract)
	GetSmartContract() *SmartContract
	SetContextBC(bc BCContextI)
	GetContextBC() BCContextI
	GetRestPoints() map[string]SmartContractRestHandler
	GetName() string
	GetAddress() string

	GetState() util.MerklePatriciaTrieI
	//	GetStateFromGlobal(clientState util.MerklePatriciaTrieI, version util.Sequence) (util.MerklePatriciaTrieI, error)
	//	GetStateFromGlobalNoChange(clientState util.MerklePatriciaTrieI, version util.Sequence) (util.MerklePatriciaTrieI, error)

	InitState(key datastore.Key) util.MerklePatriciaTrieI
	UseSelfState() bool
	InitSC()

	GetStateDB() util.NodeDB

}

/*BCContextI interface for smart contracts to access blockchain.
These functions should not modify blockchain states in anyway.
*/
type BCContextI interface {
	GetNodepoolInfo() interface{}
}

func (sc *SmartContract) GetStateDB() util.NodeDB {
	return sc.db
}

func (sc *SmartContract) InitState(key datastore.Key) util.MerklePatriciaTrieI {
	tdb := util.NewLevelNodeDB(util.NewMemoryNodeDB(), sc.db, false)
	mpt := util.NewMerklePatriciaTrie(tdb, 0)
	mpt.Insert(util.Path(encryption.Hash(key)), &util.KeyWrap{Key: util.Key(sc.ID)})
	mpt.SaveChanges(sc.db, false)
	sc.state = mpt
	return mpt
}


func CreateMPT(mpt util.MerklePatriciaTrieI) util.MerklePatriciaTrieI {
	tdb := util.NewLevelNodeDB(util.NewMemoryNodeDB(), mpt.GetNodeDB(), false)
	tmpt := util.NewMerklePatriciaTrie(tdb, mpt.GetVersion())
	tmpt.SetRoot(mpt.GetRoot())
	return tmpt
}

func (sc *SmartContract) GetState() util.MerklePatriciaTrieI {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.state
}

func (sc *SmartContract) SetContextBC(bc BCContextI) {
	sc.bcContext = bc
}

func (sc *SmartContract) GetContextBC() BCContextI {
	return sc.bcContext
}

func (sc *SmartContract) GetSmartContract() *SmartContract {
	return sc
}
