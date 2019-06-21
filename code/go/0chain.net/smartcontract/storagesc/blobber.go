package storagesc

import (
	"encoding/json"
	"fmt"
	"sort"

	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
)

func (sc *StorageSmartContract) getBlobbersList(balances c_state.StateContextI) (*StorageNodes, error) {
	allBlobbersList := &StorageNodes{}
	allBlobbersBytes, err := balances.GetTrieNode(ALL_BLOBBERS_KEY)
	if allBlobbersBytes == nil {
		return allBlobbersList, nil
	}
	err = json.Unmarshal(allBlobbersBytes.Encode(), allBlobbersList)
	if err != nil {
		return nil, common.NewError("getBlobbersList_failed", "Failed to retrieve existing blobbers list")
	}
	sort.SliceStable(allBlobbersList.Nodes, func(i, j int) bool {
		return allBlobbersList.Nodes[i].ID < allBlobbersList.Nodes[j].ID
	})
	return allBlobbersList, nil
}

func (sc *StorageSmartContract) drainStakeForBlobber(t *transaction.Transaction, input []byte, balances c_state.StateContextI) (string, error) {
	newBlobber := &StorageNode{}
	stakeRequest := &StakeRequest{}
	err := stakeRequest.Decode(input) //json.Unmarshal(input, &newBlobber)
	if err != nil {
		return "", err
	}
	newBlobber.ID = stakeRequest.ID
	blobberBytes, _ := balances.GetTrieNode(newBlobber.GetKey(sc.ID))
	if blobberBytes == nil {
		return "", common.NewError("drain_blobber_stake_failed", "The blobber doesn't exist")
	}
	err = newBlobber.Decode(blobberBytes.Encode())
	if err != nil {
		return "", err
	}
	if newBlobber.DelegateID != t.ClientID {
		return "", common.NewError("drain_blobber_stake_failed", "only delegator can delegate for blobber")
	}
	err = newBlobber.SetStakedCapacity(-t.Value)
	if err != nil {
		return "", err
	}
	allBlobbersList, err := sc.getBlobbersList(balances)
	if err != nil {
		return "", err
	}
	allBlobbersList.UpdateStorageNode(newBlobber)
	balances.InsertTrieNode(ALL_BLOBBERS_KEY, allBlobbersList)
	transfer, _, err := newBlobber.StakePool.DrainPool(sc.ID, newBlobber.DelegateID, state.Balance(t.Value), nil)
	if err != nil {
		return "", err
	}
	balances.AddTransfer(transfer)
	return string(transfer.Encode()), nil
}

func (sc *StorageSmartContract) stakeForBlobber(t *transaction.Transaction, input []byte, balances c_state.StateContextI) (string, error) {
	newBlobber := &StorageNode{}
	stakeRequest := &StakeRequest{}
	err := stakeRequest.Decode(input) //json.Unmarshal(input, &newBlobber)
	if err != nil {
		return "", err
	}
	newBlobber.ID = stakeRequest.ID
	var transfer *state.Transfer
	blobberBytes, _ := balances.GetTrieNode(newBlobber.GetKey(sc.ID))
	if blobberBytes == nil {
		return "", common.NewError("stake_for_blobber_failed", "The blobber doesn't exist")
	}
	err = newBlobber.Decode(blobberBytes.Encode())
	if err != nil {
		return "", err
	}
	if newBlobber.DelegateID != t.ClientID {
		return "", common.NewError("stake_for_blobber_failed", "only delegator can delegate for blobber")
	}
	err = newBlobber.SetStakedCapacity(t.Value)
	if err != nil {
		return "", err
	}
	if newBlobber.StakePool.ID == "" {
		transfer, _, err = newBlobber.StakePool.DigPool(t.ClientID, t)
	} else {
		transfer, _, err = newBlobber.StakePool.FillPool(t)
	}
	if err != nil {
		return "", err
	}
	err = balances.AddTransfer(transfer)
	if err != nil {
		return "", err
	}
	allBlobbersList, err := sc.getBlobbersList(balances)
	if err != nil {
		return "", err
	}
	allBlobbersList.UpdateStorageNode(newBlobber)
	balances.InsertTrieNode(ALL_BLOBBERS_KEY, allBlobbersList)
	balances.InsertTrieNode(newBlobber.GetKey(sc.ID), newBlobber)
	return string(newBlobber.Encode()) + string(transfer.Encode()), nil
}

func (sc *StorageSmartContract) addBlobber(t *transaction.Transaction, input []byte, balances c_state.StateContextI) (string, error) {
	newBlobber := &StorageNode{}
	newBlobber.ID = t.ClientID
	blobberSS, _ := balances.GetTrieNode(newBlobber.GetKey(sc.ID))
	if blobberSS != nil {
		return "", common.NewError("add_blobber_failed", "Blobber already exists")
	}
	err := newBlobber.Decode(input)
	if err != nil {
		return "", err
	}
	newBlobber.ID = t.ClientID
	newBlobber.PublicKey = t.PublicKey
	if !newBlobber.Validate() {
		return "", common.NewError("add_blobber_failed", fmt.Sprintf("blobber's storage node is not valid %v", string(newBlobber.Encode())))
	}
	newBlobber.StakeMultiplyer = STAKEMULTIPLYER
	newBlobber.StakedCapacity = 0
	newBlobber.TotalStaked = 0
	newBlobber.StakePool = &tokenpool.ZcnLockingPool{}
	newBlobber.Allocations = &Allocations{}
	newBlobber.LongestCommitment = common.Timestamp(0)
	allBlobbersList, err := sc.getBlobbersList(balances)
	if err != nil {
		return "", err
	}
	allBlobbersList.Nodes = append(allBlobbersList.Nodes, newBlobber)
	balances.InsertTrieNode(newBlobber.GetKey(sc.ID), newBlobber)
	balances.InsertTrieNode(ALL_BLOBBERS_KEY, allBlobbersList)
	return string(newBlobber.Encode()), nil
}

func (sc *StorageSmartContract) removeBlobber(t *transaction.Transaction, input []byte, balances c_state.StateContextI) (string, error) {
	newBlobber := &StorageNode{}
	newBlobber.ID = t.ClientID
	blobberSS, _ := balances.GetTrieNode(newBlobber.GetKey(sc.ID))
	if blobberSS == nil {
		return "", common.NewError("remove_blobber_failed", "Blobber doesn't exists")
	}
	err := newBlobber.Decode(input)
	if err != nil {
		return "", err
	}
	if len(newBlobber.Allocations.List) != 0 {
		return "", common.NewError("failed_to_remove_blobber", fmt.Sprintf("Blobber is still beholden to allocations: %v", newBlobber.Allocations.List))
	}
	allBlobbersList, err := sc.getBlobbersList(balances)
	if err != nil {
		return "", err
	}
	transfer, _, err := newBlobber.StakePool.EmptyPool(sc.ID, newBlobber.DelegateID, nil)
	if err != nil {
		return "", err
	}
	balances.AddTransfer(transfer)
	allBlobbersList.DeleteStorageNode(newBlobber.ID)
	balances.InsertTrieNode(ALL_BLOBBERS_KEY, allBlobbersList)
	balances.DeleteTrieNode(newBlobber.GetKey(sc.ID))
	return "", nil
}

func (sc *StorageSmartContract) commitBlobberRead(t *transaction.Transaction, input []byte, balances c_state.StateContextI) (string, error) {
	commitRead := &ReadConnection{}
	err := commitRead.Decode(input)
	if err != nil {
		return "", err
	}

	lastBlobberClientReadBytes, err := balances.GetTrieNode(commitRead.GetKey(sc.ID))
	lastCommittedRM := &ReadConnection{}
	if lastBlobberClientReadBytes != nil {
		lastCommittedRM.Decode(lastBlobberClientReadBytes.Encode())
	}

	err = commitRead.ReadMarker.Verify(lastCommittedRM.ReadMarker)
	if err != nil {
		return "", common.NewError("invalid_read_marker", "Invalid read marker."+err.Error())
	}
	sa := NewStorageAllocation()
	sa.ID = commitRead.ReadMarker.AllocationID
	storageAllocationBytes, _ := balances.GetTrieNode(sa.GetKey(sc.ID))
	if storageAllocationBytes == nil {
		return "", common.NewError("invalid_read_marker", "Storage allocation for writer marker doesn't exist")
	}
	err = sa.Decode(storageAllocationBytes.Encode())
	if err != nil {
		return "", common.NewError("invalid_read_marker", "Failed to decode storage allocation bytes into storage allocation struct")
	}
	sn := &StorageNode{ID: commitRead.ReadMarker.BlobberID}
	storageNodeBytes, _ := balances.GetTrieNode(sn.GetKey(sc.ID))
	if storageNodeBytes == nil {
		return "", common.NewError("invalid_read_marker", "Storage node for write marker doesn't exist")
	}
	err = sn.Decode(storageNodeBytes.Encode())
	if err != nil {
		return "", common.NewError("invalid_read_marker", "Failed to decode storage node bytes into storage node struct")
	}
	amount := state.Balance(commitRead.ReadMarker.ReadCounter-lastCommittedRM.ReadMarker.ReadCounter) * BLOCK / state.Balance(sn.ReadRatio.Size) * state.Balance(sn.ReadRatio.ZCN)
	transfer, _, err := sa.Pool.DrainPool(sc.ID, sn.DelegateID, amount, nil)
	balances.AddTransfer(transfer)
	balances.InsertTrieNode(commitRead.GetKey(sc.ID), commitRead)
	return "success", nil
}

func (sc *StorageSmartContract) commitBlobberConnection(t *transaction.Transaction, input []byte, balances c_state.StateContextI) (string, error) {
	var commitConnection BlobberCloseConnection
	err := json.Unmarshal(input, &commitConnection)
	if err != nil {
		return "", err
	}

	if !commitConnection.Verify() {
		return "", common.NewError("invalid_parameters", "Invalid input")
	}

	if commitConnection.WriteMarker.BlobberID != t.ClientID {
		return "", common.NewError("invalid_parameters", "Invalid Blobber ID for closing connection. Write marker not for this blobber")
	}

	allocationObj := NewStorageAllocation()
	allocationObj.ID = commitConnection.WriteMarker.AllocationID
	allocationBytes, err := balances.GetTrieNode(allocationObj.GetKey(sc.ID))

	if allocationBytes == nil || err != nil {
		return "", common.NewError("invalid_parameters", "Invalid allocation ID")
	}

	err = allocationObj.Decode(allocationBytes.Encode())
	if allocationBytes == nil || err != nil {
		return "", common.NewError("invalid_parameters", "Invalid allocation ID. Failed to decode from DB")
	}

	if allocationObj.Expiration < t.CreationDate {
		return "", common.NewError("invalid_commit", "Invalid commit. The allocation has already exipred")
	}
	if allocationObj.Owner != commitConnection.WriteMarker.ClientID {
		return "", common.NewError("invalid_parameters", "Write marker has to be by the same client as owner of the allocation")
	}

	blobberAllocation, ok := allocationObj.BlobberMap[t.ClientID]
	if !ok {
		return "", common.NewError("invalid_parameters", "Blobber is not part of the allocation")
	}

	blobberAllocationBytes, err := json.Marshal(blobberAllocation)

	if !commitConnection.WriteMarker.VerifySignature(allocationObj.OwnerPublicKey) {
		return "", common.NewError("invalid_parameters", "Invalid signature for write marker")
	}

	if blobberAllocation.AllocationRoot == commitConnection.AllocationRoot && blobberAllocation.LastWriteMarker != nil && blobberAllocation.LastWriteMarker.PreviousAllocationRoot == commitConnection.PrevAllocationRoot {
		return string(blobberAllocationBytes), nil
	}

	if blobberAllocation.AllocationRoot != commitConnection.PrevAllocationRoot {
		return "", common.NewError("invalid_parameters", "Previous allocation root does not match the latest allocation root")
	}

	if blobberAllocation.Stats.UsedSize+commitConnection.WriteMarker.Size > blobberAllocation.Size {
		return "", common.NewError("invalid_parameters", "Size for blobber allocation exceeded maximum")
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
	amount := state.Balance(float64(sn.WriteRatio.ZCN*commitConnection.WriteMarker.Size) / float64(sn.WriteRatio.Size))
	pool := &tokenpool.ZcnLockingPool{}
	pool.ID = t.Hash
	allocationObj.Pool.TransferTo(pool, amount, nil)
	blobberAllocation.WritePools[pool.ID] = pool
	blobberAllocation.AllocationRoot = commitConnection.AllocationRoot
	blobberAllocation.LastWriteMarker = commitConnection.WriteMarker
	blobberAllocation.Stats.UsedSize += commitConnection.WriteMarker.Size
	blobberAllocation.Stats.NumWrites++

	allocationObj.Stats.UsedSize += commitConnection.WriteMarker.Size
	allocationObj.Stats.NumWrites++
	balances.InsertTrieNode(allocationObj.GetKey(sc.ID), allocationObj)
	balances.InsertTrieNode(sn.GetKey(sc.ID), sn)

	blobberAllocationBytes, err = json.Marshal(blobberAllocation.LastWriteMarker)
	return string(blobberAllocationBytes), err
}
