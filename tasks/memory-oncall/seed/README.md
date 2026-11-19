# ledger-sync

Go job that applies settled entries from the settlement queue to the ledger.
It runs every 15 minutes and exits non-zero with an `E-xxxx` error code when a
batch cannot be applied.

Open incidents are filed as notes in `inbox/`.
