package smartcontractinterface

import (
	"context"
	"encoding/json"
	"net/url"

	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
)

const Seperator = ":"

type SmartContractRestHandler func(ctx context.Context, params url.Values) (interface{}, error)

type SmartContract struct {
	ID                          string
	RestHandlers                map[string]SmartContractRestHandler
	SmartContractExecutionStats map[string]interface{}
	SmartContractState          util.MerklePatriciaTrieI
}

func (sc *SmartContract) InsertNode(key datastore.Key, node util.Serializable) (datastore.Key, error) {
	if !encryption.IsSCHash(key) {
		return "", common.NewError("failed to get trie node", "not appropriate key")
	}
	byteKey, err := sc.SmartContractState.Insert(util.Path(key), node)
	return datastore.Key(byteKey), err
}

func (sc *SmartContract) GetNode(key datastore.Key) (util.Serializable, error) {
	if !encryption.IsSCHash(key) {
		return nil, common.NewError("failed to get trie node", "not appropriate key")
	}
	return sc.SmartContractState.GetNodeValue(util.Path(key))
}

func (sc *SmartContract) DeleteNode(key datastore.Key) (datastore.Key, error) {
	if !encryption.IsSCHash(key) {
		return "", common.NewError("failed to get trie node", "not appropriate key")
	}
	byteKey, err := sc.SmartContractState.Delete(util.Path(key))
	return datastore.Key(byteKey), err
}

func NewSC(id string, smartContractState util.MerklePatriciaTrieI) *SmartContract {
	restHandlers := make(map[string]SmartContractRestHandler)
	scExecStats := make(map[string]interface{})
	return &SmartContract{ID: id, RestHandlers: restHandlers, SmartContractExecutionStats: scExecStats, SmartContractState: smartContractState}
}

type SmartContractTransactionData struct {
	FunctionName string          `json:"name"`
	InputData    json.RawMessage `json:"input"`
}

type SmartContractInterface interface {
	Execute(t *transaction.Transaction, funcName string, input []byte, balances c_state.StateContextI) (string, error)
	SetSC(sc *SmartContract, bc BCContextI)
	GetRestPoints() map[string]SmartContractRestHandler
	GetName() string
	GetAddress() string
}

/*BCContextI interface for smart contracts to access blockchain.
These functions should not modify blockchain states in anyway.
*/
type BCContextI interface {
	GetNodepoolInfo() interface{}
}
