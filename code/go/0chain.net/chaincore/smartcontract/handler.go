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

	balances = GetStateSmartContract(balances, smi)
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
	stateOrigin util.MerklePatriciaTrieI
	state       util.MerklePatriciaTrieI
}

func NewStateContextSCDecorator(balances c_state.StateContextI, state util.MerklePatriciaTrieI) *StateContextSCDecorator {
	return &StateContextSCDecorator{
		StateContextI: balances,
		stateOrigin:   state,
		state:         CreateMPT(state),
	}
}

func (s *StateContextSCDecorator) GetState() util.MerklePatriciaTrieI {
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
	var stateOrigin util.MerklePatriciaTrieI
	if contractObj.UseSelfState() {
		nameSC := contractObj.GetName()
		stateOrigin = balances.GetBlock().SmartContextStates.GetStateSmartContract(nameSC)
		if stateOrigin == nil {
			return "", common.NewError("invalid_smart_contract_state", "invalid Smart Contract state")
		}
		balances = NewStateContextSCDecorator(balances, stateOrigin)
	}

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

	if contractObj.UseSelfState() {
		/*
			+ 0.0 +use cloned
			+ 1.1 bind mergeMPT
			? 1.2 apply root sc to global state
			+ 2.0 bind save

		*/

		stateSC := balances.GetState()
		//stateOrigin := (balances).(*StateContextSCDecorator).GetStateOrigin()
		stateGlobal := balancesGlobal.GetState()
		stateGlobal.AddMergeChild(func() error {
			log.Println("Merge!")
			printStates(stateSC, stateOrigin)

			err := stateOrigin.MergeMPTChanges(stateSC)
			if err != nil {
				log.Println("Merged err=", err)
				return err
			}
			b := balancesGlobal.GetBlock()
			b.SmartContextStates.SetStateSmartContractHash(contractObj.GetName(), stateOrigin.GetRoot())
			log.Println("Merged! new root", stateOrigin.GetRoot())
			//FIXME: save root sc to global state
			return nil
		})

		stateGlobal.AddSaveChild(func() error {
			err := stateOrigin.SaveChanges(stateOrigin.GetNodeDB(), false)
			if err != nil {
				log.Println("SaveChanges error", err)
				return err
			}
			printStates(stateOrigin, stateSC)
			log.Println("Saved!")
			return nil
		})
	}

	return transactionOutput, nil

	//
	//globalState := balances.GetState()
	//startRoot := contract.GetState().GetRoot()
	//
	//scState, err := contract.GetStateFromGlobal(globalState, globalState.GetVersion())
	//if err != nil {
	//	log.Println("call GetStateFromGlobal round=", balances.GetBlock().Round,
	//		"ClientStateHash=", balances.GetBlock().ClientStateHash,
	//		"Version=", balances.GetBlock().Version,
	//		" global root=", globalState.GetRoot(),
	//		"err=", err)
	//
	//	return "", err
	//}
	//
	//if len(scState.GetRoot()) == 0 {
	//	log.Println("SET DEFAULT ROOT", util.Key(startRoot))
	//	scState.SetRoot(util.Key(startRoot))
	//}
	//
	//balances.SetState(scState)
	//defer func() {
	//	balances.SetState(globalState)
	//}()
	//
	//var smartContractData sci.SmartContractTransactionData
	//dataBytes := []byte(t.TransactionData)
	//err = json.Unmarshal(dataBytes, &smartContractData)
	//if err != nil {
	//	Logger.Error("Error while decoding the JSON from transaction",
	//		zap.Any("input", t.TransactionData), zap.Error(err))
	//	log.Println("json error:", err)
	//	return "", err
	//}
	//
	//transactionOutput, err := ExecuteWithStats(contractObj, contract, t,
	//	smartContractData.FunctionName, smartContractData.InputData, balances)
	//if err != nil {
	//	log.Println("Error ExecuteWithStats error:", err)
	//	return "", err
	//}
	//
	//balances.SetState(globalState)
	//
	//printStates(scState, contract.GetState())
	//
	//globalState.AddMergeChild(func() error {
	//
	//	log.Println("MergeState!", err)
	//	if err := contract.ApplyStateToGlobal(globalState, scState.GetRoot()); err != nil {
	//		log.Println("ApplyStateToGlobal sc state error", err)
	//		return err
	//	}
	//
	//	if err := contract.MergeState(scState); err != nil {
	//		log.Println("MergeState merge sc state error", err)
	//		return err
	//	}
	//	return nil
	//})
	//
	///*if err := contract.ApplyStateToGlobal(balances); err != nil {
	//	log.Println("SetState sc state error", err)
	//	return "", err
	//}*/
	//
	//currentState := contract.GetState()
	//globalState.AddSaveChild(func() error {
	//	log.Println("SaveChanges!", err)
	//	if err := currentState.SaveChanges(currentState.GetNodeDB(), false); err != nil {
	//		log.Println("SaveChanges error", err)
	//	}
	//	return err
	//})
	//
	//checkState, err := contractObj.GetStateFromGlobalNoChange(globalState, globalState.GetVersion())
	//log.Println("CheckState  root=", checkState.GetRoot(), "error", err)
	//
	//log.Println("Stored round=", balances.GetBlock().Round,
	//	"ClientStateHash=", balances.GetBlock().ClientStateHash,
	//	"Version=", balances.GetBlock().Version,
	//	"mpt root=", contract.GetState().GetRoot(),
	//	"globalState root=", globalState.GetRoot())
	//
	//return transactionOutput, nil
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

/*
func GetStateSmartContractByAddress(address string, currentClientState util.MerklePatriciaTrieI) (util.MerklePatriciaTrieI, util.MerklePatriciaTrieI, error) {
	clientState := CreateMPT(currentClientState)
	sc, ok := contractMap[address]
	if !ok {
		return nil, nil, ErrSmartContractNotFound
	}
	if !sc.UseSelfState() {
		return clientState, currentClientState, nil
	}

	scState, err := sc.GetStateFromGlobalNoChange(clientState, clientState.GetVersion())
	if err != nil {
		return nil, nil, err
	}
	return clientState, scState, nil
}*/

func GetStateSmartContract(balances c_state.StateContextI, smartContract sci.SmartContractInterface) c_state.StateContextI {
	if !smartContract.UseSelfState() {
		return balances
	}
	name := smartContract.GetName()
	stateSC := balances.GetBlock().SmartContextStates.GetStateSmartContract(name)
	balances = NewStateContextSCDecorator(balances, stateSC)
	return balances
}
