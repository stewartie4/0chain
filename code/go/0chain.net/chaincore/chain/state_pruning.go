package chain

import (
	"0chain.net/chaincore/node"
	"context"
	"fmt"
	"time"

	"0chain.net/chaincore/block"
	. "0chain.net/core/logging"
	"0chain.net/core/util"
	metrics "github.com/rcrowley/go-metrics"
	"github.com/remeh/sizedwaitgroup"
	"go.uber.org/zap"
)

//StatePruneUpdateTimer - a metric that tracks the time it takes to update older nodes still referrred from the given version
var StatePruneUpdateTimer metrics.Timer

//StatePruneDeleteTimer - a metric that tracks the time it takes to delete all the obsolete nodes w.r.t a given version
var StatePruneDeleteTimer metrics.Timer

func init() {
	StatePruneUpdateTimer = metrics.GetOrRegisterTimer("state_prune_update_timer", nil)
	StatePruneDeleteTimer = metrics.GetOrRegisterTimer("state_prune_delete_timer", nil)
}

func (c *Chain) pruneClientState(ctx context.Context) {
	bc := c.BlockChain
	bc = bc.Move(-c.PruneStateBelowCount)
	for i := 0; i < 10 && bc.Value == nil; i++ {
		bc = bc.Prev()
	}
	var bs *block.BlockSummary
	lfb := c.LatestFinalizedBlock
	if bc.Value != nil {
		bs = bc.Value.(*block.BlockSummary)
		for bs.Round%100 != 0 {
			bc = bc.Prev()
			if bc.Value == nil {
				break
			}
			bs = bc.Value.(*block.BlockSummary)
		}
	} else {
		if lfb.Round == 0 {
			return
		}
	}
	if bs == nil {
		bs = &block.BlockSummary{Round: lfb.Round, ClientStateHash: lfb.ClientStateHash}
	}
	Logger.Info("prune client state - new version", zap.Int64("current_round", c.CurrentRound), zap.Int64("latest_finalized_round", c.LatestFinalizedBlock.Round), zap.Int64("round", bs.Round), zap.String("block", bs.Hash), zap.String("state_hash", util.ToHex(bs.ClientStateHash)))
	newVersion := util.Sequence(bs.Round)
	if c.pruneStats != nil && c.pruneStats.Version == newVersion && c.pruneStats.MissingNodes == 0 {
		return // already done with pruning this
	}
	mpt := util.NewMerklePatriciaTrie(c.stateDB, newVersion)
	mpt.SetRoot(bs.ClientStateHash)
	pctx := util.WithPruneStats(ctx, c.ID)
	ps := util.GetPruneStats(pctx, c.ID)
	ps.Stage = util.PruneStateUpdate
	c.pruneStats = ps
	t := time.Now()
	var missingKeys []util.Key
	wg := sizedwaitgroup.New(2)
	missingNodesHandler := func(ctx context.Context, path util.Path, key util.Key) error {
		missingKeys = append(missingKeys, key)
		if len(missingKeys) == 1000 {
			ps.Stage = util.PruneStateSynch
			wg.Add()
			go func(nodes []util.Key) {
				c.GetStateNodes(ctx, nodes)
				wg.Done()
			}(missingKeys[:])
			missingKeys = nil
		}
		return nil
	}
	var stage = ps.Stage
	err := mpt.UpdateVersion(pctx, newVersion, missingNodesHandler, c.ID)
	wg.Wait()
	ps.Stage = stage
	d1 := time.Since(t)
	ps.UpdateTime = d1
	StatePruneUpdateTimer.Update(d1)
	node.GetSelfNode(ctx).Info.StateMissingNodes = ps.MissingNodes
	if err != nil {
		Logger.Error("prune client state (update origin)", zap.Int64("current_round", c.CurrentRound), zap.Int64("round", bs.Round), zap.String("block", bs.Hash), zap.String("state_hash", util.ToHex(bs.ClientStateHash)), zap.Any("prune_stats", ps), zap.Error(err))
		if ps.MissingNodes > 0 {
			if len(missingKeys) > 0 {
				c.GetStateNodes(ctx, missingKeys[:])
			}
			ps.Stage = util.PruneStateAbandoned
			return
		}
	} else {
		Logger.Info("prune client state (update origin)", zap.Int64("current_round", c.CurrentRound), zap.Int64("round", bs.Round), zap.String("block", bs.Hash), zap.String("state_hash", util.ToHex(bs.ClientStateHash)), zap.Any("prune_stats", ps))
	}
	if c.LatestFinalizedBlock.Round-int64(c.PruneStateBelowCount) < bs.Round {
		ps.Stage = util.PruneStateAbandoned
		return
	}
	t1 := time.Now()
	ps.Stage = util.PruneStateDelete
	err = c.stateDB.PruneBelowVersion(pctx, newVersion, c.ID)
	if err != nil {
		Logger.Error("prune client state error", zap.Error(err))
	}
	ps.Stage = util.PruneStateCommplete
	d2 := time.Since(t1)
	ps.DeleteTime = d2
	StatePruneDeleteTimer.Update(d2)
	logf := Logger.Info
	if d1 > time.Second || d2 > time.Second {
		logf = Logger.Error
	}
	logf("prune client state stats", zap.Int64("round", bs.Round), zap.String("block", bs.Hash), zap.String("state_hash", util.ToHex(bs.ClientStateHash)),
		zap.Duration("duration", time.Since(t)), zap.Any("stats", ps))
	/*
		if stateOut != nil {
			if err = util.IsMPTValid(mpt); err != nil {
				fmt.Fprintf(stateOut, "prune validation failure: %v %v\n", util.ToHex(mpt.GetRoot()), bs.Round)
				mpt.PrettyPrint(stateOut)
				stateOut.Sync()
				panic(err)
			}
		}*/
}

func (c *Chain) pruneSCStates(ctx context.Context, address string) {
	bc := c.BlockChain
	bc = bc.Move(-c.PruneStateBelowCount)
	for i := 0; i < 10 && bc.Value == nil; i++ {
		bc = bc.Prev()
	}
	var bs *block.BlockSummary
	lfb := c.LatestFinalizedBlock
	if bc.Value != nil {
		bs = bc.Value.(*block.BlockSummary)
		for bs.Round%100 != 0 {
			bc = bc.Prev()
			if bc.Value == nil {
				break
			}
			bs = bc.Value.(*block.BlockSummary)
		}
	} else {
		if lfb.Round == 0 {
			return
		}
	}
	if bs == nil {
		bs = &block.BlockSummary{Round: lfb.Round, ClientStateHash: lfb.ClientStateHash}
	}
	newVersion := util.Sequence(bs.Round)
	scps := c.GetSCPruneStats(address)
	if scps != nil && scps.Version == newVersion && scps.MissingNodes == 0 {
		return // already done with pruning this
	}
	db, _ := c.GetSCDB(address)
	mpt := util.NewMerklePatriciaTrie(db, newVersion)
	mpt.SetRoot(lfb.SCStatesHashes[address])
	pctx := util.WithPruneStats(ctx, address)
	ps := util.GetPruneStats(pctx, address)
	ps.Stage = util.PruneStateUpdate
	c.SetSCPruneStats(address, ps)
	t := time.Now()
	var missingKeys []util.Key
	missingNodesHandler := func(ctx context.Context, path util.Path, key util.Key) error {
		missingKeys = append(missingKeys, key)
		if len(missingKeys) == 1000 {
			stage := ps.Stage
			ps.Stage = util.PruneStateSynch
			c.GetSCStateNodes(ctx, address, missingKeys[:])
			ps.Stage = stage
			missingKeys = nil
		}
		return nil
	}
	err := mpt.UpdateVersion(pctx, newVersion, missingNodesHandler, address)
	d1 := time.Since(t)
	ps.UpdateTime = d1
	StatePruneUpdateTimer.Update(d1)
	if err != nil {
		Logger.Error("prune client state (update origin)", zap.Int64("current_round", c.CurrentRound), zap.Int64("round", bs.Round), zap.String("block", bs.Hash), zap.Any("prune_stats", ps), zap.Error(err))
		if ps.MissingNodes > 0 {
			if len(missingKeys) > 0 {
				c.GetSCStateNodes(ctx, address, missingKeys[:])
			}
			ps.Stage = util.PruneStateAbandoned
			return
		}
	} else {
		Logger.Info("prune client state (update origin)", zap.Int64("current_round", c.CurrentRound), zap.Int64("round", bs.Round), zap.String("block", bs.Hash), zap.Any("prune_stats", ps))
	}
	if c.LatestFinalizedBlock.Round-int64(c.PruneStateBelowCount) < bs.Round {
		ps.Stage = util.PruneStateAbandoned
		return
	}
	ps.Stage = util.PruneStateDelete
	t1 := time.Now()
	err = db.PruneBelowVersion(pctx, newVersion, address)
	if err != nil {
		Logger.Error("prune client state error", zap.Error(err))
	}
	ps.Stage = util.PruneStateCommplete
	d2 := time.Since(t1)
	ps.DeleteTime = d2
	StatePruneDeleteTimer.Update(d2)
	logf := Logger.Info
	if d1 > time.Second || d2 > time.Second {
		logf = Logger.Error
	}
	logf("prune sc state stats", zap.Int64("round", bs.Round), zap.String("block", bs.Hash),
		zap.Duration("duration", time.Since(t1)))
}

func (c *Chain) LogSCDB(ctx context.Context, pndb util.NodeDB, where string) {
	handler := func(ctx context.Context, key util.Key, node util.Node) error {
		if node != nil {
			Logger.Info(fmt.Sprintf("iterate through scdb -- %v", where), zap.Any("key", util.ToHex(key)), zap.Any("node", string(node.Encode())))
		} else {
			Logger.Info(fmt.Sprintf("iterate through scdb -- node is nil -- %v", where), zap.Any("key", util.ToHex(key)))
		}
		return nil
	}
	Logger.Info(fmt.Sprintf("start iterate scdb %v", where), zap.Any("scdb_size", pndb.Size(ctx)))
	err := pndb.Iterate(ctx, handler)
	if err == nil {
		Logger.Info(fmt.Sprintf("finish iterate scdb %v", where), zap.Any("scdb_size", pndb.Size(ctx)))
	} else {
		Logger.Error(fmt.Sprintf("finish iterate scdb %v", where), zap.Any("scdb_size", pndb.Size(ctx)), zap.Any("error", err))
	}
}
