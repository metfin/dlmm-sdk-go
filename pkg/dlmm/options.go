package dlmm

import (
    "log"

    "github.com/gagliardetto/solana-go"
    solanarpc "github.com/gagliardetto/solana-go/rpc"
)

// Option configures a Client.
type Option func(*Client)

// WithProgramID sets the DLMM program ID.
func WithProgramID(programID solana.PublicKey) Option {
    return func(c *Client) { c.programID = programID }
}

// WithCommitment sets the default RPC commitment.
func WithCommitment(commitment solanarpc.CommitmentType) Option {
    return func(c *Client) { c.commitment = commitment }
}

// WithLogger sets a custom logger.
func WithLogger(logger *log.Logger) Option {
    return func(c *Client) { c.logger = logger }
}


