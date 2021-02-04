// +build state_debug_blck state_debug_txn

package debug

import (
	"fmt"
	"os"
)

const BlockLevel bool = true;

var Output *os.File = nil;

func Init() {
	out, err := os.Create("/0chain/states/debug.log")
	if err != nil {
		panic(err)
	}
	if out == nil {
		panic("Couldn't create file for state debugging")
	}

	Output = out
	fmt.Fprintf(Output, "Starting states log...\n")
}
