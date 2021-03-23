package storagesc

import (
	"encoding/json"
	"errors"
	"fmt"

	"0chain.net/core/common"
)

// update allocation request
type updateAllocationRequest struct {
	ID         string           `json:"id"`              // allocation id
	OwnerID    string           `json:"owner_id"`        // Owner of the allocation
	Size       int64            `json:"size"`            // difference
	Expiration common.Timestamp `json:"expiration_date"` // difference
}

func (uar *updateAllocationRequest) decode(b []byte) error {
	return json.Unmarshal(b, uar)
}

// validate request
func (uar *updateAllocationRequest) validate(conf *scConfig,
	alloc *StorageAllocation) (err error) {
	if uar.Size == 0 && uar.Expiration == 0 {
		return errors.New("update allocation changes nothing")
	}
	if ns := alloc.Size + uar.Size; ns < conf.MinAllocSize {
		return fmt.Errorf("new allocation size is too small: %d < %d",
			ns, conf.MinAllocSize)
	}
	if len(alloc.BlobberDetails) == 0 {
		return errors.New("invalid allocation for updating: no blobbers")
	}
	return
}

// calculate size difference for every blobber of the allocations
func (uar *updateAllocationRequest) getBlobbersSizeDiff(
	alloc *StorageAllocation) (diff int64) {

	var size = alloc.DataShards + alloc.ParityShards
	if uar.Size > 0 {
		diff = (uar.Size + int64(size-1)) / int64(size)
	} else if uar.Size < 0 {
		diff = (uar.Size - int64(size-1)) / int64(size)
	}
	// else -> (0), no changes, avoid unnecessary calculation

	return
}

// new size of blobbers' allocation
func (uar *updateAllocationRequest) getNewBlobbersSize(
	alloc *StorageAllocation) (newSize int64) {
	return alloc.BlobberDetails[0].Size + uar.getBlobbersSizeDiff(alloc)
}
