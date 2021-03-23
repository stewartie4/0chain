package storagesc

import (
	"fmt"

	"0chain.net/chaincore/chain"
	"0chain.net/core/common"
	"0chain.net/core/encryption"
)

type ReadMarker struct {
	ClientID        string           `json:"client_id"`
	ClientPublicKey string           `json:"client_public_key"`
	BlobberID       string           `json:"blobber_id"`
	AllocationID    string           `json:"allocation_id"`
	OwnerID         string           `json:"owner_id"`
	Timestamp       common.Timestamp `json:"timestamp"`
	ReadCounter     int64            `json:"counter"`
	Signature       string           `json:"signature"`
	PayerID         string           `json:"payer_id"`
	AuthTicket      *AuthTicket      `json:"auth_ticket"`
}

func (rm *ReadMarker) VerifySignature(clientPublicKey string) bool {
	hashData := rm.GetHashData()
	signatureHash := encryption.Hash(hashData)
	signatureScheme := chain.GetServerChain().GetSignatureScheme()
	signatureScheme.SetPublicKey(clientPublicKey)
	sigOK, err := signatureScheme.Verify(rm.Signature, signatureHash)
	if err != nil {
		return false
	}
	if !sigOK {
		return false
	}
	return true
}

func (rm *ReadMarker) verifyAuthTicket(alloc *StorageAllocation, now common.Timestamp) (err error) {

	// owner downloads, pays itself, no ticket needed
	if rm.PayerID == alloc.Owner {
		return
	}
	// 3rd party payment
	if rm.AuthTicket == nil {
		return common.NewError("invalid_read_marker", "missing auth. ticket")
	}
	return rm.AuthTicket.verify(alloc, now, rm.PayerID)
}

func (rm *ReadMarker) GetHashData() string {
	hashData := fmt.Sprintf("%v:%v:%v:%v:%v:%v:%v", rm.AllocationID,
		rm.BlobberID, rm.ClientID, rm.ClientPublicKey, rm.OwnerID,
		rm.ReadCounter, rm.Timestamp)
	return hashData
}

func (rm *ReadMarker) Verify(prevRM *ReadMarker) error {

	if rm.ReadCounter <= 0 || len(rm.BlobberID) == 0 || len(rm.ClientID) == 0 ||
		rm.Timestamp == 0 {

		return common.NewError("invalid_read_marker",
			"length validations of fields failed")
	}

	if prevRM != nil {
		if rm.ClientID != prevRM.ClientID || rm.BlobberID != prevRM.BlobberID ||
			rm.Timestamp < prevRM.Timestamp ||
			rm.ReadCounter < prevRM.ReadCounter {

			return common.NewError("invalid_read_marker",
				"validations with previous marker failed.")
		}
	}

	if ok := rm.VerifySignature(rm.ClientPublicKey); ok {
		return nil
	}

	return common.NewError("invalid_read_marker",
		"Signature verification failed for the read marker")
}
