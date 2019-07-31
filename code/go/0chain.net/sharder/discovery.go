package sharder

import (
	"0chain.net/chaincore/discovery"
	"0chain.net/chaincore/node"
	. "0chain.net/core/logging"
	"context"
	"go.uber.org/zap"
	"net/http"
	"net/url"
	"time"
)

// FRTSTYDiscoveryCounters - Counters
type DiscoveryCounters struct {
	Invocations            int64
	CheckViewChangeEvents  uint64
	UpdateMagicBlockEvents uint64
}

// NewDiscoveryControl - Used to create a new discovery control structure
func NewDiscoveryControl() *DiscoveryControl {
	dsControl := &DiscoveryControl{}
	dsControl.Fields = make(url.Values)
	return dsControl
}

// DiscoveryControl - Used to display counters
type DiscoveryControl struct {
	Bucket                  string
	Chain                   string

	// Last propogated block
	LastPropagatedBlockHash string
	LastPropagatedBlockRound int64

	MagicBlockHash          string
	Fields                  url.Values
}

func (dsControl *DiscoveryControl) init() {
	dc := discovery.Control

	// Get self node
	url := node.Self.GetURLBase()

	// Set the fields to send to the discovery server
	dsControl.Fields.Set("bucket", dc.Bucket)
	dsControl.Fields.Set("chain", dc.Chain)

	// Set the sharder url
	dsControl.Fields.Set("url", url)

	dsControl.Fields.Set("publickey", node.Self.Client.PublicKey)
	dsControl.Fields.Set("clientid", node.Self.Client.ID)

}

// DiscoveryWorker - Workers for discovery server
func (sc *Chain) DiscoveryWorker(ctx context.Context) {
	dsControl := sc.DSControl
	// dsStats := sc.DS_Stats

	// Initialize the control structure
	dsControl.init()

	Logger.Info("DS-Init")
	for true {
		select {
		case <-ctx.Done():
			return
		default:
			sc.DiscoveryWaitForWork(ctx)
		}
	}
}

// DSPropagateMagicBlock the magic block
func (sc *Chain) DSPropagateMagicBlock() {
	dc := discovery.Control

	dsControl := sc.DSControl
	dsStats := sc.DSStats

	// Get the finalized block
	// block := sc.GetLatestFinalizedBlock()

	// Update the block hash
	// Endpoint and bucket are assumed to be setup at initialization
	dsControl.Fields.Set("block", dsControl.LastPropagatedBlockHash)

	// Send request to the discovery server
	resp, err := http.PostForm(dc.MagicBlockURL.String(), dsControl.Fields)
	if err == nil {
		defer resp.Body.Close()
	} else {
		Logger.Error("DS-Post", zap.Error(err), zap.Any("post", dc.MagicBlockURL))
	}
	dsStats.UpdateMagicBlockEvents++
}

// DSViewChangeUpdated -
func (sc *Chain) DSViewChangeUpdated(ctx context.Context) bool {
	dsStats := sc.DSStats
	dsControl := sc.DSControl
	dsStats.CheckViewChangeEvents++

	block := sc.GetLatestFinalizedBlock()
	newMagicRound := (block.Round / 100) * 100
	if newMagicRound > dsControl.LastPropagatedBlockRound {
		// This would be the new magic block.
		magicBlock, err := sc.GetRoundFromStore(ctx, newMagicRound)
		if err == nil {
			dsControl.LastPropagatedBlockRound = magicBlock.Number
			dsControl.LastPropagatedBlockHash = magicBlock.BlockHash
			return true
		}
	}
	return false
}

// DiscoveryWaitForWork -
func (sc *Chain) DiscoveryWaitForWork(ctx context.Context) {
	dc := discovery.Control

	startOuter := time.Now()
	propagateMagicBlock := startOuter.Add(dc.PropagateMagicBlock)

	for ok := true; ok; ok = propagateMagicBlock.After(time.Now()) {
		startInner := time.Now()
		time.Sleep(dc.MonitorViewChange)
		// Time to check if there are any view changes.
		if sc.DSViewChangeUpdated(ctx) {
			sc.DSPropagateMagicBlock()
		}
		Logger.Debug("DS-WaitForWork",
			zap.String("block", "MonitorViewChange.."),
			zap.Time("entry", startInner),
			zap.Time("exit", time.Now()),
			zap.Duration("sleep", dc.MonitorViewChange))
	}

	// Side effect is to update the last propogated block.
	_  = sc.DSViewChangeUpdated(ctx)

	sc.DSPropagateMagicBlock()

	Logger.Debug("DS-WaitForWork",
		zap.String("block", "PropagateMagicBlock"),
		zap.Time("entry", startOuter),
		zap.Time("exit", time.Now()),
		zap.Duration("sleep", dc.PropagateMagicBlock))

}
