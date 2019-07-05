package sharder

import (
	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	. "0chain.net/core/logging"
	"context"
	"go.uber.org/zap"
	"time"
)

const (
	Sync     = "syncing"
	SyncDone = "synced"
)

type SyncStats struct {
	Status          string
	SyncBeginR      int64
	SyncUntilR      int64
	CurrSyncR       int64
	SyncBlocksCount int64
}


/*HealthCheckWorker - checks the health for each round*/
func (sc *Chain) HealthCheckWorker(ctx context.Context) {
	hr := sc.HealthyRoundNumber
	hRound, err := sc.ReadHealthyRound(ctx)
	if err == nil {
		if hRound.Number > hr {
			hr = hRound.Number
		}
	}
	Logger.Info("health-check: round",
			zap.Int64("start", hr),
			zap.Int64("config", sc.HealthyRoundNumber),
			zap.Int64("datastore", hRound.Number))

	// The sharder is expected to have rounds <= hr
	sc.BSyncStats.SyncBeginR = hr + 1
	for true {
		select {
		case <-ctx.Done():
			return
		default:
			sc.SharderStats.HealthyRoundNum = hr
			hr = hr + 1
			t := time.Now()
			sc.healthCheck(ctx, hr)
			duration := time.Since(t)
			hRound.Number = hr
			err = sc.WriteHealthyRound(ctx, hRound)
			if err != nil {
				Logger.Error("health-check: datastore write failure",
						zap.Int64("round", hr),
						zap.Error(err))
			}
			sc.updateSyncStats(hr, duration)
		}
	}
}

func (sc *Chain) updateSyncStats(rNum int64, duration time.Duration) {
	var diff int64
	if sc.BSyncStats.CurrSyncR > 0 {
		diff = sc.BSyncStats.SyncUntilR - sc.BSyncStats.CurrSyncR
	} else {
		diff = sc.BSyncStats.SyncUntilR - sc.BSyncStats.SyncBeginR
	}
	if diff <= 0 {
		sc.BSyncStats.Status = SyncDone
	} else {
		sc.BSyncStats.Status = Sync
		BlockSyncTimer.Update(duration)
	}

	if sc.BSyncStats.Status == Sync {
		sc.BSyncStats.CurrSyncR = rNum
		sc.BSyncStats.SyncBlocksCount++
	} else {
		sc.BSyncStats.CurrSyncR = 0
	}
}

func (sc *Chain) healthCheck(ctx context.Context, rNum int64) {
	var r *round.Round
	var bs *block.BlockSummary
	var b *block.Block
	var hasEntity bool

	self := node.GetSelfNode(ctx)

	r, hasEntity = sc.hasRoundSummary(ctx, rNum)
	if !hasEntity {
		r = sc.syncRoundSummary(ctx, rNum, sc.BatchSyncSize)
	}
	bs, hasEntity = sc.hasBlockSummary(ctx, r.BlockHash)
	if !hasEntity {
		bs = sc.syncBlockSummary(ctx, r, sc.BatchSyncSize)
	}
	canShard := sc.IsBlockSharderFromHash(bs.Hash, self.Node)
	if canShard {
		b, hasEntity = sc.hasBlock(bs.Hash, r.Number)
		if !hasEntity {
			b = sc.syncBlock(ctx, r, canShard)
		}
	}
	hasTxns := sc.hasTransactions(ctx, bs)
	if !hasTxns {
		if b == nil {
			b = sc.syncBlock(ctx, r, canShard)
		}
		sc.storeBlockTransactions(ctx, b)
	}
}
