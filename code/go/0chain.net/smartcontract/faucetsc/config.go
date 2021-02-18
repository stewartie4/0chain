package faucetsc

import (
	"context"
	"net/url"
	"time"

	chainstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/state"
)

type faucetConfig struct {
	PourAmount      state.Balance `json:"pour_amount"`
	MaxPourAmount   state.Balance `json:"max_pour_amount"`
	PeriodicLimit   state.Balance `json:"periodic_limit"`
	GlobalLimit     state.Balance `json:"global_limit"`
	IndividualReset time.Duration `json:"individual_reset"` //in hours
	GlobalReset     time.Duration `json:"global_rest"`      //in hours
}

// configurations from sc.yaml
func getConfig() (*faucetConfig, error) {
	return &faucetConfig{
		PourAmount:      state.Balance(config.SmartContractConfig.GetInt("smart_contracts.faucetsc.pour_amount")),
		MaxPourAmount:   state.Balance(config.SmartContractConfig.GetInt("smart_contracts.faucetsc.max_pour_amount")),
		PeriodicLimit:   state.Balance(config.SmartContractConfig.GetInt("smart_contracts.faucetsc.periodic_limit")),
		GlobalLimit:     state.Balance(config.SmartContractConfig.GetInt("smart_contracts.faucetsc.global_limit")),
		IndividualReset: config.SmartContractConfig.GetDuration("smart_contracts.faucetsc.individual_reset"),
		GlobalReset:     config.SmartContractConfig.GetDuration("smart_contracts.faucetsc.global_reset"),
	}, nil
}

//
// REST-handler
//

func (fc *FaucetSmartContract) getConfigHandler(context.Context,
	url.Values, chainstate.StateContextI) (interface{}, error) {
	return getConfig()
}
