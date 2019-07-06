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
	SyncProgress     = "syncing"
	SyncDone = "synced"
)

type SyncStats struct {
	Status     string

	// Interval bounds to start, current and final.
	HealthyRoundStart      int64
	Final      int64
	Current    int64

	ProcessedBlocks int64
	Invocations int64
}


/*HealthCheckWorker - checks the health for each round*/
func (sc *Chain) HealthCheckWorker(ctx context.Context) {
	// Read the healthy round number from the configuration file.
	// It will be set to zero for default case. This would be
	// the genesis block.
	hr := sc.HealthyRoundNumber

	// Read the round number stored in the database.
	hRound, err := sc.ReadHealthyRound(ctx)
	if err == nil {
		if hRound.Number > hr {
			hr = hRound.Number
		}
	}

	// Log the initial startup conditions.
	Logger.Info("health-check: init-round",
			zap.Int64("start", hr),
			zap.Int64("config", sc.HealthyRoundNumber),
			zap.Int64("datastore", hRound.Number))

	Logger.Info("health-check: init-batch",
			zap.Int("size", sc.BatchSyncSize))

	// Initialize the health check statistics
	sc.SharderStats.HealthyRoundNum = hr
	sc.initSyncStats(ctx, hr)

	for true {
		select {
		case <-ctx.Done():
			return
		default:
			sc.SharderStats.HealthyRoundNum = hr
			currentRound := sc.BSyncStats.Current + 1
			t := time.Now()
			sc.healthCheck(ctx, currentRound)
			duration := time.Since(t)
			hRound.Number = currentRound
			err = sc.WriteHealthyRound(ctx, hRound)
			if err != nil {
				Logger.Error("health-check: datastore write failure",
						zap.Int64("round", hr),
						zap.Error(err))
			}
			sc.updateSyncStats(ctx, hr, duration)
			sc.waitForWork(ctx)
		}
	}
}

func (sc *Chain) initSyncStats(ctx context.Context, healthyRound int64) {

	// The sharder is expected to have rounds <= healthyRound
	sc.BSyncStats.HealthyRoundStart = healthyRound
	sc.BSyncStats.Current = healthyRound
	sc.BSyncStats.Invocations = 0
	sc.BSyncStats.ProcessedBlocks = 0

	// Update the sync until round.
	roundEntity, err := sc.GetMostRecentRoundFromDB(ctx)
	if err != nil {
		// Update the sync until to the last finalized block
		sc.BSyncStats.Final = roundEntity.Number
	}

}

func (sc *Chain) updateSyncStats(ctx context.Context, current int64, duration time.Duration) {
	sc.BSyncStats.Current = current
	BlockSyncTimer.Update(duration)

	// Update the number of invocations
	sc.BSyncStats.Invocations++

	// Update number of processed blocks
	sc.BSyncStats.ProcessedBlocks++

	// Update the sync until round.
	roundEntity, err := sc.GetMostRecentRoundFromDB(ctx)
	if err != nil {
		// Update the sync until to the last finalized block
		sc.BSyncStats.Final = roundEntity.Number
	}

}

func(sc *Chain) waitForWork(ctx context.Context) {
	if sc.BSyncStats.Current >= sc.BSyncStats.Final {
		// Exceeded the current time. Sleep
		time.Sleep(time.Minute)
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
		// No round found. Fetch the round summary and round information.
		r = sc.syncRoundSummary(ctx, rNum, sc.BatchSyncSize)
	}

	if sc.isValidRound(r) == false {
		// Unable to get the round information.
		return
	}

	// Obtained valid round. Retrieve blocks.
	bs, hasEntity = sc.hasBlockSummary(ctx, r.BlockHash)
	if !hasEntity {
		// Missing block summary. Sync the blocks
		bs = sc.syncBlockSummary(ctx, r, sc.BatchSyncSize)
	}

	if bs == nil {
		// Unable to retrieve block summary.
		return
	}

	// Check for block presence.
	n := sc.GetActivesetSharder(self.GNode)
	canShard := sc.IsBlockSharderFromHash(bs.Hash, n)
	if canShard {
		b, hasEntity = sc.hasBlock(bs.Hash, r.Number)
		if !hasEntity {
			b = sc.syncBlock(ctx, r, canShard)
		}
	}

	hasTxns := sc.hasTransactions(ctx, bs)
	if hasTxns && b != nil {
		// The block has transactions and needs to be stored.
		sc.storeBlockTransactions(ctx, b)
	}
}
