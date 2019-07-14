package sharder

import (
	"0chain.net/chaincore/diagnostics"
	"fmt"
	"net/http"
	"time"

	"github.com/rcrowley/go-metrics"
)

var BlockSyncTimer metrics.Timer

func init() {
	BlockSyncTimer = metrics.GetOrRegisterTimer("block_sync_timer", nil)
}

//Stats - a struct to store various runtime stats of the chain
type Stats struct {
	ShardedBlocksCount int64

	QOSRound           int64

	// Repair block count as part of healthcheck
	RepairBlocksCount int64
	RepairBlocksFailure int64
}

func (sc *Chain) WriteHealthCheckConfiguration(w http.ResponseWriter, scan HealthCheckScan) {
	bss := sc.BlockSyncStats

	// Get cycle control
	cc := bss.getCycleControl(scan)
	bounds := &cc.bounds

	_ = cc

	// Get health check config
	config := &sc.HC_CycleScan[scan]

	fmt.Fprintf(w, "<table width='100%%'>")
	fmt.Fprintf(w, "<tr><td class='sheader' colspan=2'>Tunables</td></tr>")
	fmt.Fprintf(w, "<tr><td>Scan Enabled</td><td class='string'>%v</td></tr>",
		config.Enabled)
	fmt.Fprintf(w, "<tr><td>Repeat Interval (mins)</td><td class='string'>%v</td></tr>",
		config.IntervalMins)
	fmt.Fprintf(w, "<tr><td class='sheader' colspan=2'>Invocation History</td></tr>")
	fmt.Fprintf(w, "<tr><td>Inception</td><td class='string'>%v</td></tr>",
		cc.inception.Format(HealthCheckDateTimeFormat))
	fmt.Fprintf(w, "<tr><td>Repeat Interval (mins)</td><td class='string'>%v</td></tr>",
		config.IntervalMins)

	fmt.Fprintf(w, "<tr><td>Cycle Count</td><td class='string'>%v</td></tr>", cc.CycleCount)
	fmt.Fprintf(w, "<tr><td>Invocations</td><td class='string'>%v</td></tr>", cc.Invocations)

	fmt.Fprintf(w, "<tr><td class='sheader' colspan=2'>Cycle Bounds</td></tr>")
	fmt.Fprintf(w, "<tr><td>Batch Size</td><td class='string'>%v</td></tr>", config.BatchSize)

	var window string
	if config.Window == 0 {
		window = "Entire BlockChain"
	} else {
		window = fmt.Sprintf("%v", config.Window)
	}

	fmt.Fprintf(w, "<tr><td>Scan Window Size</td><td class='string'>%v</td></tr>", window)

	fmt.Fprintf(w, "<tr><td>High Limit</td><td class='string'>%v</td></tr>", bounds.highRound)
	fmt.Fprintf(w, "<tr><td>Low Limit</td><td class='string'>%v</td></tr>", bounds.lowRound)
	fmt.Fprintf(w, "<tr><td>Current</td><td class='string'>%v</td></tr>", bounds.currentRound)
	var pendingCount int64
	if bounds.currentRound > bounds.lowRound {
		pendingCount = bounds.currentRound - bounds.lowRound
	}
	fmt.Fprintf(w, "<tr><td>Pending</td><td class='string'>%v</td></tr>", pendingCount)
	fmt.Fprintf(w, "</table>")

}

func (sc *Chain) WriteHealthCheckBlockSummary(w http.ResponseWriter, scan HealthCheckScan) {
	bss := sc.BlockSyncStats
	// Get cycle control
	cc := bss.getCycleControl(scan)
	current := &cc.counters.current
	previous := &cc.counters.previous

	fmt.Fprintf(w, "<table width='100%%'>")
	fmt.Fprintf(w, "<tr>" +
		"<td class='sheader' colspan=1'>Invocation Status</td>" +
		"<td class='sheader' colspan=1'>Current</td>" +
		"<td class='sheader' colspan=1'>Previous</td>" +
		"</tr>")


	var previousStart, currentStart string
	var previousElapsed, currentElapsed  string
	var previousStatus, currentStatus string
	roundUnit := time.Minute
	if scan == ProximityScan {
		roundUnit = time.Second
	}
	if previous.CycleStart.IsZero() {
		previousStart = "n/a"
		previousElapsed = "n/a"
		previousStatus = "n/a"
	} else {
		previousStart = previous.CycleStart.Format(HealthCheckDateTimeFormat)
		previousElapsed = previous.CycleDuration.Round(roundUnit).String()
		previousStatus = SyncDone
	}

	if current.CycleStart.IsZero() {
		currentStart = "n/a"
		currentElapsed = "n/a"
		currentStatus = "n/a"
	} else {
		currentStart = current.CycleStart.Format(HealthCheckDateTimeFormat)
		switch cc.Status {
		case SyncHiatus:
			currentElapsed = current.CycleDuration.Round(roundUnit).String()
		case SyncProgress:
			currentElapsed = time.Since(current.CycleStart).Round(roundUnit).String()
		}
		currentStatus = string(cc.Status)
	}

	fmt.Fprintf(w, "<tr>" +
		"<td>Status</td>" +
		"<td class='string'>%v</td>" +
		"<td class='string'>%v</td></tr>",
		currentStatus,
		previousStatus)

	fmt.Fprintf(w, "<tr>" +
		"<td>Start</td>" +
		"<td class='string'>%v</td>" +
		"<td class='string'>%v</td></tr>",
			currentStart,
			previousStart)

	fmt.Fprintf(w, "<tr>" +
		"<td>Elapsed</td>" +
		"<td class='string'>%v</td>" +
		"<td class='string'>%v</td></tr>",
		currentElapsed,
		previousElapsed)

	fmt.Fprintf(w, "<tr></tr>")

	fmt.Fprintf(w, "<tr>" +
		"<td class='sheader' colspan=1'>Round Statistics</td>" +
		"<td class='sheader' colspan=1'>Current</td>" +
		"<td class='sheader' colspan=1'>Previous</td>" +
		"</tr>")

	fmt.Fprintf(w, "<tr><td>Complete Blocks</td>" +
		"<td class='string'>%v</td><td class='string'>%v</td></tr>",
		current.BlockSuccess, previous.BlockSuccess)

	fmt.Fprintf(w, "<tr><td>Missing Round Summary</td>" +
		"<td class='string'>%v</td><td class='string'>%v</td></tr>",
		current.MissingRoundSummary, previous.MissingRoundSummary)

	fmt.Fprintf(w, "<tr><td>Missing Block Summary</td>" +
		"<td class='string'>%v</td><td class='string'>%v</td></tr>",
		current.MissingBlockSummary, previous.MissingBlockSummary)

	fmt.Fprintf(w, "<tr><td>Missing Txn Summary</td>" +
		"<td class='string'>%v</td><td class='string'>%v</td></tr>",
		current.MissingTxnSummary, previous.MissingTxnSummary)

	fmt.Fprintf(w, "<tr><td>Missing Block</td>" +
		"<td class='string'>%v</td><td class='string'>%v</td></tr>",
		current.MissingBlock, previous.MissingBlock)


	fmt.Fprintf(w, "</table>")

}

func (sc *Chain) WriteBlockSyncStatistics(w http.ResponseWriter, scan HealthCheckScan) {
	bss := sc.BlockSyncStats
	// Get cycle control
	cc := bss.getCycleControl(scan)
	diagnostics.WriteTimerStatistics(w, sc.Chain, cc.BlockSyncTimer, 1000000.0)
}
