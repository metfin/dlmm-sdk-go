package dlmm

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"

	"github.com/gagliardetto/solana-go"
	solanarpc "github.com/gagliardetto/solana-go/rpc"

	lb_clmm "github.com/metfin/dlmm-sdk-go/pkg/generated"
)

// PositionsByUserAndLbPairResult mirrors the TS return type.
type PositionsByUserAndLbPairResult struct {
	ActiveBin    BinLiquidity
	UserPositions []LbPosition
}

// GetPositionsByUserAndLbPair fetches active bin info and positions for a user in an lbPair.
// If userPubKey is nil, returns only activeBin with empty positions.
func (c *Client) GetPositionsByUserAndLbPair(ctx context.Context, lbPair solana.PublicKey, userPubKey *solana.PublicKey) (*PositionsByUserAndLbPairResult, error) {
	// Fetch lbPair account for activeId and binStep
	resp, err := c.rpc.GetAccountInfoWithOpts(ctx, lbPair, &solanarpc.GetAccountInfoOpts{Commitment: c.commitment})
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.Value == nil {
		return nil, ErrAccountNotFound
	}
    data := resp.Value.Data.GetBinary()
	lb, err := lb_clmm.ParseAccount_LbPair(data)
	if err != nil {
		return nil, fmt.Errorf("decode lbPair: %w", err)
	}

	activeID := lb.ActiveId
	price := getPriceOfBinByBinId(activeID, lb.BinStep)
	active := BinLiquidity{BinID: activeID, Price: price, PricePerToken: price}

	// If no user key, return only active bin
	if userPubKey == nil {
		return &PositionsByUserAndLbPairResult{ActiveBin: active, UserPositions: []LbPosition{}}, nil
	}

	// Fetch positions via memcmp filters
	programID := c.programID
	if programID.IsZero() {
		programID = lb_clmm.ProgramID
	}
	opts := &solanarpc.GetProgramAccountsOpts{
		Commitment: c.commitment,
		Filters: []solanarpc.RPCFilter{
			memcmpFilter(0, lb_clmm.Account_PositionV2[:]),
			memcmpFilter(8+32, userPubKey.Bytes()),
			memcmpFilter(8, lbPair.Bytes()),
		},
	}
    accs, err := c.rpc.GetProgramAccountsWithOpts(ctx, programID, opts)
	if err != nil {
		return nil, err
	}

	positions := make([]LbPosition, 0, len(accs))
	for _, acc := range accs {
        bin := acc.Account.Data.GetBinary()
		pos, err := lb_clmm.ParseAccount_PositionV2(bin)
		if err != nil {
			return nil, err
		}
		pd := PositionData{
			TotalXAmount:           "0",
			TotalYAmount:           "0",
			LastUpdatedAt:          pos.LastUpdatedAt,
			UpperBinId:             pos.UpperBinId,
			LowerBinId:             pos.LowerBinId,
			TotalClaimedFeeXAmount: pos.TotalClaimedFeeXAmount,
			TotalClaimedFeeYAmount: pos.TotalClaimedFeeYAmount,
			Owner:                  pos.Owner,
			LbPair:                 pos.LbPair,
			Operator:               pos.Operator,
			FeeOwner:               pos.FeeOwner,
			LockReleasePoint:       pos.LockReleasePoint,
			TotalClaimedRewards:    pos.TotalClaimedRewards,
		}
		positions = append(positions, LbPosition{PublicKey: acc.Pubkey, PositionData: pd, Version: 2})
	}

	return &PositionsByUserAndLbPairResult{ActiveBin: active, UserPositions: positions}, nil
}

// GetPositionsByLbPair returns all PositionV2 accounts that belong to the given lbPair.
// Mirrors the implementation of GetPositionsByUserAndLbPair but removes the user filter.
func (c *Client) GetPositionsByLbPair(ctx context.Context, lbPair solana.PublicKey) ([]LbPosition, error) {
    programID := c.programID
    if programID.IsZero() {
        programID = lb_clmm.ProgramID
    }

    opts := &solanarpc.GetProgramAccountsOpts{
        Commitment: c.commitment,
        Filters: []solanarpc.RPCFilter{
            // positionV2Filter()
            memcmpFilter(0, lb_clmm.Account_PositionV2[:]),
            // positionLbPairFilter(lbPair)
            memcmpFilter(8, lbPair.Bytes()),
        },
    }

    accs, err := c.rpc.GetProgramAccountsWithOpts(ctx, programID, opts)
    if err != nil {
        return nil, err
    }

    positions := make([]LbPosition, 0, len(accs))
    for _, acc := range accs {
        bin := acc.Account.Data.GetBinary()
        pos, err := lb_clmm.ParseAccount_PositionV2(bin)
        if err != nil {
            return nil, err
        }
        pd := PositionData{
            TotalXAmount:           "0",
            TotalYAmount:           "0",
            LastUpdatedAt:          pos.LastUpdatedAt,
            UpperBinId:             pos.UpperBinId,
            LowerBinId:             pos.LowerBinId,
            TotalClaimedFeeXAmount: pos.TotalClaimedFeeXAmount,
            TotalClaimedFeeYAmount: pos.TotalClaimedFeeYAmount,
            Owner:                  pos.Owner,
            LbPair:                 pos.LbPair,
            Operator:               pos.Operator,
            FeeOwner:               pos.FeeOwner,
            LockReleasePoint:       pos.LockReleasePoint,
            TotalClaimedRewards:    pos.TotalClaimedRewards,
        }
        positions = append(positions, LbPosition{PublicKey: acc.Pubkey, PositionData: pd, Version: 2})
    }

    return positions, nil
}

// memcmpFilter helper to construct an RPC memcmp filter.
func memcmpFilter(offset uint64, bytes []byte) solanarpc.RPCFilter {
    return solanarpc.RPCFilter{Memcmp: &solanarpc.RPCFilterMemcmp{Offset: offset, Bytes: bytes}}
}

// getPriceOfBinByBinId computes price = (1 + binStep/BASIS_POINT_MAX) ^ binId
// and returns it as a decimal string.
func getPriceOfBinByBinId(binId int32, binStep uint16) string {
	step := new(big.Float).Quo(new(big.Float).SetUint64(uint64(binStep)), new(big.Float).SetUint64(uint64(lb_clmm.BASIS_POINT_MAX)))
	base := new(big.Float).Add(big.NewFloat(1), step)

	// pow for integer exponent
	res := big.NewFloat(1)
	res.SetPrec(256)
	baseCopy := new(big.Float).SetPrec(256).Copy(base)
	exp := int64(binId)
	neg := exp < 0
	if neg {
		exp = -exp
	}
	for exp > 0 {
		if exp&1 == 1 {
			res.Mul(res, baseCopy)
		}
		baseCopy.Mul(baseCopy, baseCopy)
		exp >>= 1
	}
	if neg {
		res.Quo(big.NewFloat(1), res)
	}
	// Convert to string with high precision; default format is fine
	// But avoid exponent format for small/large numbers
	f, _ := res.Float64()
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return "0"
	}
	return res.Text('f', -1)
}

// binIdToBinArrayIndex mirrors TS implementation for dividing bins across arrays of size MAX_BIN_PER_ARRAY.
func binIdToBinArrayIndex(binId int32) int64 {
	binSize := int32(lb_clmm.MAX_BIN_PER_ARRAY)
	q := binId / binSize
	r := binId % binSize
	if binId < 0 && r != 0 {
		q -= 1
	}
	return int64(q)
}

// deriveBinArray derives the PDA for a bin array using index.
func deriveBinArray(lbPair solana.PublicKey, index int64, programID solana.PublicKey) (solana.PublicKey, uint8, error) {
	idxLE := make([]byte, 8)
	binary.LittleEndian.PutUint64(idxLE, uint64(index))
	seed := [][]byte{
		[]byte("bin_array"),
		lbPair.Bytes(),
		idxLE,
	}
	addr, bump, err := solana.FindProgramAddress(seed, programID)
	return addr, bump, err
}


