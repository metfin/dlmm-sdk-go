package dlmm

import (
	"context"

	"github.com/gagliardetto/solana-go"
	solanarpc "github.com/gagliardetto/solana-go/rpc"

	lb_clmm "github.com/metfin/dlmm-sdk-go/pkg/generated"
)

// GetPool fetches and decodes a DLMM pool account.
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

    data := resp.Value.Data.GetBinary()

    parsed, err := lb_clmm.ParseAccount_LbPair(data)
    if err != nil {
        return nil, err
    }

    pool := &Pool{Address: poolPubkey, LbPair: parsed}
    return pool, nil
}


