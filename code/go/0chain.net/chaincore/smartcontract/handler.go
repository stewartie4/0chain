package smartcontract

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/rcrowley/go-metrics"
	"go.uber.org/zap"

	c_state "0chain.net/chaincore/chain/state"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	. "0chain.net/core/logging"
)

type SmartContract struct {
	mtx      sync.Mutex
	contract map[string]sci.SmartContractInterface
}

var SCObject *SmartContract

// NewSCObject creates a new global SmartContract object instance
func NewSCObject() *SmartContract {
	if SCObject == nil {
		SCObject = &SmartContract{
			mtx:      sync.Mutex{},
			contract: map[string]sci.SmartContractInterface{},
		}
	}
	return SCObject
}

func (s *SmartContract) ExecuteRestAPI(ctx context.Context, cmd *SmartContractExecuteRest) (interface{}, error) {
	_, sc := s.getByAddr(cmd.Address)
	if sc != nil {
		//add bc context here
		handler, restpathok := sc.RestHandlers[cmd.Path]
		if !restpathok {
			return nil, common.NewError("invalid_path", "Invalid path")
		}
		return handler(ctx, cmd.Params, cmd.Balances)
	}
	return nil, common.NewError("invalid_sc", "Invalid Smart contract address")
}

func (s *SmartContract) ExecuteStats(ctx context.Context, data *SmartContractExecuteStats) {
	_, sc := s.getByAddr(data.Address)
	if sc != nil {
		int, err := sc.HandlerStats(ctx, data.Params)
		if err != nil {
			Logger.Warn("unexpected error", zap.Error(err))
		}
		fmt.Fprintf(data.Response, "%v", int)
		return
	}
	fmt.Fprintf(data.Response, "invalid_sc: Invalid Smart contract address")
}

func (s *SmartContract) getByAddr(addr string) (sci.SmartContractInterface, *sci.SmartContract) {
	contracti, ok := s.contract[addr]
	if ok {
		s.mtx.Lock()
		defer s.mtx.Unlock()
		sc := sci.NewSC(addr)
		bc := &BCContext{}
		contracti.SetSC(sc, bc)
		return contracti, sc
	}
	return nil, nil
}

func (s *SmartContract) Execute(ctx context.Context, t *transaction.Transaction, balances c_state.StateContextI) (string, error) {
	contractObj, contract := s.getByAddr(t.ToClientID)
	if contractObj != nil {
		var smartContractData sci.SmartContractTransactionData
		dataBytes := []byte(t.TransactionData)
		err := json.Unmarshal(dataBytes, &smartContractData)
		if err != nil {
			Logger.Error("Error while decoding the JSON from transaction", zap.Any("input", t.TransactionData), zap.Any("error", err))
			return "", err
		}
		// transactionOutput, err := contractObj.executeWithStats(t, smartContractData.FunctionName, []byte(smartContractData.InputData), balances)
		transactionOutput, err := s.executeWithStats(contractObj, contract, t, smartContractData.FunctionName, []byte(smartContractData.InputData), balances)
		if err != nil {
			return "", err
		}
		return transactionOutput, nil
	}
	return "", common.NewError("invalid_smart_contract_address", "Invalid Smart Contract address")
}

func (s *SmartContract) executeWithStats(smcoi sci.SmartContractInterface, sc *sci.SmartContract, t *transaction.Transaction, funcName string, input []byte, balances c_state.StateContextI) (string, error) {
	ts := time.Now()
	inter, err := smcoi.Execute(t, funcName, input, balances)
	if err == nil {
		if tm := sc.SmartContractExecutionStats[funcName]; tm != nil {
			if timer, ok := tm.(metrics.Timer); ok {
				timer.Update(time.Since(ts))
			}
		}
	}
	return inter, err
}
