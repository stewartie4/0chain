package storagesc

import (
	"encoding/json"
	"math/rand"
	"sort"

	"0chain.net/chaincore/block"
	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

func (sc *StorageSmartContract) completeChallengeForBlobber(blobberChallengeObj *BlobberChallenge, challengeCompleted *StorageChallenge, challengeResponse *ChallengeResponse) {
	found := false
	idx := -1
	for i, challenge := range blobberChallengeObj.Challenges {
		if challenge.ID == challengeCompleted.ID {
			found = true
			idx = i
			break
		}
	}
	if found && idx >= 0 && idx < len(blobberChallengeObj.Challenges) {
		blobberChallengeObj.Challenges = append(blobberChallengeObj.Challenges[:idx], blobberChallengeObj.Challenges[idx+1:]...)
		if len(blobberChallengeObj.LatestCompletedChallenges) >= 20 {
			startIndex := (20 - len(blobberChallengeObj.LatestCompletedChallenges)) + 1
			blobberChallengeObj.LatestCompletedChallenges = blobberChallengeObj.LatestCompletedChallenges[startIndex:]
		}
		challengeCompleted.Response = challengeResponse
		blobberChallengeObj.LatestCompletedChallenges = append(blobberChallengeObj.LatestCompletedChallenges, challengeCompleted)
	}

}

func (sc *StorageSmartContract) verifyChallenge(t *transaction.Transaction, input []byte, balances c_state.StateContextI) (string, error) {
	var challengeResponse ChallengeResponse
	err := json.Unmarshal(input, &challengeResponse)
	if err != nil {
		return "", err
	}
	if len(challengeResponse.ID) == 0 || len(challengeResponse.ValidationTickets) == 0 {
		return "", common.NewError("invalid_parameters", "Invalid parameters to challenge response")
	}

	blobberChallengeObj := &BlobberChallenge{}
	blobberChallengeObj.BlobberID = t.ClientID

	blobberChallengeBytes, err := balances.GetTrieNode(blobberChallengeObj.GetKey(sc.ID))
	if blobberChallengeBytes == nil {
		return "", common.NewError("invalid_parameters", "Cannot find the blobber challenge entity with ID "+t.ClientID)
	}

	err = blobberChallengeObj.Decode(blobberChallengeBytes.Encode())
	if err != nil {
		return "", common.NewError("blobber_challenge_decode_error", "Error decoding the blobber challenge")
	}

	challengeRequest, ok := blobberChallengeObj.ChallengeMap[challengeResponse.ID]

	if !ok {
		for _, completedChallenge := range blobberChallengeObj.LatestCompletedChallenges {
			if challengeResponse.ID == completedChallenge.ID && completedChallenge.Response != nil {
				return "Challenge Already redeemed by Blobber", nil
			}
		}
		return "", common.NewError("invalid_parameters", "Cannot find the challenge with ID "+challengeResponse.ID)
	}

	if challengeRequest.Blobber.ID != t.ClientID {
		return "", common.NewError("invalid_parameters", "Challenge response should be submitted by the same blobber as the challenge request")
	}

	allocationObj := NewStorageAllocation()
	allocationObj.ID = challengeRequest.AllocationID

	allocationBytes, err := balances.GetTrieNode(allocationObj.GetKey(sc.ID))
	if allocationBytes == nil || err != nil {
		return "", common.NewError("invalid_allocation", "Client state has invalid allocations")
	}

	err = allocationObj.Decode(allocationBytes.Encode())
	if err != nil {
		return "", common.NewError("decode_error", "Error decoding the allocation")
	}

	blobberAllocation, ok := allocationObj.BlobberMap[t.ClientID]
	if !ok {
		return "", common.NewError("invalid_parameters", "Blobber is not part of the allocation")
	}

	numSuccess := 0
	numFailure := 0
	for _, vt := range challengeResponse.ValidationTickets {
		if vt != nil {
			ok, err := vt.VerifySign()
			if !ok || err != nil {
				continue
			}
			if vt.Result {
				numSuccess++
			} else {
				numFailure++
			}
		}
	}

	sn := &StorageNode{ID: t.ClientID}
	storageNodeBytes, err := balances.GetTrieNode(sn.GetKey(sc.ID))
	if storageNodeBytes == nil || err != nil {
		return "", common.NewError("invalid_parameters", "Invalid blobber ID")
	}
	err = sn.Decode(storageNodeBytes.Encode())
	if err != nil {
		return "", common.NewError("invalid_parameters", "Failed to decode from DB")
	}
	if numSuccess > (len(challengeRequest.Validators) / 2) {
		//challengeRequest.Response = &challengeResponse
		//delete(blobberChallengeObj.ChallengeMap, challengeResponse.ID)
		sc.completeChallengeForBlobber(blobberChallengeObj, challengeRequest, &challengeResponse)
		allocationObj.Stats.LastestClosedChallengeTxn = challengeRequest.ID
		allocationObj.Stats.SuccessChallenges++
		allocationObj.Stats.OpenChallenges--

		blobberAllocation.Stats.LastestClosedChallengeTxn = challengeRequest.ID
		blobberAllocation.Stats.SuccessChallenges++
		blobberAllocation.Stats.OpenChallenges--
		validatorFee := state.Balance(0)
		totalFee := state.Balance(0)
		for _, pool := range blobberAllocation.ChallengePools {
			totalFee += pool.Balance
			allocationObj.AmountPaid += pool.Balance
			validatorFee += state.Balance(float64(pool.Balance) * sn.ValidatorPercentage)
		}
		blobberAllocation.ChallengePools = make(map[string]*tokenpool.ZcnLockingPool)
		vns, err := sc.getValidatorsList(balances)
		if err != nil {
			return "", err
		}
		indivValidatorFee := validatorFee / state.Balance(len(vns.Nodes))
		for _, validator := range vns.Nodes {
			balances.AddTransfer(state.NewTransfer(sc.ID, validator.DelegateID, indivValidatorFee))
		}
		balances.AddTransfer(state.NewTransfer(sc.ID, sn.DelegateID, totalFee-(indivValidatorFee*state.Balance(len(vns.Nodes)))))
		balances.InsertTrieNode(allocationObj.GetKey(sc.ID), allocationObj)
		balances.InsertTrieNode(blobberChallengeObj.GetKey(sc.ID), blobberChallengeObj)
		Logger.Info("Challenge passed", zap.Any("challenge", challengeResponse.ID))
		return "Challenge Passed by Blobber", nil
	}

	if numFailure > (len(challengeRequest.Validators) / 2) {
		sc.completeChallengeForBlobber(blobberChallengeObj, challengeRequest, &challengeResponse)
		//delete(blobberChallengeObj.ChallengeMap, challengeResponse.ID)
		//challengeRequest.Response = &challengeResponse
		allocationObj.Stats.LastestClosedChallengeTxn = challengeRequest.ID
		allocationObj.Stats.FailedChallenges++
		allocationObj.Stats.OpenChallenges--

		blobberAllocation.Stats.LastestClosedChallengeTxn = challengeRequest.ID
		blobberAllocation.Stats.FailedChallenges++
		blobberAllocation.Stats.OpenChallenges--

		for _, pool := range blobberAllocation.ChallengePools {
			_, _, err := pool.TransferTo(allocationObj.Pool, pool.Balance, nil)
			if err != nil {
				return "", err
			}
			_, _, err = sn.StakePool.TransferTo(allocationObj.Pool, pool.Balance*state.Balance(sn.StakeMultiplyer-1), nil)
			if err != nil {
				return "", err
			}
		}
		blobberAllocation.ChallengePools = make(map[string]*tokenpool.ZcnLockingPool)
		balances.InsertTrieNode(sn.GetKey(sc.ID), sn)
		balances.InsertTrieNode(allocationObj.GetKey(sc.ID), allocationObj)
		balances.InsertTrieNode(blobberChallengeObj.GetKey(sc.ID), blobberChallengeObj)
		Logger.Info("Challenge failed", zap.Any("challenge", challengeResponse.ID))
		return "Challenge Failed by Blobber", nil
	}

	return "", common.NewError("not_enough_validations", "Not enough validations for the challenge")
}

func (sc *StorageSmartContract) addChallenge(t *transaction.Transaction, b *block.Block, input []byte, balances c_state.StateContextI) (string, error) {

	validatorList, _ := sc.getValidatorsList(balances)

	if len(validatorList.Nodes) == 0 {
		return "", common.NewError("no_validators", "Not enough validators for the challenge")
	}

	foundValidator := false
	for _, validator := range validatorList.Nodes {
		if validator.ID == t.ClientID {
			foundValidator = true
			break
		}
	}

	if !foundValidator {
		return "", common.NewError("invalid_challenge_request", "Challenge can be requested only by validators")
	}

	var storageChallenge StorageChallenge
	storageChallenge.ID = t.Hash

	allocationList, err := sc.getAllAllocationsList(balances)
	if err != nil {
		return "", common.NewError("adding_challenge_error", "Error gettting the allocation list. "+err.Error())
	}
	if len(allocationList.List) == 0 {
		return "", common.NewError("adding_challenge_error", "No allocations at this time")
	}

	rand.Seed(b.RoundRandomSeed)
	allocationIndex := rand.Int63n(int64(len(allocationList.List)))
	allocationKey := allocationList.List[allocationIndex]

	allocationObj := NewStorageAllocation()
	allocationObj.ID = allocationKey

	allocationBytes, err := balances.GetTrieNode(allocationObj.GetKey(sc.ID))
	if allocationBytes == nil || err != nil {
		return "", common.NewError("invalid_allocation", "Client state has invalid allocations")
	}

	allocationObj.Decode(allocationBytes.Encode())
	sort.SliceStable(allocationObj.Blobbers, func(i, j int) bool {
		return allocationObj.Blobbers[i].ID < allocationObj.Blobbers[j].ID
	})

	rand.Seed(b.RoundRandomSeed)
	randIdx := rand.Int63n(int64(len(allocationObj.Blobbers)))
	Logger.Info("Challenge blobber selected.", zap.Any("challenge", t.Hash), zap.Any("selected_blobber", allocationObj.Blobbers[randIdx]), zap.Any("blobbers", allocationObj.Blobbers), zap.Any("random_index", randIdx), zap.Any("seed", b.RoundRandomSeed))

	storageChallenge.Validators = validatorList.Nodes
	storageChallenge.Blobber = allocationObj.Blobbers[randIdx]
	storageChallenge.RandomNumber = b.RoundRandomSeed
	storageChallenge.AllocationID = allocationObj.ID

	blobberAllocation, ok := allocationObj.BlobberMap[storageChallenge.Blobber.ID]

	if !ok {
		return "", common.NewError("invalid_parameters", "Blobber is not part of the allocation. Could not find blobber")
	}

	if len(blobberAllocation.AllocationRoot) == 0 || blobberAllocation.Stats.UsedSize == 0 {
		return "", common.NewError("blobber_no_wm", "Blobber does not have any data for the allocation. "+allocationObj.ID+" blobber: "+blobberAllocation.BlobberID)
	}

	storageChallenge.AllocationRoot = blobberAllocation.AllocationRoot

	blobberChallengeObj := &BlobberChallenge{}
	blobberChallengeObj.BlobberID = storageChallenge.Blobber.ID

	blobberChallengeBytes, err := balances.GetTrieNode(blobberChallengeObj.GetKey(sc.ID))
	blobberChallengeObj.LatestCompletedChallenges = make([]*StorageChallenge, 0)
	if blobberChallengeBytes != nil {
		err = blobberChallengeObj.Decode(blobberChallengeBytes.Encode())
		if err != nil {
			return "", common.NewError("blobber_challenge_decode_error", "Error decoding the blobber challenge")
		}
	}

	storageChallenge.Created = t.CreationDate
	addedChallege := blobberChallengeObj.addChallenge(&storageChallenge)
	if !addedChallege {
		challengeBytes, err := json.Marshal(storageChallenge)
		return string(challengeBytes), err
	}

	balances.InsertTrieNode(blobberChallengeObj.GetKey(sc.ID), blobberChallengeObj)

	allocationObj.Stats.OpenChallenges++
	allocationObj.Stats.TotalChallenges++
	blobberAllocation.UpdatePools()
	blobberAllocation.Stats.OpenChallenges++
	blobberAllocation.Stats.TotalChallenges++
	balances.InsertTrieNode(allocationObj.GetKey(sc.ID), allocationObj)
	//Logger.Info("Adding a new challenge", zap.Any("blobberChallengeObj", blobberChallengeObj), zap.Any("challenge", storageChallenge.ID))
	challengeBytes, err := json.Marshal(storageChallenge)
	return string(challengeBytes), err
}
