// +build !state_debug_blck
// +build !state_debug_txn

package chain

import (
	"0chain.net/chaincore/block"
	"0chain.net/core/util"
	"context"
)

/* StateSanityCheck - after generating a block or verification of a block,
   this can be called to run some state sanity checks */
func (c *Chain) StateSanityCheck(ctx context.Context, b *block.Block) {
	// must be inlined automatically
}

func (c *Chain) validateState(ctx context.Context, b *block.Block, priorRoot util.Key) error {
	// must be inlined automatically
	return nil
}

func printStates(curr util.MerklePatriciaTrieI, prev util.MerklePatriciaTrieI) {
	// must be inlined automatically
}