// +build state_debug_blck state_debug_txn

package chain

import (
	"0chain.net/chaincore/state/debug"
	"bytes"
	"context"
	"fmt"

	"0chain.net/chaincore/block"
	. "0chain.net/core/logging"
	"0chain.net/core/util"
	"go.uber.org/zap"
)

/* StateSanityCheck - after generating a block or verification of a block,
   this can be called to run some state sanity checks */
func (c *Chain) StateSanityCheck(ctx context.Context, b *block.Block) {
	if bytes.Compare(b.ClientStateHash, b.PrevBlock.ClientStateHash) == 0 {
		return
	}
	if err := c.validateState(ctx, b, b.PrevBlock.ClientState.GetRoot()); err != nil {
		Logger.DPanic("state sanity check - state change validation", zap.Error(err))
	}
	if err := c.validateStateChangesRoot(b); err != nil {
		Logger.DPanic("state sanity check - state changes root validation", zap.Error(err))
	}
}

func (c *Chain) validateStateChangesRoot(b *block.Block) error {
	bsc := block.NewBlockStateChange(b)

	if b.ClientStateHash != nil && (bsc.GetRoot() == nil ||
		bytes.Compare(bsc.GetRoot().GetHashBytes(), b.ClientStateHash) != 0) {
		computedRoot := ""

		if bsc.GetRoot() != nil {
			computedRoot = bsc.GetRoot().GetHash()
		}

		Logger.Error("block state change - root mismatch",
			zap.Int64("round", b.Round),
			zap.String("block", b.Hash),
			zap.String("state_root", util.ToHex(b.ClientStateHash)),
			zap.Any("computed_root", computedRoot))
		return ErrStateMismatch
	}

	return nil
}

func (c *Chain) validateState(ctx context.Context, b *block.Block, priorRoot util.Key) error {
	if len(b.ClientState.GetChangeCollector().GetChanges()) > 0 {
		changes := block.NewBlockStateChange(b)
		stateRoot := changes.GetRoot()
		if stateRoot == nil {
			b.ClientState.PrettyPrint(debug.Output)

			Logger.DPanic("validate state - state root is null", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int("changes", len(changes.Nodes)))
		}
		if bytes.Compare(stateRoot.GetHashBytes(), b.ClientState.GetRoot()) != 0 {
			b.ClientState.GetChangeCollector().PrintChanges(debug.Output)
			b.ClientState.PrettyPrint(debug.Output)

			Logger.DPanic("validate state",
				zap.Int64("round", b.Round),
				zap.String("block", b.Hash),
				zap.Any("state", util.ToHex(b.ClientState.GetRoot())),
				zap.String("computed_state", stateRoot.GetHash()),
				zap.Int("changes", len(changes.Nodes)))
		}
		if priorRoot == nil {
			priorRoot = b.PrevBlock.ClientState.GetRoot()
		}

		err := changes.Validate(ctx)
		if err != nil {
			Logger.Error("validate state - changes validate failure", zap.Error(err))
			pstate := util.CloneMPT(b.ClientState)
			pstate.SetRoot(priorRoot)
			printStates(b.ClientState, pstate)
			return err
		}

		err = b.ClientState.Validate()
		if err != nil {
			Logger.Error("validate state - client state validate failure", zap.Error(err))
			pstate := util.CloneMPT(b.ClientState)
			pstate.SetRoot(priorRoot)
			printStates(b.ClientState, pstate)
			return err
		}
	}

	return nil
}

func printStates(curr util.MerklePatriciaTrieI, prev util.MerklePatriciaTrieI) {
	curr.PrettyPrint(debug.Output)

	fmt.Fprintf(debug.Output, "previous state\n")
	prev.PrettyPrint(debug.Output)
}