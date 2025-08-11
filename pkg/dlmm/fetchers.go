package dlmm

import (
	"context"

	"github.com/gagliardetto/solana-go"
	solanarpc "github.com/gagliardetto/solana-go/rpc"
)

// GetPool fetches and decodes a DLMM pool account.
// TODO: Replace placeholder decode with actual mapping from pkg/generated.
func (c *Client) GetPool(ctx context.Context, poolPubkey solana.PublicKey) (*Pool, error) {
    resp, err := c.rpc.GetAccountInfoWithOpts(ctx, poolPubkey, &solanarpc.GetAccountInfoOpts{
        Commitment: c.commitment,
    })
    if err != nil {
        return nil, err
    }
    if resp == nil || resp.Value == nil {
        return nil, ErrAccountNotFound
    }

    // Placeholder: we'll decode using pkg/generated account struct later.
    pool := &Pool{Address: poolPubkey}
    return pool, nil
}


