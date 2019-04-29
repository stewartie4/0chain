package storagesc

import (
	"context"
	// "encoding/json"
	"net/url"

	"0chain.net/core/common"
)

func (ssc *StorageSmartContract) AllocationStatsHandler(ctx context.Context, params url.Values) (interface{}, error) {
	allocationID := params.Get("allocation")
	allocationObj := &StorageAllocation{}
	allocationObj.ID = allocationID

	allocationBytes, err := ssc.GetNode(allocationObj.GetKey())
	if err != nil {
		return nil, err
	}
	allocationObj.Decode(allocationBytes.Encode())
	return allocationObj, err
}

func (ssc *StorageSmartContract) LatestReadMarkerHandler(ctx context.Context, params url.Values) (interface{}, error) {
	clientID := params.Get("client")
	blobberID := params.Get("blobber")
	commitRead := &ReadConnection{}
	commitRead.ReadMarker = &ReadMarker{BlobberID: blobberID, ClientID: clientID}

	commitReadBytes, err := ssc.GetNode(commitRead.GetKey())
	if err != nil {
		return nil, err
	}
	if commitReadBytes == nil {
		return make(map[string]string), nil
	}
	commitRead.Decode(commitReadBytes.Encode())

	return commitRead.ReadMarker, err

}

func (ssc *StorageSmartContract) OpenChallengeHandler(ctx context.Context, params url.Values) (interface{}, error) {
	blobberID := params.Get("blobber")
	blobberChallengeObj := &BlobberChallenge{}
	blobberChallengeObj.BlobberID = blobberID
	blobberChallengeObj.Challenges = make([]*StorageChallenge, 0)

	blobberChallengeBytes, err := ssc.GetNode(blobberChallengeObj.GetKey())
	if err != nil {
		return "", common.NewError("blobber_challenge_read_err", "Error reading blobber challenge from DB. "+err.Error())
	}
	blobberChallengeObj.Decode(blobberChallengeBytes.Encode())

	// for k, v := range blobberChallengeObj.ChallengeMap {
	// 	if v.Response != nil {
	// 		delete(blobberChallengeObj.ChallengeMap, k)
	// 	}
	// }

	return &blobberChallengeObj, err
}
