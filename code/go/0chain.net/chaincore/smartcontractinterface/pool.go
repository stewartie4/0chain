package smartcontractinterface

import (
	"encoding/json"

	"0chain.net/chaincore/tokenpool"
)

type PoolStatsI interface {
	GetStats() interface{}
}

type DelegatePool struct {
	PoolStatsI                `json:"stats"`
	*tokenpool.ZcnLockingPool `json:"pool"`
}

func NewDelegatePool(ps PoolStatsI) *DelegatePool {
	return &DelegatePool{ZcnLockingPool: &tokenpool.ZcnLockingPool{}, PoolStatsI: ps}
}

func (dp *DelegatePool) Encode() []byte {
	buff, _ := json.Marshal(dp)
	return buff
}

func (dp *DelegatePool) Decode(input []byte) error {
	return json.Unmarshal(input, dp)
}
