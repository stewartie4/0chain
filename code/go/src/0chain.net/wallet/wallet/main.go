package main

import (
	"fmt"

	"0chain.net/common"
	"0chain.net/config"
	"0chain.net/logging"
	"0chain.net/wallet"
	"github.com/spf13/viper"
)

var w *wallet.Wallet

func init() {
	logging.InitLogging("development")
	wallet.SetupWallet()
	wallet.SetupW2MSenders()
	wallet.SetupW2SSenders()
	config.SetupWalletConfig()
}

func main() {
	c := &wallet.Cluster{}
	c.TBWorkers = viper.GetInt("cluster.workers.transaction_blaster.workers")
	c.Wallets = viper.GetInt("cluster.wallets")
	c.ClientSignatureScheme = viper.GetString("cluster.signature_scheme")
	err := c.ReadNodes(viper.GetString("cluster.nodes_file"))
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
	w = &wallet.Wallet{}
	w.GetOwnerWallet(c)
	ctx := common.GetRootContext()
	wallet.SetupWorkers(ctx, c, w)
}
