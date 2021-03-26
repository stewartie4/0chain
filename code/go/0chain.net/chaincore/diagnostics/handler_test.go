package diagnostics

import (
	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/core/common"
	"0chain.net/core/memorystore"
	"0chain.net/core/util"
	"fmt"
	"github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
)

func init() {
	sp := memorystore.GetStorageProvider()
	block.SetupEntity(sp)

	if err := os.MkdirAll("data/rocksdb/config", 0700); err != nil {
		panic(err)
	}
	chain.SetupEntity(sp)
	if err := os.RemoveAll("data"); err != nil {
		panic(err)
	}

	common.ConfigRateLimits()
	SetupHandlers()
}

func TestGetStatistics(t *testing.T) {
	c, ok := chain.Provider().(*chain.Chain)
	if !ok {
		t.Error("expected chain")
	}
	c.BlockSize = 5
	c.CurrentRound = 2
	c.LatestFinalizedBlock = block.NewBlock("", 2)

	var (
		timer   = metrics.NewTimer()
		scaleBy = 2.0

		lfb  = c.GetLatestFinalizedBlock()
		want = map[string]interface{}{
			"delta":                  chain.DELTA,
			"block_size":             c.BlockSize,
			"current_round":          c.GetCurrentRound(),
			"latest_finalized_round": lfb.Round,
			"count":                  timer.Count(),
			"min":                    float64(timer.Min()) / scaleBy,
			"mean":                   timer.Mean() / scaleBy,
			"std_dev":                timer.StdDev() / scaleBy,
			"max":                    float64(timer.Max()) / scaleBy,
			"total_txns":             lfb.RunningTxnCount,
			"rate_1_min":             timer.Rate1(),
			"rate_5_min":             timer.Rate5(),
			"rate_15_min":            timer.Rate15(),
			"rate_mean":              timer.RateMean(),
		}
	)

	percentiles := []float64{0.5, 0.9, 0.95, 0.99}
	pvals := timer.Percentiles(percentiles)
	for idx, p := range percentiles {
		want[fmt.Sprintf("percentile_%v", 100*p)] = pvals[idx] / 2
	}

	type args struct {
		c       *chain.Chain
		timer   metrics.Timer
		scaleBy float64
	}
	tests := []struct {
		name string
		args args
		want interface{}
	}{
		{
			name: "OK",
			args: args{
				c:       c,
				timer:   timer,
				scaleBy: scaleBy,
			},
			want: want,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetStatistics(tt.args.c, tt.args.timer, tt.args.scaleBy); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetStatistics() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWriteStatisticsCSS(t *testing.T) {
	want := httptest.NewRecorder()
	if _, err := want.WriteString("<style>.sheader { color: orange; font-weight: bold; }</style>"); err != nil {
		t.Error(err)
	}

	type args struct {
		w http.ResponseWriter
	}
	tests := []struct {
		name  string
		args  args
		wantW http.ResponseWriter
	}{
		{
			name:  "OK",
			args:  args{w: httptest.NewRecorder()},
			wantW: want,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			WriteStatisticsCSS(tt.args.w)
			assert.Equal(t, tt.wantW, tt.args.w)
		})
	}
}

func TestWriteConfiguration(t *testing.T) {
	c, ok := chain.Provider().(*chain.Chain)
	if !ok {
		t.Error("expected chain")
	}
	want := httptest.NewRecorder()

	ws := "<table width='100%'>" +
		"<tr><th class='sheader' colspan='2'>Configuration <a href='v1/config/get'>...</a></th></tr>" +
		fmt.Sprintf("<tr><td class='tname'>Round Generators/Replicators</td><td>%d/%d</td></tr>",
			c.NumGenerators, c.NumReplicators) +
		fmt.Sprintf("<tr><td class='tname'>Block Size</td><td>%v - %v</td></tr>", c.MinBlockSize, c.BlockSize) +
		fmt.Sprintf("<tr><td class='tname'>Network Latency (Delta)</td><td>%v</td></tr>", chain.DELTA) +
		fmt.Sprintf("<tr><td class='tname'>Block Proposal Wait Time</td><td>%v (static)</td>",
			c.BlockProposalMaxWaitTime) +
		fmt.Sprintf("<tr><td class='tname'>Validation Batch Size</td><td>%d</td>", c.ValidationBatchSize) +
		"</table>"
	if _, err := want.WriteString(ws); err != nil {
		t.Error(err)
	}

	type args struct {
		w http.ResponseWriter
		c *chain.Chain
	}
	tests := []struct {
		name  string
		args  args
		wantW http.ResponseWriter
	}{
		{
			name:  "OK",
			args:  args{w: httptest.NewRecorder(), c: c},
			wantW: want,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			WriteConfiguration(tt.args.w, tt.args.c)
			assert.Equal(t, tt.wantW, tt.args.w)
		})
	}
}

func TestWriteTimerStatistics(t *testing.T) {
	c, ok := chain.Provider().(*chain.Chain)
	if !ok {
		t.Error("expected chain")
	}
	want := httptest.NewRecorder()
	scaleBy := 2.0
	timer := metrics.NewTimer()

	percentiles := []float64{0.5, 0.9, 0.95, 0.99, 0.999}
	pvals := timer.Percentiles(percentiles)
	ws := "<table width='100%'>" +
		"<tr><td class='sheader' colspan=2'>Metrics</td></tr>" +
		fmt.Sprintf("<tr><td>Count</td><td>%v</td></tr>", timer.Count()) +
		"<tr><td class='sheader' colspan='2'>Time taken</td></tr>" +
		fmt.Sprintf("<tr><td>Min</td><td>%.2f ms</td></tr>", float64(timer.Min())/scaleBy) +
		fmt.Sprintf("<tr><td>Mean</td><td>%.2f &plusmn;%.2f ms</td></tr>", timer.Mean()/scaleBy, timer.StdDev()/scaleBy) +
		fmt.Sprintf("<tr><td>Max</td><td>%.2f ms</td></tr>", float64(timer.Max())/scaleBy)
	for idx, p := range percentiles {
		ws += fmt.Sprintf("<tr><td>%.2f%%</td><td>%.2f ms</td></tr>", 100*p, pvals[idx]/scaleBy)
	}
	ws += "<tr><td class='sheader' colspan='2'>Rate per second</td></tr>" +
		fmt.Sprintf("<tr><td>Last 1-min rate</td><td>%.2f</td></tr>", timer.Rate1()) +
		fmt.Sprintf("<tr><td>Last 5-min rate</td><td>%.2f</td></tr>", timer.Rate5()) +
		fmt.Sprintf("<tr><td>Last 15-min rate</td><td>%.2f</td></tr>", timer.Rate15()) +
		fmt.Sprintf("<tr><td>Overall mean rate</td><td>%.2f</td></tr>", timer.RateMean()) +
		"</table>"

	if _, err := want.WriteString(ws); err != nil {
		t.Error(err)
	}

	type args struct {
		w       http.ResponseWriter
		c       *chain.Chain
		timer   metrics.Timer
		scaleBy float64
	}
	tests := []struct {
		name string
		args args
		want http.ResponseWriter
	}{
		{
			name: "OK",
			args: args{
				w:       httptest.NewRecorder(),
				c:       c,
				timer:   timer,
				scaleBy: scaleBy,
			},
			want: want,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			WriteTimerStatistics(tt.args.w, tt.args.c, tt.args.timer, tt.args.scaleBy)
			assert.Equal(t, tt.want, tt.args.w)
		})
	}
}

func TestWriteHistogramStatistics(t *testing.T) {
	var (
		want   = httptest.NewRecorder()
		metric = metrics.NewHistogram(new(metrics.NilSample))
	)

	percentiles := []float64{0.5, 0.9, 0.95, 0.99, 0.999}
	pvals := metric.Percentiles(percentiles)
	ws := "<table width='100%'>" +
		"<tr><td class='sheader' colspan=2'>Metrics</td></tr>" +
		fmt.Sprintf("<tr><td>Count</td><td>%v</td></tr>", metric.Count()) +
		"<tr><td class='sheader' colspan='2'>Metric Value</td></tr>" +
		fmt.Sprintf("<tr><td>Min</td><td>%.2f</td></tr>", float64(metric.Min())) +
		fmt.Sprintf("<tr><td>Mean</td><td>%.2f &plusmn;%.2f</td></tr>", metric.Mean(), metric.StdDev()) +
		fmt.Sprintf("<tr><td>Max</td><td>%.2f</td></tr>", float64(metric.Max()))
	for idx, p := range percentiles {
		ws += fmt.Sprintf("<tr><td>%.2f%%</td><td>%.2f</td></tr>", 100*p, pvals[idx])
	}
	ws += "</table>"

	if _, err := want.WriteString(ws); err != nil {
		t.Error(err)
	}

	type args struct {
		w      http.ResponseWriter
		c      *chain.Chain
		metric metrics.Histogram
	}
	tests := []struct {
		name string
		args args
		want http.ResponseWriter
	}{
		{
			name: "OK",
			args: args{w: httptest.NewRecorder(), metric: metric},
			want: want,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			WriteHistogramStatistics(tt.args.w, tt.args.c, tt.args.metric)
			assert.Equal(t, tt.want, tt.args.w)
		})
	}
}

func TestWriteCurrentStatus(t *testing.T) {
	c, ok := chain.Provider().(*chain.Chain)
	if !ok {
		t.Error("expected chain")
	}
	c.LatestFinalizedBlock = block.NewBlock("", 2)
	c.LatestFinalizedBlock.PrevBlock = block.NewBlock("", 1)
	c.LatestFinalizedBlock.PrevBlock.UniqueBlockExtensions = map[string]bool{
		"key": true,
	}
	c.LatestDeterministicBlock = block.NewBlock("", 0)
	want := httptest.NewRecorder()

	ws := "<table width='100%' >" +
		"<tr><th class='sheader' colspan='2'>Current Status</th></tr>" +
		fmt.Sprintf("<tr><td class='tname'>Current Round</td><td>%v</td></tr>", c.GetCurrentRound()) +
		fmt.Sprintf("<tr><td class='tname'>Finalized Round</td><td>%v (%v)</td></tr>",
			c.LatestFinalizedBlock.Round, len(c.LatestFinalizedBlock.UniqueBlockExtensions)) +
		fmt.Sprintf("<tr><td class='tname'>Deterministic Finalized Round</td><td>%v (%v)</td></tr>",
			c.LatestDeterministicBlock.Round, len(c.LatestDeterministicBlock.UniqueBlockExtensions)) +
		fmt.Sprintf("<tr><td class='tname'>Next round to be deterministic</td><td>%v (%v)</td></tr>", 1, 1) +
		"</table>"

	if _, err := want.WriteString(ws); err != nil {
		t.Error(err)
	}

	type args struct {
		w http.ResponseWriter
		c *chain.Chain
	}
	tests := []struct {
		name string
		args args
		want http.ResponseWriter
	}{
		{
			name: "OK",
			args: args{w: httptest.NewRecorder(), c: c},
			want: want,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			WriteCurrentStatus(tt.args.w, tt.args.c)
			assert.Equal(t, tt.want, tt.args.w)
		})
	}
}

func TestWritePruneStats(t *testing.T) {
	want := httptest.NewRecorder()
	ps := util.PruneStats{}

	ws := "<table>" +
		"<tr><th class='sheader' colspan='2'>Prune Stats</th></tr>" +
		fmt.Sprintf("<tr><td>Stage</td><td>%v</td>", ps.Stage) +
		fmt.Sprintf("<tr><td>Pruned Below Round</td><td class='number'>%v</td></tr>", ps.Version) +
		fmt.Sprintf("<tr><td>Missing Nodes</td><td class='number'>%v</td></tr>", ps.MissingNodes) +
		fmt.Sprintf("<tr><td>Total nodes</td><td class='number'>%v</td></tr>", ps.Total) +
		fmt.Sprintf("<tr><td>Leaf Nodes</td><td class='number'>%v</td></tr>", ps.Leaves) +
		fmt.Sprintf("<tr><td>Nodes Below Pruned Round</td><td class='number'>%v</td></tr>", ps.BelowVersion) +
		fmt.Sprintf("<tr><td>Update Time</td><td class='number'>%v</td>", ps.UpdateTime) +
		fmt.Sprintf("<tr><td>Deleted Nodes</td><td class='number'>%v</td></tr>", ps.Deleted) +
		fmt.Sprintf("<tr><td>Delete Time</td><td class='number'>%v</td>", ps.DeleteTime) +
		fmt.Sprintf("</table>")

	if _, err := want.WriteString(ws); err != nil {
		t.Error(err)
	}

	type args struct {
		w  http.ResponseWriter
		ps *util.PruneStats
	}
	tests := []struct {
		name string
		args args
		want http.ResponseWriter
	}{
		{
			name: "OK",
			args: args{w: httptest.NewRecorder(), ps: &ps},
			want: want,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			WritePruneStats(tt.args.w, tt.args.ps)
			assert.Equal(t, tt.want, tt.args.w)
		})
	}
}
