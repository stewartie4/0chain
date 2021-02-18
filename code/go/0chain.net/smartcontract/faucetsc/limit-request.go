package faucetsc

import (
	"encoding/json"
	"time"

	"0chain.net/chaincore/state"
)

type limitRequest struct {
	PourAmount      state.Balance `json:"pour_amount"`
	MaxPourAmount   state.Balance `json:"max_pour_amount"`
	PeriodicLimit   state.Balance `json:"periodic_limit"`
	GlobalLimit     state.Balance `json:"global_limit"`
	IndividualReset time.Duration `json:"individual_reset"` //in hours
	GlobalReset     time.Duration `json:"global_rest"`      //in hours
}

func (lr *limitRequest) encode() []byte {
	buff, _ := json.Marshal(lr)
	return buff
}

func (lr *limitRequest) decode(input []byte) error {
	err := json.Unmarshal(input, lr)
	return err
}
