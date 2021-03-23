package storagesc

import (
	"encoding/json"
	"time"

	"0chain.net/core/common"
)

type newAllocationRequest struct {
	DataShards                 int              `json:"data_shards"`
	ParityShards               int              `json:"parity_shards"`
	Size                       int64            `json:"size"`
	Expiration                 common.Timestamp `json:"expiration_date"`
	Owner                      string           `json:"owner_id"`
	OwnerPublicKey             string           `json:"owner_public_key"`
	PreferredBlobbers          []string         `json:"preferred_blobbers"`
	ReadPriceRange             PriceRange       `json:"read_price_range"`
	WritePriceRange            PriceRange       `json:"write_price_range"`
	MaxChallengeCompletionTime time.Duration    `json:"max_challenge_completion_time"`
}

// storageAllocation from the request
func (nar *newAllocationRequest) storageAllocation() (sa *StorageAllocation) {
	sa = new(StorageAllocation)
	sa.DataShards = nar.DataShards
	sa.ParityShards = nar.ParityShards
	sa.Size = nar.Size
	sa.Expiration = nar.Expiration
	sa.Owner = nar.Owner
	sa.OwnerPublicKey = nar.OwnerPublicKey
	sa.PreferredBlobbers = nar.PreferredBlobbers
	sa.ReadPriceRange = nar.ReadPriceRange
	sa.WritePriceRange = nar.WritePriceRange
	sa.MaxChallengeCompletionTime = nar.MaxChallengeCompletionTime
	return
}

func (nar *newAllocationRequest) decode(b []byte) error {
	return json.Unmarshal(b, nar)
}
