package sharder

import (
	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	. "0chain.net/core/logging"
	"context"
	"fmt"
	"go.uber.org/zap"
	"time"
	"github.com/rcrowley/go-metrics"

)

var HealthCheckDateTimeFormat = "2006-01-02T15:04:05"

type BlockStatus int

const (
	BlockSuccess BlockStatus = 1 + iota
	MissingRoundSummary
	MissingBlockSummary
	MissingBlock
	MissingTxnSummary
)

type HealthCheckScan int

const (
	DeepScan HealthCheckScan = iota
	ProximityScan
)

func (e HealthCheckScan) String() string {
	modeNames := []string{"Deep.....", "Proximity"}
	return modeNames[e]
}

type HealthCheckStatus string

const (
	SyncProgress HealthCheckStatus = "syncing"
	SyncHiatus                     = "hiatus"
	SyncDone                       = "synced"
)

type BlockCounters struct {
	CycleIteration int64
	CycleStart     time.Time
	CycleEnd       time.Time
	CycleDuration  time.Duration

	// Sweep Rate for blocks
	SweepCount     int64
	ElapsedSeconds int64
	SweepRate      int64

	BlockSuccess        int64
	MissingRoundSummary int64
	MissingBlockSummary int64
	InvalidRound        int64
	MissingSummary      int64
	MissingBlock        int64
	MissingTxnSummary   int64
}

func (bc *BlockCounters) init() {
	bc.CycleStart = time.Now().Truncate(time.Second)
	bc.CycleEnd = time.Time{}
	bc.CycleDuration = 0

	// Clear sweep rate counters
	bc.SweepRate = 0
	bc.ElapsedSeconds = 0
	bc.SweepCount = 0

	bc.BlockSuccess = 0
	bc.MissingRoundSummary = 0
	bc.MissingBlockSummary = 0
	bc.MissingTxnSummary = 0
	bc.MissingBlock = 0
	bc.MissingSummary = 0


}

type CycleCounters struct {
	ScanMode HealthCheckScan

	current  BlockCounters
	previous BlockCounters
}

func (cc *CycleCounters) transfer() {
	cc.previous = cc.current
}

type CycleBounds struct {
	window       int64
	lowRound     int64
	currentRound int64
	highRound    int64
}

type CycleControl struct {
	ScanMode HealthCheckScan
	Status   HealthCheckStatus

	inception time.Time
	bounds    CycleBounds

	CycleCount  int64
	Invocations int64

	BlockSyncTimer metrics.Timer

	counters CycleCounters
}

func (bss *SyncStats) getCycleControl(scanMode HealthCheckScan) *CycleControl {
	return &bss.cycle[scanMode]
}

type SyncStats struct {
	cycle [2]CycleControl

	//deepScan CycleStats
	//proximityScan CycleStats

	// ScanMode HealthCheckScan

	// Status HealthCheckStatus

	// Interval bounds to start, current and final.
	//LowRound     int64
	// CurrentRound int64
	//HighRound    int64

	// deepScan      CycleCounters
	// proximityScan CycleCounters

}

func (sc *Chain) setCycleBounds(ctx context.Context, scanMode HealthCheckScan) {
	bss := sc.BlockSyncStats
	cb := &bss.cycle[scanMode].bounds

	// Clear old bounds
	*cb = CycleBounds{}
	config := &sc.HC_CycleScan[scanMode]
	cb.window = config.Window

	roundEntity, err := sc.GetMostRecentRoundFromDB(ctx)
	if err == nil {
		cb.highRound = roundEntity.Number

		// Start from the high round
		cb.currentRound = cb.highRound
		if cb.window == 0 || cb.window > cb.highRound {
			// Cover entire blockchain.
			cb.lowRound = 1
		} else {
			cb.lowRound = cb.highRound - cb.window + 1
		}
	}
}

/*HealthCheckWorker - checks the health for each round*/

func (sc *Chain) HealthCheckWorker(ctx context.Context, scanMode HealthCheckScan) {
	bss := sc.BlockSyncStats

	// Get the configuration
	config := &sc.HC_CycleScan[scanMode]

	// Get cycle control
	cc := bss.getCycleControl(scanMode)

	// Update the scan mode.
	cc.ScanMode = scanMode
	cc.inception = time.Now()

	if config.Enabled == false {

		// Scan is disabled. Print event periodically.
		wakeToReport := config.ReportStatus
		for true  {
		Logger.Info("HC-CycleHistory",
			zap.String("scan", scanMode.String()),
			zap.Bool("enabled", config.Enabled))
			time.Sleep(wakeToReport)
		}
	}

	cc.BlockSyncTimer = metrics.GetOrRegisterTimer(scanMode.String(), nil)

	// Set the cycle bounds
	sc.setCycleBounds(ctx, scanMode)
	cb := &cc.bounds

	Logger.Info("HC-Init",
		zap.String("mode", scanMode.String()),
		zap.Int64("high", cb.highRound),
		zap.Int64("low", cb.lowRound),
		zap.Int64("current", cb.currentRound),
		zap.Int64("window", cb.window))

	Logger.Info("HC-Init",
		zap.String("mode", scanMode.String()),
		zap.Int("batch-size", config.BatchSize),
		zap.Int("interval", config.IntervalMins))

	// Initial setup
	Logger.Info("HC-Init",
		zap.String("mode", scanMode.String()),
		zap.Time("inception", cc.inception))

	// Initialize the health check statistics
	sc.initSyncStats(ctx, scanMode)

	for true {
		select {
		case <-ctx.Done():
			return
		default:
			cc.Status = SyncProgress
			for cb.currentRound = cb.highRound; cb.currentRound >= cb.lowRound; cb.currentRound-- {
				t := time.Now()
				blockStatus := sc.healthCheck(ctx, cb.currentRound, scanMode)
				duration := time.Since(t)

				// Update the statistics
				sc.updateSyncStats(ctx, cb.currentRound, duration, blockStatus, scanMode)
			}

			// Wait for new work.
			sc.waitForWork(ctx, scanMode)
		}
	}
}

func (sc *Chain) initSyncStats(ctx context.Context, scanMode HealthCheckScan) {

	bss := sc.BlockSyncStats

	// Get cycle control
	cc := bss.getCycleControl(scanMode)

	// Bounds for the round.
	bounds := cc.bounds

	// Update the cycle count.
	cc.CycleCount++

	// Clear the number of invocations for this next cycle
	cc.Invocations = 0

	// Copy current to previous cycle.
	cc.counters.transfer()

	// Clear current cycle counters
	cc.counters.current.init()

	// Log start of new round
	Logger.Info("HC-CycleHistory",
		zap.String("mode", cc.ScanMode.String()),
		zap.Int64("cycle", cc.CycleCount),
		zap.String("event", "start"),
		zap.String("bounds",
			fmt.Sprintf("[%v-%v]", bounds.highRound, bounds.lowRound)),
		zap.Time("start", cc.counters.current.CycleStart.Truncate(time.Second)))
}

func (sc *Chain) updateSyncStats(ctx context.Context, current int64, duration time.Duration, status BlockStatus,
	scanMode HealthCheckScan) {

	// var highRound int64
	bss := sc.BlockSyncStats

	// Get cycle control
	cc := bss.getCycleControl(scanMode)

	// Get the current cycle counters.
	counters := &cc.counters.current

	// Update the number of invocations
	cc.Invocations++

	// Update the timer. Common for both scans.
	cc.BlockSyncTimer.Update(duration)

	BlockSyncTimer.Update(duration)

	switch status {
	case BlockSuccess:
		counters.BlockSuccess++
	case MissingRoundSummary:
		counters.MissingRoundSummary++
	case MissingBlockSummary:
		counters.MissingBlockSummary++
	case MissingBlock:
		counters.MissingBlock++
	case MissingTxnSummary:
		counters.MissingTxnSummary++
	}
}

func (sc *Chain) waitForWork(ctx context.Context, scanMode HealthCheckScan) {
	bss := sc.BlockSyncStats

	// Get cycle control
	cc := bss.getCycleControl(scanMode)

	// Get the current cycle
	bc := &cc.counters.current

	// Bounds for the round.
	bounds := cc.bounds

	for true {
		if bounds.currentRound > bounds.lowRound {
			// Not reached the round bounds.
			break;
		}

		// Log end of the current cycle
		bc.CycleEnd = time.Now().Truncate(time.Second)
		bc.CycleDuration = time.Since(bc.CycleStart).Truncate(time.Second)
		bc.ElapsedSeconds = int64(bc.CycleDuration.Seconds())

		// Mark as cycle is in hiatus
		cc.Status = SyncHiatus

		Logger.Info("HC-CycleHistory",
			zap.String("mode", cc.ScanMode.String()),
			zap.Int64("cycle", cc.CycleCount),
			zap.String("event", "end"),
			zap.String("bounds",
				fmt.Sprintf("[%v-%v]", bounds.highRound, bounds.lowRound)),
			zap.Time("start", bc.CycleStart.Truncate(time.Second)),
			zap.Time("end", bc.CycleEnd.Truncate(time.Second)),
			zap.Duration("duration", bc.CycleDuration))

		// Calculate the sweep rate
		bc.SweepCount = bounds.highRound - bounds.lowRound + 1

		if bc.ElapsedSeconds > 0 {
			bc.SweepRate = bc.SweepCount / bc.ElapsedSeconds
		}

		Logger.Info("HC-CycleHistory",
			zap.String("mode", cc.ScanMode.String()),
			zap.Int64("cycle", cc.CycleCount),
			zap.String("event", "sweep-rate"),
			zap.Int64("BlocksSweeped", bc.SweepCount),
			zap.Int64("ElapsedSeconds", bc.ElapsedSeconds),
			zap.Int64("SweepRate", bc.SweepRate))

		// End of the cycle. Sleep between cycles.
		config := &sc.HC_CycleScan[scanMode]

		sleepTime := config.Interval
		wakeToReport := config.ReportStatus
		if wakeToReport > sleepTime {
			wakeToReport = sleepTime
		}

		// Add time to sleep before waking up
		restartCycle := time.Now().Add(sleepTime)
		for ok := true; ok; ok = restartCycle.After(time.Now()) {
			Logger.Info("HC-CycleHistory",
				zap.String("mode", cc.ScanMode.String()),
				zap.Int64("cycle", cc.CycleCount),
				zap.String("event", "hiatus"),
				zap.Time("restart", restartCycle.Truncate(time.Second)))
			time.Sleep(wakeToReport)
		}

		// Time to start a new cycle
		sc.setCycleBounds(ctx, scanMode)
		sc.initSyncStats(ctx, scanMode)
		break;
	}
}

func (sc *Chain) healthCheck(ctx context.Context, rNum int64, scanMode HealthCheckScan) BlockStatus {
	bss := sc.BlockSyncStats
	config := &sc.HC_CycleScan[scanMode]
	// Get cycle control
	cc := bss.getCycleControl(scanMode)

	var r *round.Round
	var bs *block.BlockSummary
	var b *block.Block
	var hasEntity bool

	self := node.GetSelfNode(ctx)

	r, hasEntity = sc.hasRoundSummary(ctx, rNum)
	if !hasEntity {
		// No round found. Fetch the round summary and round information.
		r = sc.syncRoundSummary(ctx, rNum, config.BatchSize, scanMode)
	}

	if sc.isValidRound(r) == false {
		// Unable to get the round summary information.
		return MissingRoundSummary
	}

	// Obtained valid round. Retrieve blocks.
	bs, hasEntity = sc.hasBlockSummary(ctx, r.BlockHash)
	if !hasEntity {
		// Missing block summary. Sync the blocks
		bs = sc.syncBlockSummary(ctx, r, config.BatchSize, scanMode)
	}

	if bs == nil {
		// Unable to retrieve block summary.
		return MissingBlockSummary
	}

	// Check for block presence.
	n := sc.GetActivesetSharder(self.GNode)
	canShard := sc.IsBlockSharderFromHash(bs.Hash, n)

	// The sharder has block summary. Check if the sharder has txn_summary.
	needTxnSummary := false
	if bs.NumTxns > 0 {
		count, err := sc.getTxnAndCountForRound(ctx, bs.Round)
		if err != nil || count != bs.NumTxns {
				needTxnSummary = true
		}
	}

	// The sharder needs txn_summary. Get the block
	b, hasEntity = sc.hasBlock(bs.Hash, r.Number)
	if hasEntity == false {
		// The sharder doesn't have the block.
		if needTxnSummary || canShard {
			b = sc.requestBlock(ctx, r)
			if b == nil {
				Logger.Info("HC-MissingObject",
					zap.String("mode", cc.ScanMode.String()),
					zap.Int64("cycle", cc.CycleCount),
					zap.String("object", "Block"),
					zap.Int64("round", r.Number),
					zap.String("hash", r.BlockHash))
				return MissingBlock
			}
		}

		// The sharder has the block.
		if canShard {
			// Save the block
			err := sc.storeBlock(ctx, b)
			if err != nil {
				Logger.Error("HC-DSWriteFailure",
					zap.String("mode", cc.ScanMode.String()),
					zap.Int64("cycle", cc.CycleCount),
					zap.String("object", "block"),
					zap.Int64("round", r.Number),
					zap.Error(err))
				return MissingBlock
			}
		}
	}
	if needTxnSummary {
		if b == nil {
			Logger.Panic("HC-Assertion",
				zap.String("mode", cc.ScanMode.String()),
				zap.Int64("cycle", cc.CycleCount),
				zap.String("object", "block"),
				zap.Int64("round", r.Number),
				zap.String("Missing block", bs.Hash))
		}
		// The block has transactions and may need to be stored.
		err := sc.storeBlockTransactions(ctx, b)
		if err != nil {
			Logger.Error("HC-DSWriteFailure",
				zap.String("mode", cc.ScanMode.String()),
				zap.Int64("cycle", cc.CycleCount),
				zap.String("object", "TransactionSummary"),
				zap.Int64("round", bs.Round),
				zap.Int("txn-count", bs.NumTxns),
				zap.String("block-hash", bs.Hash),
				zap.Error(err))
			return MissingTxnSummary
		}
	}
	return BlockSuccess
}
