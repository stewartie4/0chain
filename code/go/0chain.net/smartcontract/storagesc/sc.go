package storagesc

import (
	"fmt"
	"time"

	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	metrics "github.com/rcrowley/go-metrics"
)

const (
	ADDRESS = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
	name    = "storage"
	BLOCK   = state.Balance(64000)
)

var (
	// everything below will eventually be taken over by governance sc
	USDPRICEPERTOKEN        = 0.99                            // individual price for 1 zcn
	WORKINGVALIDATORPERCENT = 0.6                             // percent of validation fee that is taken by validators whose validation ticket is in the challenge response
	INTERESTRATE            = 0.1                             // interest paid to blobber for staking
	STAKEMULTIPLYER         = int64(10)                       // multiplyer staked capacity to ensure blobbers have skin in the game
	MINPERCENT              = 0.01                            // percent of total write cost for an allocation (will be given to blobber if no reads or writes are preformed)
	MAXLOCKPERIOD           = time.Duration(time.Hour * 8760) // one year's worth of hours
)

type StorageSmartContract struct {
	*smartcontractinterface.SmartContract
}

func (ssc *StorageSmartContract) SetSC(sc *smartcontractinterface.SmartContract, bcContext smartcontractinterface.BCContextI) {
	ssc.SmartContract = sc
	ssc.SmartContract.RestHandlers["/allocation"] = ssc.AllocationStatsHandler
	ssc.SmartContract.RestHandlers["/latestreadmarker"] = ssc.LatestReadMarkerHandler
	ssc.SmartContract.RestHandlers["/openchallenges"] = ssc.OpenChallengeHandler
	ssc.SmartContract.RestHandlers["/blobberPricePoints"] = ssc.BlobberPricePoints
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
	case "adust_usd_percent":
		return sc.adjustUSDPercent(t, input, balances)
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
