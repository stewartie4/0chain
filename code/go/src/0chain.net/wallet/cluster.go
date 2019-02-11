package wallet

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"0chain.net/block"
	"0chain.net/client"
	"0chain.net/common"
	. "0chain.net/logging"
	"0chain.net/state"
	"0chain.net/transaction"
	"go.uber.org/zap"
)

var defaultClientID = "255763f9d43ccfaaa2b6525b5e6a85a759f9c3ed6e8998722004f7fac1f64dde"

type Nodes []string

type Cluster struct {
	Name                  string        `json:"cluster_name"`
	TransactionTimeout    time.Duration `json:"transaction_timeout"`
	Miners                Nodes         `json:"miner_access_points"`
	Sharders              Nodes         `json:"sharder_access_points"`
	Blobbers              Nodes         `json:"blobber_access_points"`
	ClientSignatureScheme string
	TBWorkers             int
	Wallets               int
}

type Block struct {
	block.Block `json:"block"`
}

func (c *Cluster) ReadNodes(file string) error {
	jsonFile, err := os.Open(file)
	if err != nil {
		return err
	}
	defer jsonFile.Close()
	bytes, _ := ioutil.ReadAll(jsonFile)
	err = json.Unmarshal(bytes, c)
	return err
}

func (c *Cluster) RegisterWithMiners(w *Wallet) {
	err := c.sendClient(w)
	for err != nil {
		time.Sleep(time.Millisecond * 250)
		err = c.sendClient(w)
	}
	err = common.NewError("", "")
	for err != nil {
		Logger.Info("Get client", zap.Any("error", err))
		err = c.GetClient(w)
	}
}

func (c *Cluster) sendClient(w *Wallet) error {
	cli, _ := client.Provider().(*client.Client)
	cli.ID = w.ClientID
	cli.PublicKey = w.SignatureScheme.GetPublicKey()
	nodes := c.SendToMiners(RegisterClient(cli))
	if len(nodes) == 0 {
		return common.NewError("send client", "there appears to be no miners up")
	}
	return nil
}

func (c *Cluster) SendRandomTransaction(w *Wallet) (*transaction.Transaction, error) {
	txn := w.CreateRandomSendTransaction(defaultClientID)
	nodes := c.SendToMiners(SubmitTransaction(txn))
	if len(nodes) == 0 {
		return txn, common.NewError("send transaction", "there appears to be no miners up")
	}
	return txn, nil
}

func (c *Cluster) SendTransaction(txn *transaction.Transaction) (int, error) {
	nodes := len(c.SendToMiners(SubmitTransaction(txn)))
	if nodes == 0 {
		return 0, common.NewError("send transaction", "there appears to be no miners up")
	}
	return nodes, nil
}

func (c *Cluster) GetTransaction(hash string) (*transaction.Confirmation, error) {
	m := make(map[string]string)
	m["hash"] = hash
	test := transaction.Confirmation{}
	nodes := c.SendToSharders(ConfirmTransaction(m, &test))
	if len(nodes) == 0 {
		return nil, common.NewError("failed transaction", fmt.Sprintf("hash: %v", hash))
	}
	return &test, nil
}

func (c *Cluster) GetBlockByHash(hash string) (*Block, error) {
	m := make(map[string]string)
	m["block"] = hash
	return c.getBlock(m)
}

func (c *Cluster) GetBlockByRound(round string) (*Block, error) {
	m := make(map[string]string)
	m["round"] = round
	return c.getBlock(m)
}

func (c *Cluster) getBlock(block map[string]string) (*Block, error) {
	block["content"] = "full"
	b := Block{}
	retry := 20
	found := false
	for retry > 0 && !found {
		nodes := c.SendToSharders(GetBlock(block, &b))
		if len(nodes) == 0 {
			retry--
			time.Sleep(100 * time.Millisecond)
		} else {
			found = true
		}
	}
	if !found {
		s := ""
		for _, key := range block {
			s = s + ", key: " + key + ", value: " + block[key]
		}
		return nil, common.NewError("aint no block", fmt.Sprintf("sharder doesn't have a block for %v", s))
	}
	return &b, nil
}

func (c *Cluster) GetBalance(id string) (state.State, error) {
	m := make(map[string]string)
	m["client_id"] = id
	state := state.State{}
	nodes := c.SendToSharders(GetBalance(m, &state))
	if len(nodes) == 0 {
		return state, common.NewError("get balance", "no sharders are reachable")
	}
	return state, nil
}

func (c *Cluster) GetClient(w *Wallet) error {
	m := make(map[string]string)
	m["id"] = w.ClientID
	nodes := c.SendToMiners(CheckClientStatus(m, nil))
	if len(nodes) == 0 {
		return common.NewError("registered client", "no miners are reachacble")
	}
	return nil
}
