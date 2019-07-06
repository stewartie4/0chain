package sharder

import (
	"fmt"
	"net/http"

	"github.com/rcrowley/go-metrics"
)

var BlockSyncTimer metrics.Timer

func init() {
	BlockSyncTimer = metrics.GetOrRegisterTimer("block_sync_timer", nil)
}

//Stats - a struct to store various runtime stats of the chain
type Stats struct {
	ShardedBlocksCount int64
	HealthyRoundNum    int64
	QOSRound           int64
}

func (sc *Chain) WriteBlockSyncStats(w http.ResponseWriter) {
	var status string

	if sc.BSyncStats.Current < sc.BSyncStats.Final {
		status = SyncProgress
	} else {
		status = SyncDone
	}
	fmt.Fprintf(w, "<tr><td>Status</td><td class='string'>%v</td></tr>", status)
	fmt.Fprintf(w, "<tr><td>Invocations</td><td class='number'>%v</td></tr>", sc.BSyncStats.Invocations)
	fmt.Fprintf(w, "<tr><td>Sync Start</td><td class='number'>%v</td></tr>", sc.BSyncStats.HealthyRoundStart)
	fmt.Fprintf(w, "<tr><td>Sync Final</td><td class='number'>%v</td></tr>", sc.BSyncStats.Final)
	fmt.Fprintf(w, "<tr><td>Last Synced</td><td class='number'>%v</td></tr>", sc.BSyncStats.Current)
	fmt.Fprintf(w, "<tr><td>Processed Blocks</td><td class='number'>%v</td></tr>", sc.BSyncStats.ProcessedBlocks)
	fmt.Fprintf(w, "<tr><td>Pending Count</td><td class='number'>%v</td></tr>", sc.BSyncStats.Final-sc.BSyncStats.Current)
}
