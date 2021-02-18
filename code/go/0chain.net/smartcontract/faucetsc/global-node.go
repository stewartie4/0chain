package faucetsc

import (
	"encoding/json"
	"time"

	"0chain.net/chaincore/state"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
)

type GlobalNode struct {
	ID              string        `json:"id"`
	PourAmount      state.Balance `json:"pour_amount"`
	MaxPourAmount   state.Balance `json:"max_pour_amount"`
	PeriodicLimit   state.Balance `json:"periodic_limit"`
	GlobalLimit     state.Balance `json:"global_limit"`
	IndividualReset time.Duration `json:"individual_reset"` //in hours
	GlobalReset     time.Duration `json:"global_rest"`      //in hours
	Used            state.Balance `json:"used"`
	StartTime       time.Time     `json:"start_time"`
}

func (gn *GlobalNode) GetKey() datastore.Key {
	return datastore.Key(gn.ID + gn.ID)
}

func (gn *GlobalNode) GetHash() string {
	return util.ToHex(gn.GetHashBytes())
}

func (gn *GlobalNode) GetHashBytes() []byte {
	return encryption.RawHash(gn.Encode())
}

func (gn *GlobalNode) Encode() []byte {
	buff, _ := json.Marshal(gn)
	return buff
}

func (gn *GlobalNode) Decode(input []byte) error {
	err := json.Unmarshal(input, gn)
	return err
}

