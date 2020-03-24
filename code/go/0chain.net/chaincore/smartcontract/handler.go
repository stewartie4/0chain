package smartcontract

import (
	"0chain.net/core/encryption"
	"0chain.net/core/util"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	c_state "0chain.net/chaincore/chain/state"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	. "0chain.net/core/logging"
	metrics "github.com/rcrowley/go-metrics"
	"go.uber.org/zap"
)

//lock used to setup smartcontract rest handlers
var scLock = sync.RWMutex{}

//contractMap - stores the map of valid smart contracts mapping from its address to its interface implementation
var contractMap = map[string]sci.SmartContractInterface{}

//ExecuteRestAPI - executes the rest api on the smart contract
func ExecuteRestAPI(ctx context.Context, scAddress string, restpath string, params url.Values, balances c_state.StateContextI) (interface{}, error) {
	smi, sc := getSmartContract(scAddress)
	if sc == nil {
		return nil, common.NewError("invalid_sc", "Invalid Smart contract address")
	}
	//add bc context here
	handler, restpathok := sc.RestHandlers[restpath]
	if !restpathok {
		return nil, common.NewError("invalid_path", "Invalid path")
	}

	if !smi.UseSelfState() {
		return handler(ctx, params, balances)
	}

	balances, done := GetStateSmartContract(balances, smi)
	defer done()

	return handler(ctx, params, balances)

}

func ExecuteStats(ctx context.Context, scAdress string, params url.Values, w http.ResponseWriter) {
	_, sc := getSmartContract(scAdress)
	if sc != nil {
		int, err := sc.HandlerStats(ctx, params)
		if err != nil {
			Logger.Warn("unexpected error", zap.Error(err))
		}
		fmt.Fprintf(w, "%v", int)
		return
	}
	fmt.Fprintf(w, "invalid_sc: Invalid Smart contract address")
}

func getSmartContract(scAddress string) (sci.SmartContractInterface, *sci.SmartContract) {
	scLock.RLock()
	defer scLock.RUnlock()
	contract, ok := contractMap[scAddress]
	if !ok {
		return nil, nil
	}

	bc := &BCContext{}
	contract.SetContextBC(bc)

	return contract, contract.GetSmartContract()
}

func SetSmartContract(scAddress string, smartContract sci.SmartContractInterface) {
	scLock.Lock()
	defer scLock.Unlock()
	contractMap[scAddress] = smartContract
}

func GetSmartContract(scAddress string) (sci.SmartContractInterface, bool) {
	scLock.RLock()
	defer scLock.RUnlock()
	sc, ok := contractMap[scAddress]
	return sc, ok
}

func GetSmartContractsKeys() []string {
	scLock.RLock()
	defer scLock.RUnlock()
	result := make([]string, 0, len(contractMap))
	for key := range contractMap {
		result = append(result, key)
	}
	return result
}

func ExecuteWithStats(smcoi sci.SmartContractInterface, sc *sci.SmartContract,
	t *transaction.Transaction, funcName string, input []byte,
	balances c_state.StateContextI) (string, error) {

	ts := time.Now()
	inter, err := smcoi.Execute(t, funcName, input, balances)
	if sc.SmartContractExecutionStats[funcName] != nil {
		timer, ok := sc.SmartContractExecutionStats[funcName].(metrics.Timer)
		if ok {
			timer.Update(time.Since(ts))
		}
	}
	return inter, err
}

func getRootSmartContract(_ context.Context, sc sci.SmartContractInterface, balances c_state.StateContextI) (util.MerklePatriciaTrieI, error) {
	//util.Path(encryption.Hash(sc.GetAddress()))
	clientState := balances.GetState()

	path := util.Path(encryption.Hash(sc.GetAddress()))
	scState, err := clientState.GetNodeValue(path)
	if err != nil && util.ErrNodeNotFound != nil {
		return nil, err
	}

	if err == util.ErrNodeNotFound {
		tdb := util.NewLevelNodeDB(util.NewMemoryNodeDB(), clientState.GetNodeDB(), false)
		scState := util.NewMerklePatriciaTrie(tdb, clientState.GetVersion())
		_, err = balances.InsertTrieNode(sc.GetAddress(), &util.KeyWrap{Key: scState.GetRoot()})
		if err != nil {
			return nil, err
		}
		return scState, nil
	}

	keySC := &util.KeyWrap{}
	if err == nil {
		err = keySC.Decode(scState.Encode())
		if err != nil {
			return nil, err
		}
	}

	//clientState.
	return nil, nil
}

func CreateMPT(mpt util.MerklePatriciaTrieI) util.MerklePatriciaTrieI {
	tdb := util.NewLevelNodeDB(util.NewMemoryNodeDB(), mpt.GetNodeDB(), false)
	tmpt := util.NewMerklePatriciaTrie(tdb, mpt.GetVersion())
	tmpt.SetRoot(mpt.GetRoot())
	return tmpt
}

type StateContextSCDecorator struct {
	c_state.StateContextI
	balanceStateOrigin util.MerklePatriciaTrieI
	stateOrigin        util.MerklePatriciaTrieI
	state              util.MerklePatriciaTrieI
	isDone             bool
}

func NewStateContextSCDecorator(balances c_state.StateContextI, stateOrigin util.MerklePatriciaTrieI) *StateContextSCDecorator {
	result := &StateContextSCDecorator{
		StateContextI:      balances,
		stateOrigin:        stateOrigin,
		state:              CreateMPT(stateOrigin),
		balanceStateOrigin: balances.GetState(),
	}
	balances.SetState(result.state)
	return result
}

func (s *StateContextSCDecorator) Done() {
	if !s.isDone {
		s.isDone = true
		s.StateContextI.SetState(s.balanceStateOrigin)
	}
}

func (s *StateContextSCDecorator) GetStateSC() util.MerklePatriciaTrieI {
	return s.state
}

func (s *StateContextSCDecorator) GetStateOrigin() util.MerklePatriciaTrieI {
	return s.stateOrigin
}
func (s *StateContextSCDecorator) GetStateGlobal() util.MerklePatriciaTrieI {
	return s.StateContextI.GetState()
}

//ExecuteSmartContract - executes the smart contract in the context of the given transaction
func ExecuteSmartContract(_ context.Context, t *transaction.Transaction,
	balances c_state.StateContextI) (string, error) {

	log.Println("ExecuteSmartContract with  global ROOT", balances.GetState().GetRoot())
	contractObj, contract := getSmartContract(t.ToClientID)
	if contractObj == nil {
		return "", common.NewError("invalid_smart_contract_address", "Invalid Smart Contract address")
	}

	balancesGlobal := balances
	balancesGlobalState := balances.GetState()
	restoreBalanceState := func() {}
	var stateSCOrigin util.MerklePatriciaTrieI
	if contractObj.UseSelfState() {
		nameSC := contractObj.GetName()
		b := balances.GetBlock()
		scs := b.GetSmartContractState()
		stateSCOrigin = scs.GetStateSmartContract(nameSC)
		if stateSCOrigin == nil {
			return "", common.NewError("invalid_smart_contract_state", "invalid Smart Contract state")
		}

		log.Println("Root SC=", nameSC, "root=", stateSCOrigin.GetRoot())

		balancesDecorator := NewStateContextSCDecorator(balances, stateSCOrigin)
		restoreBalanceState = func() {
			balancesDecorator.Done()
		}
		balances = balancesDecorator
	}
	defer restoreBalanceState()

	var smartContractData sci.SmartContractTransactionData
	dataBytes := []byte(t.TransactionData)
	err := json.Unmarshal(dataBytes, &smartContractData)
	if err != nil {
		Logger.Error("1 Error while decoding the JSON from transaction",
			zap.Any("input", t.TransactionData), zap.Error(err))
		log.Println("json error:", err)
		return "", err
	}

	transactionOutput, err := ExecuteWithStats(contractObj, contract, t,
		smartContractData.FunctionName, smartContractData.InputData, balances)
	if err != nil {
		log.Println("1 Error ExecuteWithStats error:", err)
		return "", err
	}
	restoreBalanceState()

	if contractObj.UseSelfState() {
		stateSC := balances.(*StateContextSCDecorator).GetStateSC() //state SC from StateContextSCDecorator
		balancesGlobalState.AddMergeChild(func() error {
			log.Println("Merge!")
			printStates(stateSC, stateSCOrigin)

			err := stateSCOrigin.MergeMPTChanges(stateSC)
			if err != nil {
				log.Println("Merged err=", err)
				return err
			}
			b := balancesGlobal.GetBlock()
			b.SmartContextStates.SetStateSmartContractHash(contractObj.GetName(), stateSCOrigin.GetRoot())
			log.Println("Merged! new root", stateSCOrigin.GetRoot())
			return nil
		})

		balancesGlobalState.AddSaveChild(func() error {
			err := stateSCOrigin.SaveChanges(contractObj.GetStateDB(), false)
			if err != nil {
				log.Println("SaveChanges error", err)
				return err
			}
			printStates(stateSCOrigin, stateSC)
			log.Println("Saved!")
			return nil
		})
	}

	return transactionOutput, nil
}

func printStates(cstate util.MerklePatriciaTrieI, pstate util.MerklePatriciaTrieI) {
	stateOut := os.Stdout
	fmt.Fprintf(stateOut, "== current state\n")
	cstate.PrettyPrint(stateOut)

	if pstate != nil {
		fmt.Fprintf(stateOut, "== previous state\n\n")
		pstate.PrettyPrint(stateOut)
	}
}

var ErrSmartContractNotFound = errors.New("smart contract not found")

func GetStateSmartContract(balances c_state.StateContextI, smartContract sci.SmartContractInterface) (c_state.StateContextI, func()) {
	if !smartContract.UseSelfState() {
		return balances, func() {}
	}
	name := smartContract.GetName()
	scs := balances.GetBlock().GetSmartContractState()
	stateSC := scs.GetStateSmartContract(name)
	balancesSC := NewStateContextSCDecorator(balances, stateSC)
	return balancesSC, balancesSC.Done
}
