package sharder

import (
	"0chain.net/core/common"
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

func (sc *Chain) WriteBlockSyncStats(w http.ResponseWriter) {
	bss := sc.BlockSyncStats
	elapsed := time.Since(bss.current.CycleStart).Round(time.Minute)
	fmt.Fprintf(w, "<table width='100%%'>")
	fmt.Fprintf(w, "<tr><td class='sheader' colspan=2'>Invocation History</td></tr>")
	fmt.Fprintf(w, "<tr><td>Status</td><td class='string'>%v</td></tr>", bss.Status)

	fmt.Fprintf(w, "<tr><td>Inception</td><td class='string'>%v</td></tr>",
		bss.Inception.Format(common.DateTimeFormat))

	fmt.Fprintf(w, "<tr><td>Cycle Count</td><td class='string'>%v</td></tr>", bss.CycleCount)
	fmt.Fprintf(w, "<tr><td>Cycle Start (Elapsed)</td><td class='string'>%v (%v)</td></tr>",
		bss.current.CycleStart.Format(common.DateTimeFormat), elapsed)
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

func (sc *Chain) WriteHealthCheckCounters(w http.ResponseWriter) {
	bss := sc.BlockSyncStats
	fmt.Fprintf(w, "<table width='100%%'>")
	fmt.Fprintf(w, "<tr>" +
		"<td class='sheader' colspan=1'>Round Statistics</td>" +
		"<td class='sheader' colspan=1'>Current</td>" +
		"<td class='sheader' colspan=1'>Previous</td>" +
		"</tr>")
	fmt.Fprintf(w, "<tr><td>Rounds with Blocks</td><td class='string'>%v</td><td class='string'>%v</td></tr>", bss.current.BlockSuccess, bss.previous.BlockSuccess)
	fmt.Fprintf(w, "<tr><td>Rounds w/o Blocks</td><td class='string'>%v</td><td class='string'>%v</td></tr>", bss.current.MissingBlock, bss.previous.MissingBlock)
	fmt.Fprintf(w, "<tr><td>Rounds w/o Summary</td><td class='string'>%v</td><td class='string'>%v</td></tr>", bss.current.MissingSummary, bss.previous.MissingSummary)
	fmt.Fprintf(w, "<tr><td>Rounds w/o Hash</td><td class='string'>%v</td><td class='string'>%v</td></tr>", bss.current.InvalidRound, bss.previous.InvalidRound)
	fmt.Fprintf(w, "</table>")

}
