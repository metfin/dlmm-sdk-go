package dlmm

import "github.com/gagliardetto/solana-go"

// Pool is a user-facing decoded representation of a DLMM pool account.
// This is intentionally minimal for the scaffold; it can be expanded to
// mirror the TS SDK types once `pkg/generated` is available.
type Pool struct {
    Address solana.PublicKey
    // Add fields mapped from the generated account layout.
}


