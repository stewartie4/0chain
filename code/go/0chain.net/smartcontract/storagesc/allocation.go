package storagesc

import (
	"encoding/json"
	"fmt"
	"sort"

	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/client"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
)

func (sc *StorageSmartContract) getAllocationsList(clientID string, balances c_state.StateContextI) (*Allocations, error) {
	allocationList := &Allocations{}
	var clientAlloc ClientAllocation
	clientAlloc.ClientID = clientID
	allocationListBytes, err := balances.GetTrieNode(clientAlloc.GetKey(sc.ID))
	if allocationListBytes == nil {
		return allocationList, nil
	}
	err = json.Unmarshal(allocationListBytes.Encode(), &clientAlloc)
	if err != nil {
		return nil, common.NewError("getAllocationsList_failed", "Failed to retrieve existing allocations list")
	}
	return clientAlloc.Allocations, nil
}

func (sc *StorageSmartContract) getAllAllocationsList(balances c_state.StateContextI) (*Allocations, error) {
	allocationList := &Allocations{}

	allocationListBytes, err := balances.GetTrieNode(ALL_ALLOCATIONS_KEY)
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

func (sc *StorageSmartContract) addAllocation(allocation *StorageAllocation, balances c_state.StateContextI) (string, error) {
	allBlobbersList, err := sc.getBlobbersList(balances)
	if err != nil {
		return "", err
	}
	allocationList, err := sc.getAllocationsList(allocation.Owner, balances)
	if err != nil {
		return "", common.NewError("add_allocation_failed", "Failed to get allocation list"+err.Error())
	}
	allAllocationList, err := sc.getAllAllocationsList(balances)
	if err != nil {
		return "", common.NewError("add_allocation_failed", "Failed to get allocation list"+err.Error())
	}

	allocationBytes, _ := balances.GetTrieNode(allocation.GetKey(sc.ID))
	if allocationBytes == nil {
		for _, blobberDetail := range allocation.BlobberDetails {
			blobber := &StorageNode{ID: blobberDetail.BlobberID}
			blobberBytes, _ := balances.GetTrieNode(blobber.GetKey(sc.ID))
			err := blobber.Decode(blobberBytes.Encode())
			if err != nil {
				return "", common.NewError("add_allocation_failed", "Failed to decode blobberBytes: "+err.Error())
			}
			if blobber.LongestCommitment < allocation.Expiration {
				blobber.LongestCommitment = allocation.Expiration
			}
			blobberDetail.StakedPool.ID = balances.GetTransaction().Hash
			blobber.StakePool.TransferTo(blobberDetail.StakedPool, state.Balance(blobberDetail.Size/blobber.WriteRatio.Size*blobber.WriteRatio.ZCN), nil)
			blobber.AllocationCapacity += blobberDetail.Size
			blobber.Allocations.List = append(blobber.Allocations.List, allocation.ID)
			err = allBlobbersList.UpdateStorageNode(blobber)
			if err != nil {
				return "", common.NewError("add_allocation_failed", "Failed to update blobber: "+err.Error())
			}
			balances.InsertTrieNode(blobber.GetKey(sc.ID), blobber)
		}

		allocationList.List = append(allocationList.List, allocation.ID)
		allAllocationList.List = append(allAllocationList.List, allocation.ID)
		clientAllocation := &ClientAllocation{}
		clientAllocation.ClientID = allocation.Owner
		clientAllocation.Allocations = allocationList

		balances.InsertTrieNode(ALL_ALLOCATIONS_KEY, allAllocationList)
		balances.InsertTrieNode(clientAllocation.GetKey(sc.ID), clientAllocation)
		balances.InsertTrieNode(allocation.GetKey(sc.ID), allocation)
		balances.InsertTrieNode(ALL_BLOBBERS_KEY, allBlobbersList)
	}

	buff := allocation.Encode()
	return string(buff), nil
}

func (sc *StorageSmartContract) deleteAllocation(allocation *StorageAllocation, balances c_state.StateContextI) (string, error) {
	allBlobbersList, err := sc.getBlobbersList(balances)
	if err != nil {
		return "", err
	}
	allocationList, err := sc.getAllocationsList(allocation.Owner, balances)
	if err != nil {
		return "", common.NewError("delete_allocation_failed", "Failed to get client allocation list"+err.Error())
	}
	allAllocationList, err := sc.getAllAllocationsList(balances)
	if err != nil {
		return "", common.NewError("delete_allocation_failed", "Failed to get global allocation list"+err.Error())
	}

	allocationBytes, _ := balances.GetTrieNode(allocation.GetKey(sc.ID))
	if allocationBytes == nil {
		return "", common.NewError("delete_allocation_failed", "client allocation does not exist")
	}
	err = allAllocationList.DeleteAllocation(allocation.ID)
	if err != nil {
		return "", common.NewError("delete_allocation_failed", "failed to delete allocation from global list"+err.Error())
	}
	err = allocationList.DeleteAllocation(allocation.ID)
	if err != nil {
		return "", common.NewError("delete_allocation_failed", "failed to delete allocation from client list"+err.Error())
	}
	allDuration := common.ToTime(allocation.Expiration).Sub(common.ToTime(allocation.StartDate))
	for _, blobberDetail := range allocation.BlobberDetails {
		blobber := &StorageNode{ID: blobberDetail.BlobberID}
		blobberBytes, _ := balances.GetTrieNode(blobber.GetKey(sc.ID))
		err := blobber.Decode(blobberBytes.Encode())
		if err != nil {
			return "", common.NewError("delete_allocation_failed", "failed to decode blobber bytes"+err.Error())
		}
		_, _, err = blobberDetail.StakedPool.TransferTo(blobber.StakePool, blobberDetail.StakedPool.Balance, nil)
		if err != nil {
			return "", common.NewError("delete_allocation_failed", "failed to transfer blobber pool"+err.Error())
		}
		mintAmount := state.Balance(INTERESTRATE * float64(allDuration) / float64(MAXLOCKPERIOD) * float64(blobberDetail.StakedPool.Balance))
		balances.AddMint(state.NewMint(sc.ID, blobber.DelegateID, mintAmount))
		blobber.AllocationCapacity -= blobberDetail.Size
		blobber.Allocations.DeleteAllocation(allocation.ID)
		err = allBlobbersList.UpdateStorageNode(blobber)
		if err != nil {
			return "", common.NewError("add_allocation_failed", "Failed to update blobber: "+err.Error())
		}
		balances.InsertTrieNode(blobber.GetKey(sc.ID), blobber)
	}
	clientAllocation := &ClientAllocation{}
	clientAllocation.ClientID = allocation.Owner
	clientAllocation.Allocations = allocationList
	balances.InsertTrieNode(ALL_ALLOCATIONS_KEY, allAllocationList)
	balances.InsertTrieNode(ALL_BLOBBERS_KEY, allBlobbersList)
	if len(clientAllocation.Allocations.List) > 0 {
		balances.InsertTrieNode(clientAllocation.GetKey(sc.ID), clientAllocation)
	}
	balances.DeleteTrieNode(allocation.GetKey(sc.ID))
	return string(allocation.Encode()), nil
}

func (sc *StorageSmartContract) newAllocationRequest(t *transaction.Transaction, input []byte, balances c_state.StateContextI) (string, error) {
	allBlobbersList, err := sc.getBlobbersList(balances)
	if err != nil || len(allBlobbersList.Nodes) == 0 {
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

	allocationRequest := NewStorageAllocation()
	err = allocationRequest.Decode(input)
	if err != nil {
		return "", common.NewError("allocation_creation_failed", "Failed to create a storage allocation")
	}
	if allocationRequest.Expiration < t.CreationDate {
		return "", common.NewError("allocation_creation_failed", "allocation has already expired")
	}
	if !(allocationRequest.ReadRatio.Validate() && allocationRequest.WriteRatio.Validate()) {
		return "", common.NewError("allocation_creation_failed", fmt.Sprintf("read (%v) or write (%v) marker is not valid", allocationRequest.ReadRatio, allocationRequest.WriteRatio))
	}
	if float64(allocationRequest.Size*allocationRequest.WriteRatio.ZCN)/float64(allocationRequest.WriteRatio.Size)*MINPERCENT > float64(t.Value) {
		return "", common.NewError("allocation_creation_failed", fmt.Sprintf("Insufficent funds (%v) for requested size (%v) and write ratio (%v)", t.Value, allocationRequest.Size, string(allocationRequest.WriteRatio.Encode())))
	}
	if allocationRequest.Size > 0 && allocationRequest.DataShards > 0 {
		size := allocationRequest.DataShards + allocationRequest.ParityShards
		storageNodes := &StorageNodes{}
		for _, b := range allBlobbersList.Nodes {
			if b.StakedCapacity-b.AllocationCapacity > (allocationRequest.Size+int64(size-1))/int64(size) {
				storageNodes.Nodes = append(storageNodes.Nodes, b)
			}
		}
		// TODO: come up with better way to narrow down blobbers for an allocation
		sort.Slice(storageNodes.Nodes, func(i, j int) bool {
			return storageNodes.Nodes[i].WriteRatio.GetRatio() < storageNodes.Nodes[j].WriteRatio.GetRatio()
		})
		if len(storageNodes.Nodes) < size {
			return "", common.NewError("not_enough_blobbers", "Not enough blobbers to honor the allocation")
		}

		allocatedBlobbers := make([]*StorageNode, 0)
		allocationRequest.BlobberDetails = make([]*BlobberAllocation, 0)
		allocationRequest.Stats = &StorageAllocationStats{}
		allocationRequest.Pool = &tokenpool.ZcnLockingPool{}
		for _, blobberNode := range storageNodes.Nodes {
			if blobberNode.WriteRatio.GetRatio() > allocationRequest.WriteRatio.GetRatio() || blobberNode.ReadRatio.GetRatio() > allocationRequest.ReadRatio.GetRatio() {
				continue
			}
			blobberAllocation := NewBlobberAllocation()
			blobberAllocation.Stats = &StorageAllocationStats{}
			blobberAllocation.Size = (allocationRequest.Size + int64(size-1)) / int64(size)
			blobberAllocation.AllocationID = t.Hash
			blobberAllocation.BlobberID = blobberNode.ID

			allocationRequest.BlobberDetails = append(allocationRequest.BlobberDetails, blobberAllocation)
			allocatedBlobbers = append(allocatedBlobbers, blobberNode)
			if len(allocatedBlobbers) >= size {
				break
			}
		}
		if len(allocatedBlobbers) < size {
			return "", common.NewError("not_enough_blobbers", "Not enough blobbers to honor the allocation")
		}
		sort.SliceStable(allocatedBlobbers, func(i, j int) bool {
			return allocatedBlobbers[i].ID < allocatedBlobbers[j].ID
		})
		transfer, _, err := allocationRequest.Pool.DigPool(t.Hash, t)
		if err != nil {
			return "", err
		}
		err = balances.AddTransfer(transfer)
		if err != nil {
			return "", err
		}
		allocationRequest.Blobbers = allocatedBlobbers
		allocationRequest.ID = t.Hash
		allocationRequest.Owner = t.ClientID
		allocationRequest.OwnerPublicKey = clientPublicKey
		allocationRequest.AmountPaid = 0
		allocationRequest.StartDate = t.CreationDate

		buff, err := sc.addAllocation(allocationRequest, balances)
		if err != nil {
			return "", common.NewError("allocation_request_failed", "Failed to store the allocation request")
		}
		return buff, nil
	}
	return "", common.NewError("invalid_allocation_request", "Failed storage allocate")
}

func (sc *StorageSmartContract) reclaimAllocationStake(t *transaction.Transaction, input []byte, balances c_state.StateContextI) (string, error) {
	allocationRequest := NewStorageAllocation()
	err := allocationRequest.Decode(input)
	if err != nil {
		return "", common.NewError("reclaim_allocation_stake_failed", "Failed to decode input data into storage allocation")
	}
	allocationBytes, _ := balances.GetTrieNode(allocationRequest.GetKey(sc.ID))
	if allocationBytes == nil {
		return "", common.NewError("reclaim_allocation_stake_failed", "Failed to reclaim allocation because it does not exist")
	}
	err = allocationRequest.Decode(allocationBytes.Encode())
	if err != nil {
		return "", common.NewError("reclaim_allocation_stake_failed", "Failed to decode storage allocation bytes into storage allocation")
	}
	if t.CreationDate < allocationRequest.Expiration {
		return "", common.NewError("reclaim_allocation_stake_failed", "Storage allocation has not yet expired")
	}
	guaranteedOnePercent := state.Balance(float64(allocationRequest.Size*allocationRequest.WriteRatio.ZCN) / float64(allocationRequest.WriteRatio.Size) * MINPERCENT)
	if allocationRequest.AmountPaid < guaranteedOnePercent {
		unpaidPerBlobber := (guaranteedOnePercent - allocationRequest.AmountPaid) / state.Balance(len(allocationRequest.Blobbers))
		for _, blobber := range allocationRequest.Blobbers {
			err = balances.AddTransfer(state.NewTransfer(sc.ID, blobber.DelegateID, unpaidPerBlobber))
			if err != nil {
				return "", err
			}
		}
		err = balances.AddTransfer(state.NewTransfer(sc.ID, allocationRequest.Owner, allocationRequest.Pool.Balance-(unpaidPerBlobber*state.Balance(len(allocationRequest.Blobbers)))))
		if err != nil {
			return "", err
		}
	} else {
		err = balances.AddTransfer(state.NewTransfer(sc.ID, allocationRequest.Owner, allocationRequest.Pool.Balance))
		if err != nil {
			return "", err
		}
	}
	return sc.deleteAllocation(allocationRequest, balances)
}
