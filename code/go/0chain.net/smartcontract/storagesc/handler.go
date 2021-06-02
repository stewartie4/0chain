package storagesc

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/0chain/0chain/code/go/0chain.net/smartcontract"

	"github.com/0chain/0chain/code/go/0chain.net/core/logging"

	cstate "github.com/0chain/0chain/code/go/0chain.net/chaincore/chain/state"
	"github.com/0chain/0chain/code/go/0chain.net/chaincore/state"
	"github.com/0chain/0chain/code/go/0chain.net/core/common"
	"github.com/0chain/0chain/code/go/0chain.net/core/util"
)

const cantGetBlobberMsg = "can't get blobber"

// GetBlobberHandler returns Blobber object from its individual stored value.
func (ssc *StorageSmartContract) GetBlobberHandler(ctx context.Context,
	params url.Values, balances cstate.StateContextI) (
	resp interface{}, err error) {

	var blobberID = params.Get("blobber_id")
	if blobberID == "" {
		return nil, common.NewErrBadRequest("missing 'blobber_id' URL query parameter")
	}

	bl, err := ssc.getBlobber(blobberID, balances)
	if err != nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get blobber")
	}

	return bl, nil
}

// GetBlobbersHandler returns list of all blobbers alive (e.g. excluding
// blobbers with zero capacity).
func (ssc *StorageSmartContract) GetBlobbersHandler(ctx context.Context,
	params url.Values, balances cstate.StateContextI) (interface{}, error) {

	blobbers, err := ssc.getBlobbersList(balances)
	if err != nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get blobbers list")
	}
	return blobbers, nil
}

func (ssc *StorageSmartContract) GetAllocationsHandler(ctx context.Context,
	params url.Values, balances cstate.StateContextI) (interface{}, error) {

	clientID := params.Get("client")
	allocations, err := ssc.getAllocationsList(clientID, balances)
	if err != nil {
		return nil, common.NewErrInternal("can't get allocation list", err.Error())
	}
	result := make([]*StorageAllocation, 0)
	for _, allocationID := range allocations.List {
		allocationObj := &StorageAllocation{}
		allocationObj.ID = allocationID

		allocationBytes, err := balances.GetTrieNode(allocationObj.GetKey(ssc.ID))
		if err != nil {
			continue
		}
		if err := allocationObj.Decode(allocationBytes.Encode()); err != nil {
			msg := fmt.Sprintf("can't decode allocation with id '%s'", allocationID)
			return nil, common.NewErrInternal(msg, err.Error())
		}
		result = append(result, allocationObj)
	}
	return result, nil
}

func (ssc *StorageSmartContract) GetAllocationMinLockHandler(ctx context.Context,
	params url.Values, balances cstate.StateContextI) (interface{}, error) {

	var err error
	var creationDate = common.Timestamp(time.Now().Unix())

	allocData := params.Get("allocation_data")
	var request newAllocationRequest
	if err = request.decode([]byte(allocData)); err != nil {
		return "", common.NewErrInternal("can't decode allocation request", err.Error())
	}

	var conf *scConfig
	if conf, err = ssc.getConfig(balances, true); err != nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, cantGetConfigErrMsg)
	}

	var allBlobbersList *StorageNodes
	allBlobbersList, err = ssc.getBlobbersList(balances)
	if err != nil || len(allBlobbersList.Nodes) == 0 {
		return "", smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get blobbers list")
	}

	var sa = request.storageAllocation() // (set fields, including expiration)
	sa.TimeUnit = conf.TimeUnit          // keep the initial time unit

	if err = sa.validate(creationDate, conf); err != nil {
		return "", common.NewErrBadRequest("allocation min lock failed", err.Error())
	}

	var (
		// number of blobbers required
		size = sa.DataShards + sa.ParityShards
		// size of allocation for a blobber
		bsize = (sa.Size + int64(size-1)) / int64(size)
		// filtered list
		list = sa.filterBlobbers(allBlobbersList.Nodes.copy(), creationDate,
			bsize, filterHealthyBlobbers(creationDate),
			ssc.filterBlobbersByFreeSpace(creationDate, bsize, balances))
	)

	if len(list) < size {
		return "", common.NewErrNoResource("not enough blobbers to honor the allocation")
	}

	sa.BlobberDetails = make([]*BlobberAllocation, 0)

	var blobberNodes []*StorageNode
	preferredBlobbersSize := len(sa.PreferredBlobbers)
	if preferredBlobbersSize > 0 {
		blobberNodes, err = getPreferredBlobbers(sa.PreferredBlobbers, list)
		if err != nil {
			return "", common.NewErrNoResource("can't get preferred blobbers", err.Error())
		}
	}

	// randomize blobber nodes
	if len(blobberNodes) < size {
		blobberNodes = randomizeNodes(list, blobberNodes, size, int64(creationDate))
	}

	blobberNodes = blobberNodes[:size]

	var gbSize = sizeInGB(bsize) // size in gigabytes
	var minLockDemand state.Balance
	for _, b := range blobberNodes {
		minLockDemand += b.Terms.minLockDemand(gbSize,
			sa.restDurationInTimeUnits(creationDate))
	}

	var response = map[string]interface{}{
		"min_lock_demand": minLockDemand,
	}

	return response, nil
}

const cantGetAllocation = "can't get allocation"

func (ssc *StorageSmartContract) AllocationStatsHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	allocationID := params.Get("allocation")
	allocationObj := &StorageAllocation{}
	allocationObj.ID = allocationID

	allocationBytes, err := balances.GetTrieNode(allocationObj.GetKey(ssc.ID))
	if err != nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, cantGetAllocation)
	}
	err = allocationObj.Decode(allocationBytes.Encode())
	if err != nil {
		return nil, common.NewErrInternal("can't decode allocation", err.Error())
	}
	return allocationObj, nil
}

func (ssc *StorageSmartContract) LatestReadMarkerHandler(ctx context.Context,
	params url.Values, balances cstate.StateContextI) (
	resp interface{}, err error) {

	var (
		clientID  = params.Get("client")
		blobberID = params.Get("blobber")

		commitRead = &ReadConnection{}
	)

	commitRead.ReadMarker = &ReadMarker{
		BlobberID: blobberID,
		ClientID:  clientID,
	}

	var commitReadBytes util.Serializable
	commitReadBytes, err = balances.GetTrieNode(commitRead.GetKey(ssc.ID))
	if err != nil && err != util.ErrValueNotPresent {
		return nil, common.NewErrInternal("can't get read marker", err.Error())
	}

	if commitReadBytes == nil {
		return make(map[string]string), nil
	}

	if err = commitRead.Decode(commitReadBytes.Encode()); err != nil {
		return nil, common.NewErrInternal("can't decode read marker", err.Error())
	}

	return commitRead.ReadMarker, nil // ok

}

func (ssc *StorageSmartContract) OpenChallengeHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	blobberID := params.Get("blobber")
	blobberChallengeObj := &BlobberChallenge{}
	blobberChallengeObj.BlobberID = blobberID
	blobberChallengeObj.Challenges = make([]*StorageChallenge, 0)

	blobberChallengeBytes, err := balances.GetTrieNode(blobberChallengeObj.GetKey(ssc.ID))
	if err != nil {
		return "", smartcontract.NewErrNoResourceOrErrInternal(err, true, "error reading blobber challenge from DB")
	}
	err = blobberChallengeObj.Decode(blobberChallengeBytes.Encode())
	if err != nil {
		return nil, common.NewErrInternal("fail decoding blobber challenge", err.Error())
	}

	// for k, v := range blobberChallengeObj.ChallengeMap {
	// 	if v.Response != nil {
	// 		delete(blobberChallengeObj.ChallengeMap, k)
	// 	}
	// }

	return &blobberChallengeObj, nil
}

func (ssc *StorageSmartContract) GetChallengeHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (retVal interface{}, retErr error) {
	defer func() {
		if retErr != nil {
			logging.Logger.Error("/getchallenge failed with error - " + retErr.Error())
		}
	}()
	blobberID := params.Get("blobber")
	blobberChallengeObj := &BlobberChallenge{}
	blobberChallengeObj.BlobberID = blobberID
	blobberChallengeObj.Challenges = make([]*StorageChallenge, 0)

	blobberChallengeBytes, err := balances.GetTrieNode(blobberChallengeObj.GetKey(ssc.ID))
	if err != nil {
		return "", smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get blobber challenge")
	}
	if err := blobberChallengeObj.Decode(blobberChallengeBytes.Encode()); err != nil {
		return "", common.NewErrInternal("can't decode blobber challenge", err.Error())
	}

	challengeID := params.Get("challenge")
	if _, ok := blobberChallengeObj.ChallengeMap[challengeID]; !ok {
		return nil, common.NewErrBadRequest("can't find challenge with provided 'challenge' param")
	}

	return blobberChallengeObj.ChallengeMap[challengeID], nil
}
