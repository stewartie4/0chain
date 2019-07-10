package sharder

import (
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
}

func (sc *Chain) WriteHealthCheckStatistics(w http.ResponseWriter) {
	bss := sc.BlockSyncStats
	fmt.Fprintf(w, "<table width='100%%'>")
	fmt.Fprintf(w, "<tr><td class='sheader' colspan=2'>Invocation History</td></tr>")
	fmt.Fprintf(w, "<tr><td>Inception</td><td class='string'>%v</td></tr>",
		bss.Inception.Format(HealthCheckDateTimeFormat))

	fmt.Fprintf(w, "<tr><td>Cycle Count</td><td class='string'>%v</td></tr>", bss.CycleCount)
	fmt.Fprintf(w, "<tr><td>Invocations</td><td class='string'>%v</td></tr>", bss.Invocations)
	fmt.Fprintf(w, "<tr><td>Hiatus Wait Count</td><td class='string'>%v</td></tr>", bss.WaitNewBlocks)

	fmt.Fprintf(w, "<tr><td class='sheader' colspan=2'>Cycle Bounds</td></tr>")
	fmt.Fprintf(w, "<tr><td>Batch Size</td><td class='string'>%v</td></tr>", sc.BatchSyncSize)
	fmt.Fprintf(w, "<tr><td>Initial</td><td class='string'>%v</td></tr>", bss.LowRound)
	fmt.Fprintf(w, "<tr><td>Current</td><td class='string'>%v</td></tr>", bss.CurrentRound)
	fmt.Fprintf(w, "<tr><td>Target</td><td class='string'>%v</td></tr>", bss.HighRound)
	fmt.Fprintf(w, "<tr><td>Pending</td><td class='string'>%v</td></tr>", bss.HighRound-bss.CurrentRound)
	fmt.Fprintf(w, "</table>")

}

func (sc *Chain) WritehHealthCheckBlockSummary(w http.ResponseWriter) {
	bss := sc.BlockSyncStats
	fmt.Fprintf(w, "<table width='100%%'>")
	fmt.Fprintf(w, "<tr>" +
		"<td class='sheader' colspan=1'>Invocation Status</td>" +
		"<td class='sheader' colspan=1'>Current</td>" +
		"<td class='sheader' colspan=1'>Previous</td>" +
		"</tr>")


	var previousStart string
	var previousElapsed string
	var previousStatus string
	if bss.previous.CycleStart.IsZero() {
		previousStart = "n/a"
		previousElapsed = "n/a"
		previousStatus = "n/a"
	} else {
		previousStart = bss.previous.CycleStart.Format(HealthCheckDateTimeFormat)
		previousElapsed = bss.previous.CycleDuration.Round(time.Minute).String()
		previousStatus = SyncDone
	}

	fmt.Fprintf(w, "<tr>" +
		"<td>Status</td>" +
		"<td class='string'>%v</td>" +
		"<td class='string'>%v</td></tr>",
		bss.Status,
		previousStatus)

	fmt.Fprintf(w, "<tr>" +
		"<td>Start</td>" +
		"<td class='string'>%v</td>" +
		"<td class='string'>%v</td></tr>",
			bss.current.CycleStart.Format(HealthCheckDateTimeFormat),
			previousStart)

	currentElapsed := time.Since(bss.current.CycleStart).Round(time.Minute)
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
		bss.current.BlockSuccess, bss.previous.BlockSuccess)

	fmt.Fprintf(w, "<tr><td>Missing Round Summary</td>" +
		"<td class='string'>%v</td><td class='string'>%v</td></tr>",
		bss.current.MissingRoundSummary, bss.previous.MissingRoundSummary)

	fmt.Fprintf(w, "<tr><td>Missing Block Summary</td>" +
		"<td class='string'>%v</td><td class='string'>%v</td></tr>",
		bss.current.MissingBlockSummary, bss.previous.MissingBlockSummary)

	fmt.Fprintf(w, "<tr><td>Missing Block</td>" +
		"<td class='string'>%v</td><td class='string'>%v</td></tr>",
		bss.current.MissingBlock, bss.previous.MissingBlock)


	fmt.Fprintf(w, "</table>")

}
