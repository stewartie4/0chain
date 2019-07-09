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

type BlockStatus int

const (
	BlockSuccess BlockStatus = 1 + iota
	InvalidRound
	MissingSummary
	MissingBlock
)

type HealthCheckStatus string

const (
	SyncProgress HealthCheckStatus = "syncing"
	SyncHiatus                     = "hiatus"
	SyncDone                       = "synced"
)

type BlockCounters struct {
	CycleStart time.Time
	CycleEnd time.Time
	CycleDuration time.Duration

	BlockSuccess int64
	InvalidRound int64
	MissingSummary int64
	MissingBlock int64
}

func (bc *BlockCounters) init() {
	bc.CycleStart = time.Now()
	bc.CycleEnd = time.Time{}
	bc.CycleDuration = 0

	bc.InvalidRound = 0
	bc.BlockSuccess = 0
	bc.MissingBlock = 0
	bc.MissingSummary = 0
}

type SyncStats struct {
	Status HealthCheckStatus

	// Interval bounds to start, current and final.
	LowRound     int64
	CurrentRound int64
	HighRound    int64

	current BlockCounters
	previous BlockCounters

	CycleCount    int64
	Invocations   int64

	WaitNewBlocks int64

	Inception  time.Time
	// CycleStart time.Time
}

/*HealthCheckWorker - checks the health for each round*/
func (sc *Chain) HealthCheckWorker(ctx context.Context) {
	// Read the healthy round number from the configuration file.
	// It will be set to zero for default case. This would be
	// the genesis block.
	hr := sc.HealthCheckStartRound

	bss := sc.BlockSyncStats

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
		zap.Int64("config", sc.HealthCheckStartRound),
		zap.Int64("datastore", hRound.Number),
		zap.Int("batch-size", sc.BatchSyncSize))

	// Initialize the health check statistics
	sc.initSyncStats(ctx, hr, true)

	for true {
		select {
		case <-ctx.Done():
			return
		default:
			bss.Status = SyncProgress
			currentRound := bss.CurrentRound
			if bss.CurrentRound < bss.HighRound {
				// Update the current round number.
				bss.CurrentRound++
				t := time.Now()
				blockStatus := sc.healthCheck(ctx, currentRound)
				duration := time.Since(t)
				hRound.Number = currentRound
				err = sc.WriteHealthyRound(ctx, hRound)
				if err != nil {
					Logger.Error("health-check: datastore write failure",
						zap.Int64("round", hr),
						zap.Error(err))
				}

				// Update the statistics
				sc.updateSyncStats(ctx, hr, duration, blockStatus)
			}

			// Wait for new work.
			sc.waitForWork(ctx)
		}
	}
}

func (sc *Chain) initSyncStats(ctx context.Context, roundStart int64, inception bool) {

	bss := sc.BlockSyncStats
	var highRound int64

	// Update the sync until round.
	roundEntity, err := sc.GetMostRecentRoundFromDB(ctx)
	if err != nil {
		// Update the sync until to the last finalized block
		highRound = roundEntity.Number
	}

	// The sharder is expected to have rounds <= healthyRound
	lowRound := roundStart

	if lowRound > highRound {
		lowRound = highRound
	}

	// Update the sc low, current and high limits.
	bss.LowRound = lowRound
	bss.CurrentRound = lowRound
	bss.HighRound = highRound

	if inception {
		// Initial setup
		bss.Invocations = 0
		bss.Inception = time.Now()
	}

	// Beginning of a new cycle
	bss.CycleCount++

	// Clear old counters.
	bss.WaitNewBlocks = 0

	// Copy the counters.
	bss.previous = bss.current

	// Clear current cycle counters
	bss.current.init()

	Logger.Info("health-check: cycle-init",
		zap.Int64("iteration", bss.CycleCount),
		zap.Int64("low", bss.LowRound),
		zap.Int64("current", bss.CurrentRound),
		zap.Int64("high", bss.HighRound))
}

func (sc *Chain) updateSyncStats(ctx context.Context, current int64, duration time.Duration, status BlockStatus) {
	var highRound int64
	bss := sc.BlockSyncStats

	// Update the number of invocations
	bss.Invocations++

	// Update the timer.
	BlockSyncTimer.Update(duration)

	switch status {
	case BlockSuccess:
		bss.current.BlockSuccess++
	case InvalidRound:
		bss.current.InvalidRound++
	case MissingSummary:
		bss.current.MissingSummary++
	case MissingBlock:
		bss.current.MissingBlock++
	}

	// Update the sync until round.
	roundEntity, err := sc.GetMostRecentRoundFromDB(ctx)
	if err != nil {
		// Update the sync until to the last finalized block
		highRound = roundEntity.Number
	} else {
		highRound = bss.HighRound
	}

	if current > highRound {
		current = highRound
	}

	// Update the limits
	bss.CurrentRound = current
	bss.HighRound = highRound
}

func (sc *Chain) waitForWork(ctx context.Context) {
	bss := sc.BlockSyncStats
	for true {
		// Check for new blocks.
		roundEntity, err := sc.GetMostRecentRoundFromDB(ctx)
		if err != nil {
			// Update the high round
			bss.HighRound = roundEntity.Number
		}

		if bss.CurrentRound >= bss.HighRound {
			// Reached the current goal. Sleep for new blocks.
			bss.Status = SyncHiatus
			bss.WaitNewBlocks++
			time.Sleep(time.Duration(sc.HealthCheckCycleHiatus) * time.Minute)

			// Check if it is time to repeat the cycle
			elapsedTime := time.Now().Sub(bss.current.CycleStart)
			if elapsedTime > time.Duration(sc.HealthCheckCycleRepeat)*time.Minute {
				// Time to repeat entire health-check cycle. Zero the round in the database.
				roundZero, err := sc.ReadHealthyRound(ctx)
				if err != nil {
					roundZero.Number = 0
					sc.WriteHealthyRound(ctx, roundZero)
				}
				// Log end of the current cycle
				bss.current.CycleEnd = time.Now()
				bss.current.CycleDuration = time.Since(bss.current.CycleStart)
				sc.initSyncStats(ctx, 0, false)
				break;
			}
		} else {
			break;
		}
	}
}

func (sc *Chain) healthCheck(ctx context.Context, rNum int64) BlockStatus {
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
		return InvalidRound
	}

	// Obtained valid round. Retrieve blocks.
	bs, hasEntity = sc.hasBlockSummary(ctx, r.BlockHash)
	if !hasEntity {
		// Missing block summary. Sync the blocks
		bs = sc.syncBlockSummary(ctx, r, sc.BatchSyncSize)
	}

	if bs == nil {
		// Unable to retrieve block summary.
		return MissingSummary
	}

	// Check for block presence.
	n := sc.GetActivesetSharder(self.GNode)
	canShard := sc.IsBlockSharderFromHash(bs.Hash, n)
	if canShard == false {
		return BlockSuccess
	}

	if canShard {
		b, hasEntity = sc.hasBlock(bs.Hash, r.Number)
		if !hasEntity {
			b = sc.syncBlock(ctx, r, canShard)
		}
		if b == nil {
			return MissingBlock
		}
	}

	hasTxns := sc.hasTransactions(ctx, bs)
	if hasTxns {
		// The block has transactions and needs to be stored.
		sc.storeBlockTransactions(ctx, b)
	}
	return BlockSuccess
}
