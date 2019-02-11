package wallet

import (
	"context"
	"math/rand"
	"time"

	. "0chain.net/logging"
	"0chain.net/smartcontract"
	"0chain.net/state"
	"0chain.net/transaction"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var (
	successChannel  chan bool
	pourPoint       int64
	pourAmount      int64
	balanceWaitTime time.Duration
	wallets         []*Wallet
)

func SetupWorkers(ctx context.Context, c *Cluster, owner *Wallet) {
	c.initSetup()
	c.periodicallyCheckBalances()
	c.transactionBlaster()
}

func (c *Cluster) initSetup() {
	successChannel = make(chan bool, viper.GetInt("cluster.workers.transaction_blaster.success_buffer_size"))
	pourAmount = viper.GetInt64("wallet.pour_amount")
	pourPoint = viper.GetInt64("wallet.pour_point")
	balanceWaitTime = time.Duration(time.Minute * time.Duration(viper.GetInt("cluster.workers.periodic_balance_checker")))
	c.CreateWallets()
	c.RegisterWallets()
	Logger.Info("finished registration", zap.Any("wallets", len(wallets)))
	c.checkForPours()
	Logger.Info("finished checking for pours", zap.Any("wallets", len(wallets)))
}

func (c *Cluster) CreateWallets() {
	finished := false
	count := 0
	for !finished {
		w := &Wallet{}
		err := w.Initialize(c.ClientSignatureScheme)
		Logger.Info("wallet created", zap.Any("wallet_id", w.ClientID))
		if err == nil {
			wallets = append(wallets, w)
			count++
			finished = count == c.Wallets
		}
	}
}

func (c *Cluster) RegisterWallets() {
	for i, wallet := range wallets {
		Logger.Info("registering wallet", zap.Any("index", i), zap.Any("out of", len(wallets)))
		c.RegisterWithMiners(wallet)
	}
}

func (c *Cluster) transactionBlaster() {
	for i := 0; i < c.TBWorkers; i++ {
		go c.transactionSender()
	}
	c.transactionCounter()
}

func (c *Cluster) refillFaucet(w *Wallet) {
	worked := false
	for !worked {
		txn := w.CreateSendSCTransaction(smartcontract.FAUCET_CONTRACT_ADDRESS, 10000000000000000, `{"name":"refill","input":{}}`)
		_, err := c.SendTransaction(txn)
		if err == nil {
			Logger.Info("refilled faucet", zap.Any("txn", txn))
			worked = true
		} else {
			Logger.Info("error", zap.Any("error", err))
			time.Sleep(time.Second * 1)
		}
	}
}

func (c *Cluster) transactionSender() {
	rs := rand.NewSource(time.Now().UnixNano())
	prng := rand.New(rs)
	var txn *transaction.Transaction
	for true {
		r := prng.Int63n(100)
		if r < 25 {
			txn = c.createSendTransaction(prng)
		} else {
			txn = c.createDataTransaction(prng)
		}
		_, err := c.SendTransaction(txn)
		successChannel <- err == nil
	}
}

func (c *Cluster) transactionCounter() {
	successCount := 0
	failureCount := 0
	for true {
		select {
		case success := <-successChannel:
			if success {
				successCount++
			} else {
				failureCount++
			}
			Logger.Info("transaction counter", zap.Any("success", successCount), zap.Any("failure", failureCount))
		}
	}
}

func (c *Cluster) createDataTransaction(prng *rand.Rand) *transaction.Transaction {
	csize := len(wallets)
	wf := wallets[prng.Intn(csize)]
	txn := wf.CreateRandomDataTransaction()
	return txn
}

func (c *Cluster) createSendTransaction(prng *rand.Rand) *transaction.Transaction {
	var wf, wt *Wallet
	csize := len(wallets)
	for true {
		wf = wallets[prng.Intn(csize)]
		wt = wallets[prng.Intn(csize)]
		if wf != wt {
			break
		}
	}
	txn := wf.CreateRandomSendTransaction(wt.ClientID)
	return txn
}

func (c *Cluster) periodicallyCheckBalances() {
	go func() {
		for {
			time.Sleep(balanceWaitTime)
			c.checkForPours()
		}
	}()
}

func (c *Cluster) checkForPours() {
	for _, wallet := range wallets {
		poured := false
		pourNeeded := true
		for !poured && pourNeeded {
			ws, err := c.GetBalance(wallet.ClientID)
			Logger.Info("wallet state", zap.Any("wallet_id", wallet.ClientID), zap.Any("wallet_state", ws), zap.Any("poured", poured))
			if err != nil || ws.Balance < state.Balance(pourPoint) {
				txn := wallet.CreateFaucetPourTransaction(pourAmount)
				_, err := c.SendTransaction(txn)
				if err == nil {
					poured = true
				}
			} else {
				pourNeeded = false
			}
		}
	}
}
