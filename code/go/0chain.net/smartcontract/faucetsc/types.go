package faucetsc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/core/encryption"
)

var (
	miners   []*miner
	sharders []*sharder
)

type Client struct {
	id      string                     // identifier
	pk      string                     // public key
	scheme  encryption.SignatureScheme // pk/sk
	balance state.Balance              // client wallet balance

	keep state.Balance // keep latest know balance (manual control)
}



type sharder struct {
	sharder  *Client
	delegate *Client
	stakers  []*Client
}

type miner struct {
	miner    *Client
	delegate *Client
	stakers  []*Client
}

func newClient(balance state.Balance, balances cstate.StateContextI) (
	client *Client) {

	var scheme = encryption.NewBLS0ChainScheme()
	scheme.GenerateKeys()

	client = new(Client)
	client.balance = balance
	client.scheme = scheme

	client.pk = scheme.GetPublicKey()
	client.id = encryption.Hash(client.pk)

	balances.(*testBalances).balances[client.id] = balance
	return
}
