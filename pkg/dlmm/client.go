package dlmm

import (
	"log"

	"github.com/gagliardetto/solana-go"
	solanarpc "github.com/gagliardetto/solana-go/rpc"
)

// Client is the high-level entrypoint for interacting with the DLMM program.
// It wraps an RPC client and provides convenience methods built on top of
// generated instruction builders and account types.
type Client struct {
    rpc        *solanarpc.Client
    programID  solana.PublicKey
    commitment solanarpc.CommitmentType
    logger     *log.Logger
}

// NewClient creates a new DLMM client. Customize via functional options.
func NewClient(rpc *solanarpc.Client, opts ...Option) *Client {
    c := &Client{
        rpc:        rpc,
        commitment: solanarpc.CommitmentProcessed,
        logger:     log.Default(),
    }
    for _, opt := range opts {
        opt(c)
    }
    return c
}

// ProgramID returns the configured DLMM program ID.
func (c *Client) ProgramID() solana.PublicKey { return c.programID }

// Commitment returns the configured commitment level for RPC queries.
func (c *Client) Commitment() solanarpc.CommitmentType { return c.commitment }

// Logger returns the logger used by the client.
func (c *Client) Logger() *log.Logger { return c.logger }

