package faucetsc

import (
	"encoding/json"
	"time"

	"0chain.net/chaincore/state"
)

type periodicResponse struct {
	Used    state.Balance `json:"tokens_poured"`
	Start   time.Time     `json:"start_time"`
	Restart string        `json:"time_left"`
	Allowed state.Balance `json:"tokens_allowed"`
}

func (pr *periodicResponse) encode() []byte {
	buff, _ := json.Marshal(pr)
	return buff
}

func (pr *periodicResponse) decode(input []byte) error {
	err := json.Unmarshal(input, pr)
	return err
}
