// +build !state_debug_blck
// +build !state_debug_txn

package debug

import (
	"os"
)

const BlockLevel bool = false;

var Output *os.File = nil;

func Init() {
	// must be inlined automatically
}