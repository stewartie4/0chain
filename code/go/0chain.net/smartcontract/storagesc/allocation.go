package storagesc

import (
	"encoding/json"
	"sort"

	"0chain.net/chaincore/client"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
)

func (sc *StorageSmartContract) getAllocationsList(clientID string) (*Allocations, error) {
	allocationList := &Allocations{}
	var clientAlloc ClientAllocation
	clientAlloc.ClientID = clientID
	allocationListBytes, err := sc.GetNode(clientAlloc.GetKey())
	if allocationListBytes == nil {
		return allocationList, nil
	}
	err = json.Unmarshal(allocationListBytes.Encode(), &clientAlloc)
	if err != nil {
		return nil, common.NewError("getAllocationsList_failed", "Failed to retrieve existing allocations list")
	}
	return clientAlloc.Allocations, nil
}

func (sc *StorageSmartContract) getAllAllocationsList() (*Allocations, error) {
	allocationList := &Allocations{}

	allocationListBytes, err := sc.GetNode(ALL_ALLOCATIONS_KEY)
	if allocationListBytes == nil {
		return allocationList, nil
	}
	err = json.Unmarshal(allocationListBytes.Encode(), allocationList)
	if err != nil {
		return nil, common.NewError("getAllAllocationsList_failed", "Failed to retrieve existing allocations list")
	}
	sort.SliceStable(allocationList.List, func(i, j int) bool {
		return allocationList.List[i] < allocationList.List[j]
	})
	return allocationList, nil
}

func (sc *StorageSmartContract) addAllocation(allocation *StorageAllocation) (string, error) {
	allocationList, err := sc.getAllocationsList(allocation.Owner)
	if err != nil {
		return "", common.NewError("add_allocation_failed", "Failed to get allocation list"+err.Error())
	}
	allAllocationList, err := sc.getAllAllocationsList()
	if err != nil {
		return "", common.NewError("add_allocation_failed", "Failed to get allocation list"+err.Error())
	}

	allocationBytes, _ := sc.GetNode(allocation.GetKey())
	if allocationBytes == nil {
		allocationList.List = append(allocationList.List, allocation.ID)
		allAllocationList.List = append(allAllocationList.List, allocation.ID)
		clientAllocation := &ClientAllocation{}
		clientAllocation.ClientID = allocation.Owner
		clientAllocation.Allocations = allocationList

		// allAllocationBytes, _ := json.Marshal(allAllocationList)
		sc.InsertNode(ALL_ALLOCATIONS_KEY, allAllocationList)
		sc.InsertNode(clientAllocation.GetKey(), clientAllocation)
		sc.InsertNode(allocation.GetKey(), allocation)
	}

	buff := allocation.Encode()
	return string(buff), nil
}

func (sc *StorageSmartContract) newAllocationRequest(t *transaction.Transaction, input []byte) (string, error) {
	allBlobbersList, err := sc.getBlobbersList()
	if err != nil {
		return "", common.NewError("allocation_creation_failed", "No Blobbers registered. Failed to create a storage allocation")
	}

	if len(allBlobbersList.Nodes) == 0 {
		return "", common.NewError("allocation_creation_failed", "No Blobbers registered. Failed to create a storage allocation")
	}

	if len(t.ClientID) == 0 {
		return "", common.NewError("allocation_creation_failed", "Invalid client in the transaction. No public key found")
	}

	clientPublicKey := t.PublicKey
	if len(t.PublicKey) == 0 {
		ownerClient, err := client.GetClient(common.GetRootContext(), t.ClientID)
		if err != nil || ownerClient == nil || len(ownerClient.PublicKey) == 0 {
			return "", common.NewError("invalid_client", "Invalid Client. Not found with miner")
		}
		clientPublicKey = ownerClient.PublicKey
	}

	var allocationRequest StorageAllocation

	err = allocationRequest.Decode(input)
	if err != nil {
		return "", common.NewError("allocation_creation_failed", "Failed to create a storage allocation")
	}

	if allocationRequest.Size > 0 && allocationRequest.DataShards > 0 {
		size := allocationRequest.DataShards + allocationRequest.ParityShards

		if len(allBlobbersList.Nodes) < size {
			return "", common.NewError("not_enough_blobbers", "Not enough blobbers to honor the allocation")
		}

		allocatedBlobbers := make([]*StorageNode, 0)
		allocationRequest.BlobberDetails = make([]*BlobberAllocation, 0)
		allocationRequest.Stats = &StorageAllocationStats{}

		for i := 0; i < size; i++ {
			blobberNode := allBlobbersList.Nodes[i]
			var blobberAllocation BlobberAllocation
			blobberAllocation.Stats = &StorageAllocationStats{}
			blobberAllocation.Size = (allocationRequest.Size + int64(size-1)) / int64(size)
			blobberAllocation.AllocationID = t.Hash
			blobberAllocation.BlobberID = blobberNode.ID

			allocationRequest.BlobberDetails = append(allocationRequest.BlobberDetails, &blobberAllocation)
			allocatedBlobbers = append(allocatedBlobbers, blobberNode)
		}

		sort.SliceStable(allocatedBlobbers, func(i, j int) bool {
			return allocatedBlobbers[i].ID < allocatedBlobbers[j].ID
		})

		allocationRequest.Blobbers = allocatedBlobbers
		allocationRequest.ID = t.Hash
		allocationRequest.Owner = t.ClientID
		allocationRequest.OwnerPublicKey = clientPublicKey

		buff, err := sc.addAllocation(&allocationRequest)
		if err != nil {
			return "", common.NewError("allocation_request_failed", "Failed to store the allocation request")
		}
		return buff, nil
	}
	return "", common.NewError("invalid_allocation_request", "Failed storage allocate")
}
