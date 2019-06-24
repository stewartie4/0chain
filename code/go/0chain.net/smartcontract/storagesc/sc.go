package storagesc

import (
	"fmt"

	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	metrics "github.com/rcrowley/go-metrics"
)

const (
	ADDRESS         = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
	name            = "storage"
	STAKEMULTIPLYER = 10
	MINPERCENT      = 0.01
	BLOCK           = state.Balance(64000)
)

type StorageSmartContract struct {
	*smartcontractinterface.SmartContract
}

func (ssc *StorageSmartContract) SetSC(sc *smartcontractinterface.SmartContract, bcContext smartcontractinterface.BCContextI) {
	ssc.SmartContract = sc
	ssc.SmartContract.RestHandlers["/allocation"] = ssc.AllocationStatsHandler
	ssc.SmartContract.RestHandlers["/latestreadmarker"] = ssc.LatestReadMarkerHandler
	ssc.SmartContract.RestHandlers["/openchallenges"] = ssc.OpenChallengeHandler
	ssc.SmartContractExecutionStats["read_redeem"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "read_redeem"), nil)
	ssc.SmartContractExecutionStats["commit_connection"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "commit_connection"), nil)
	ssc.SmartContractExecutionStats["new_allocation_request"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "new_allocation_request"), nil)
	ssc.SmartContractExecutionStats["add_blobber"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "add_blobber"), nil)
	ssc.SmartContractExecutionStats["add_validator"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "add_validator"), nil)
	ssc.SmartContractExecutionStats["challenge_request"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "challenge_request"), nil)
	ssc.SmartContractExecutionStats["challenge_response"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "challenge_response"), nil)
}

func (ssc *StorageSmartContract) GetName() string {
	return name
}

func (ssc *StorageSmartContract) GetAddress() string {
	return ADDRESS
}

func (ssc *StorageSmartContract) GetRestPoints() map[string]smartcontractinterface.SmartContractRestHandler {
	return ssc.RestHandlers
}

func (sc *StorageSmartContract) Execute(t *transaction.Transaction, funcName string, input []byte, balances c_state.StateContextI) (string, error) {
	switch funcName {
	case "read_redeem":
		return sc.commitBlobberRead(t, input, balances)
	case "commit_connection":
		return sc.commitBlobberConnection(t, input, balances)
	case "new_allocation_request":
		return sc.newAllocationRequest(t, input, balances)
	case "reclaim_allocation_stake":
		return sc.reclaimAllocationStake(t, input, balances)
	case "stake_for_blobber":
		return sc.stakeForBlobber(t, input, balances)
	case "drain_stake_for_blobber":
		return sc.drainStakeForBlobber(t, input, balances)
	case "add_blobber":
		return sc.addBlobber(t, input, balances)
	case "remove_blobber":
		return sc.removeBlobber(t, input, balances)
	case "add_validator":
		return sc.addValidator(t, input, balances)
	case "challenge_request":
		return sc.addChallenge(t, balances.GetBlock(), input, balances)
	case "challenge_response":
		return sc.verifyChallenge(t, input, balances)
	default:
		return "", common.NewError("failed execution", "no function with that name")
	}
}
