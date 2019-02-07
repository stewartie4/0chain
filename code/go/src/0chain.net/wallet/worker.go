package wallet

import (
	"context"
	"time"

	. "0chain.net/logging"
	"go.uber.org/zap"
)

var successChannel chan bool

func init() {
	successChannel = make(chan bool, 256)
}

func SetupWorkers(ctx context.Context, c *Cluster, w *Wallet) {
	c.transactionBlaster(w)
}

func (c *Cluster) transactionBlaster(w *Wallet) {
	c.RegisterWithMiners(w)
	for i := 0; i < c.TBWorkers; i++ {
		go c.transactionSender(w)
	}
	c.transactionCounter(w)
}

func (c *Cluster) transactionSender(w *Wallet) {
	for true {
		txn := w.CreateDataTransaction("this is a message at: " + time.Now().String())
		_, err := c.SendTransaction(txn)
		successChannel <- err == nil
	}
}

func (c *Cluster) transactionCounter(w *Wallet) {
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
