package smartcontract

import (
	"net/http"
	"net/url"

	c_state "0chain.net/chaincore/chain/state"
)

// SmartContractExecuteRest structure is responsible for all information to execute smart contract from rest api
type SmartContractExecuteRest struct {
	Address, Path string
	Params        url.Values
	Balances      c_state.StateContextI
}

// SmartContractExecuteStats structure is responsible for all information to execute smart contract from stats
type SmartContractExecuteStats struct {
	Address  string
	Params   url.Values
	Response http.ResponseWriter
}
