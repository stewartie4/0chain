package smartcontractinterface

import (
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
	"context"
	"encoding/json"
	"log"
	"net/url"
	"sync"

	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
)

const Seperator = ":"

const (
	ADDRESS = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712e3"
)

type SmartContractRestHandler func(ctx context.Context, params url.Values, balances c_state.StateContextI) (interface{}, error)

type SmartContract struct {
	mu           sync.RWMutex
	state        util.MerklePatriciaTrieI
	bcContext    BCContextI
	db           *util.PNodeDB


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


	InitState() util.MerklePatriciaTrieI
	UseSelfState() bool
	InitSC()
}

/*BCContextI interface for smart contracts to access blockchain.
These functions should not modify blockchain states in anyway.
*/
type BCContextI interface {
	GetNodepoolInfo() interface{}
}

func (sc *SmartContract) InitState() util.MerklePatriciaTrieI{
	key := sc.getKey()
	tdb := util.NewLevelNodeDB(util.NewMemoryNodeDB(), sc.db, false)
	mpt := util.NewMerklePatriciaTrie(tdb, 0)
	//_, err := mpt.GetNodeValue(util.Path(encryption.Hash(sc.ID)))
	//if err != nil && err != util.ErrValueNotPresent {
	mpt.Insert(util.Path(encryption.Hash(key)), &util.KeyWrap{Key: util.Key(sc.ID)})
	mpt.SaveChanges(sc.db, false)
	//}
	return mpt
}

func (sc *SmartContract) getState() util.MerklePatriciaTrieI {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.state
}

func (sc *SmartContract) getKey() datastore.Key {
	return datastore.Key(ADDRESS + sc.ID)
}

func (sc *SmartContract) getKeyGlobalState() datastore.Key {
	return datastore.Key(sc.ID + encryption.Hash("sc"))
}

func (sc *SmartContract) getKeySCRootFromGlobalState(mpt util.MerklePatriciaTrieI) (util.Key, error) {
	//root := mpt.GetRoot()
	//if len(root)==0 || root[0]!=2 {
		//debug.PrintStack()
	//}
	//return sc.state.GetRoot(), nil
	log.Println("getKeySCRootFromGlobalState got from Global ROOT", mpt.GetRoot())
	result := &util.KeyWrap{}

	key := sc.getKeyGlobalState()
	keyRootBytes, err := mpt.GetNodeValue(util.Path(encryption.Hash(key)))
	if err != nil && err != util.ErrValueNotPresent && err!=util.ErrNodeNotFound {
		log.Println("getKeySCRootFromGlobalState err", err)
		return nil, err
	}

	if keyRootBytes == nil {
		log.Println("getKeySCRootFromGlobalState return state.GetRoot()", )
		return nil, nil
	}

	err = result.Decode(keyRootBytes.Encode())
	if err != nil {
		return nil, err
	}
	log.Println("getKeySCRootFromGlobalState return key", result.Key, )
	return result.Key, nil
}

func (sc *SmartContract) setKeySCRootToGlobalStore(mpt util.MerklePatriciaTrieI, keySCRoot util.Key) error {
	key := sc.getKeyGlobalState()
	keyData := &util.KeyWrap{
		Key: keySCRoot,
	}
	_, err := mpt.Insert(util.Path(encryption.Hash(key)), keyData)
	if err != nil {
		//return err
	}
	return nil
}

func CreateMPT(mpt util.MerklePatriciaTrieI) util.MerklePatriciaTrieI {
	tdb := util.NewLevelNodeDB(util.NewMemoryNodeDB(), mpt.GetNodeDB(), false)
	tmpt := util.NewMerklePatriciaTrie(tdb, mpt.GetVersion())
	tmpt.SetRoot(mpt.GetRoot())
	return tmpt
}

func (sc *SmartContract) ApplyStateToGlobal(clientState util.MerklePatriciaTrieI,keySC util.Key/*balances c_state.StateContextI*/) error {
	//clientState := balances.GetState()
	//currentState := sc.getState()
	//keySC := currentState.GetRoot()
	log.Println("ApplyStateToGlobal keySC", keySC)
	if err := sc.setKeySCRootToGlobalStore(clientState, keySC); err != nil {
		log.Println("ApplyStateToGlobal err", err)
		return err
	}
	return nil
}

func (sc *SmartContract) MergeState(mpt util.MerklePatriciaTrieI) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.state.MergeMPTChanges(mpt)
}

func (sc *SmartContract) GetState() util.MerklePatriciaTrieI {
	return sc.getState()
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
