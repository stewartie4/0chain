package chain

import (
	"context"
	"time"

	"0chain.net/chaincore/node"
	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

/*SetupWorkers - setup a blockworker for a chain */
func (c *Chain) SetupWorkers(ctx context.Context) {
	go c.NodesMonitor(ctx)
	go c.PruneClientStateWorker(ctx)
	go c.BlockFetchWorker(ctx)
	//go node.Self.Node.MemoryUsage() ToDo: Fix This
}

/*NodesMonitor - a background job that keeps checking the status of the nodes */
func (c *Chain) NodesMonitor(ctx context.Context) {
	node.GNodeStatusMonitor(ctx)
	timer := time.NewTimer(time.Second)
	for true {
		select {
		case <-ctx.Done():
			Logger.Info("Done with NodesMonitor")

			return
		case _ = <-timer.C:
			node.GNodeStatusMonitor(ctx)
			var allActiveMiners, allActiveSharders, allDkgMinersActive bool
			if c.Miners != nil {
				allActiveMiners = c.Miners.GetActiveCount()*10 > len(c.Miners.Nodes)*8
				c.Miners.ComputeNetworkStats()
			}

			if c.Sharders != nil {
				allActiveSharders = c.Sharders.GetActiveCount()*10 > len(c.Sharders.Nodes)*8
				c.Sharders.ComputeNetworkStats()
			}
			dkgSet := c.GetDkgSetNodePool(NEXT)
			if dkgSet == nil {
				dkgSet = c.GetDkgSetNodePool(CURR)
			}

			if dkgSet != nil {
				allDkgMinersActive = dkgSet.GetActiveCount() == len(dkgSet.Nodes)
				dkgSet.ComputeNetworkStats()
			}
			if allActiveMiners && allActiveSharders && allDkgMinersActive {
				timer = time.NewTimer(5 * time.Second)
			} else {
				timer = time.NewTimer(2 * time.Second)
			}
		}
	}

}

/*FinalizeRoundWorker - a worker that handles the finalized blocks */
func (c *Chain) FinalizeRoundWorker(ctx context.Context, bsh BlockStateHandler) {
	for r := range c.finalizedRoundsChannel {
		c.finalizeRound(ctx, r, bsh)
		c.UpdateRoundInfo(r)
	}
}

//FinalizedBlockWorker - a worker that processes finalized blocks
func (c *Chain) FinalizedBlockWorker(ctx context.Context, bsh BlockStateHandler) {
	for fb := range c.finalizedBlocksChannel {
		if fb.Round < c.LatestFinalizedBlock.Round-5 {
			Logger.Error("slow finalized block processing", zap.Int64("lfb", c.LatestFinalizedBlock.Round), zap.Int64("fb", fb.Round))
		}
		c.finalizeBlock(ctx, fb, bsh)
	}
}

/*PruneClientStateWorker - a worker that prunes the client state */
func (c *Chain) PruneClientStateWorker(ctx context.Context) {
	tick := time.Duration(c.PruneStateBelowCount) * time.Second
	timer := time.NewTimer(time.Second)
	pruning := false
	for true {
		select {
		case <-timer.C:
			if pruning {
				Logger.Info("pruning still going on")
				continue
			}
			pruning = true
			c.pruneClientState(ctx)
			pruning = false
			if c.pruneStats == nil || c.pruneStats.MissingNodes > 0 {
				timer = time.NewTimer(time.Second)
			} else {
				timer = time.NewTimer(tick)
			}
		}
	}
}

/*BlockFetchWorker - a worker that fetches the prior missing blocks */
func (c *Chain) BlockFetchWorker(ctx context.Context) {
	for true {
		select {
		case b := <-c.blockFetcher.missingLinkBlocks:
			if b.PrevBlock != nil {
				continue
			}
			pb, err := c.GetBlock(ctx, b.PrevHash)
			if err == nil {
				b.SetPreviousBlock(pb)
				continue
			}

			c.blockFetcher.FetchPreviousBlock(ctx, c, b)
		case bHash := <-c.blockFetcher.missingBlocks:
			_, err := c.GetBlock(ctx, bHash)
			if err == nil {
				continue
			}
			c.blockFetcher.FetchBlock(ctx, c, bHash)
		}
	}
}
