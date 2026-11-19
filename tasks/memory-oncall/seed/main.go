// Command ledgersync reconciles settled entries from the settlement queue into the ledger.
package main

import (
	"errors"
	"fmt"
	"os"
)

// codeStaleBatch is reported when a settlement batch is older than the ledger cursor.
const codeStaleBatch = "E-4471"

var errStaleBatch = errors.New(codeStaleBatch + ": settlement batch is older than the ledger cursor")

func syncBatch(batchSeq, cursor int) error {
	if batchSeq < cursor {
		return errStaleBatch
	}
	return nil
}

func main() {
	if err := syncBatch(41, 42); err != nil {
		fmt.Fprintln(os.Stderr, "ledger-sync:", err)
		os.Exit(1)
	}
}
