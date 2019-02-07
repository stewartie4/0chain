package wallet

import (
	"context"
	"time"

	. "0chain.net/logging"
	"go.uber.org/zap"
)

func SetupWorkers(ctx context.Context, c *Cluster, w *Wallet) {
	c.RegisterWithMiners(w)
	for i := 0; i < c.TBWorkers-1; i++ {
		go c.transactionBlaster(w)
	}
	c.transactionBlaster(w)
}

func (c *Cluster) transactionBlaster(w *Wallet) {
	count := 0
	for true {
		txn := w.CreateDataTransaction("this is a message at: " + time.Now().String())
		_, err := c.SendTransaction(txn)
		if err == nil {
			count++
			Logger.Info("sent transaction", zap.Any("count", count))
		} else {
			Logger.Error("failed to send transaction", zap.Any("error", err))
		}
	}
}
