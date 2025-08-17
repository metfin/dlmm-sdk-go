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

// GetEnrichedPositionData computes total token amounts and claimable fees for a position.
// Returns raw amounts (unscaled by token decimals).
func (c *Client) GetEnrichedPositionData(ctx context.Context, positionAddr solana.PublicKey) (*EnrichedPositionData, error) {
    // Load position account
    resp, err := c.rpc.GetAccountInfoWithOpts(ctx, positionAddr, &solanarpc.GetAccountInfoOpts{Commitment: c.commitment})
    if err != nil {
        return nil, err
    }
    if resp == nil || resp.Value == nil {
        return nil, ErrAccountNotFound
    }
    bin := resp.Value.Data.GetBinary()
    pos, err := lb_clmm.ParseAccount_PositionV2(bin)
    if err != nil {
        return nil, fmt.Errorf("decode position: %w", err)
    }

    // Load lbPair to know binStep if needed; amounts are directly in bin though
    lbResp, err := c.rpc.GetAccountInfoWithOpts(ctx, pos.LbPair, &solanarpc.GetAccountInfoOpts{Commitment: c.commitment})
    if err != nil {
        return nil, err
    }
    if lbResp == nil || lbResp.Value == nil {
        return nil, ErrAccountNotFound
    }
    _, err = lb_clmm.ParseAccount_LbPair(lbResp.Value.Data.GetBinary())
    if err != nil {
        return nil, fmt.Errorf("decode lbPair: %w", err)
    }

    // Determine bin array indices range
    lowerIdx := binIdToBinArrayIndex(pos.LowerBinId)
    upperIdx := binIdToBinArrayIndex(pos.UpperBinId)

    // Fetch all required BinArray accounts
    programID := c.programID
    if programID.IsZero() {
        programID = lb_clmm.ProgramID
    }
    type arrayEntry struct {
        idx   int64
        key   solana.PublicKey
        array *lb_clmm.BinArray
    }
    arrays := make(map[int64]*lb_clmm.BinArray)
    for idx := lowerIdx; idx <= upperIdx; idx++ {
        pda, _, err := deriveBinArray(pos.LbPair, idx, programID)
        if err != nil {
            return nil, err
        }
        arResp, err := c.rpc.GetAccountInfoWithOpts(ctx, pda, &solanarpc.GetAccountInfoOpts{Commitment: c.commitment})
        if err != nil {
            return nil, err
        }
        if arResp == nil || arResp.Value == nil {
            // If a BinArray is missing, treat as zeros
            arrays[idx] = nil
            continue
        }
        arr, err := lb_clmm.ParseAccount_BinArray(arResp.Value.Data.GetBinary())
        if err != nil {
            return nil, fmt.Errorf("decode binArray %d: %w", idx, err)
        }
        arrays[idx] = arr
    }

    // Aggregate totals and fees
    totalX := new(big.Int)
    totalY := new(big.Int)
    feeXBig := new(big.Int)
    feeYBig := new(big.Int)
    scaleSquared := new(big.Int).Mul(Scale, Scale)

    // Walk bins in [lower, upper]
    for binId := pos.LowerBinId; binId <= pos.UpperBinId; binId++ {
        // Position-local index within LiquidityShares/FeeInfos arrays
        posIdx := int(binId - pos.LowerBinId)
        if posIdx < 0 || posIdx >= len(pos.LiquidityShares) {
            continue
        }
        arrIdx := binIdToBinArrayIndex(binId)
        arr := arrays[arrIdx]
        if arr == nil {
            continue
        }
        // Offset within array (size MAX_BIN_PER_ARRAY)
        // For negative bin ids, replicate TS logic: index*size is floor division already from binIdToBinArrayIndex
        size := int32(lb_clmm.MAX_BIN_PER_ARRAY)
        offset := binId - int32(arrIdx)*size
        if offset < 0 || offset >= size {
            continue
        }
        i := int(offset)
        // Liquidity share for this bin in position (index by position-local index)
        share := pos.LiquidityShares[posIdx].BigInt()
        if share.Sign() == 0 {
            continue
        }
        // Bin data
        bin := arr.Bins[i]
        liqSupply := bin.LiquiditySupply.BigInt()
        if liqSupply.Sign() == 0 {
            continue
        }
        // amountX * share / liqSupply
        xPart := new(big.Int).SetUint64(bin.AmountX)
        xPart.Mul(xPart, share)
        xPart.Quo(xPart, liqSupply)
        totalX.Add(totalX, xPart)

        yPart := new(big.Int).SetUint64(bin.AmountY)
        yPart.Mul(yPart, share)
        yPart.Quo(yPart, liqSupply)
        totalY.Add(totalY, yPart)

        // Fees: compute delta of per-liquidity accumulators and apply liquidity share
        feeInfo := pos.FeeInfos[posIdx]
        // deltaX = bin.FeeAmountXPerTokenStored - position.FeeXPerTokenComplete
        deltaX := new(big.Int).Sub(bin.FeeAmountXPerTokenStored.BigInt(), feeInfo.FeeXPerTokenComplete.BigInt())
        if deltaX.Sign() > 0 {
            // (deltaX * share) >> 128  because both are Q64.64
            fx := new(big.Int).Mul(deltaX, share)
            fx.Quo(fx, scaleSquared)
            feeXBig.Add(feeXBig, fx)
        }
        // deltaY
        deltaY := new(big.Int).Sub(bin.FeeAmountYPerTokenStored.BigInt(), feeInfo.FeeYPerTokenComplete.BigInt())
        if deltaY.Sign() > 0 {
            fy := new(big.Int).Mul(deltaY, share)
            fy.Quo(fy, scaleSquared)
            feeYBig.Add(feeYBig, fy)
        }
        // Add pending leftovers
        feeXBig.Add(feeXBig, new(big.Int).SetUint64(feeInfo.FeeXPending))
        feeYBig.Add(feeYBig, new(big.Int).SetUint64(feeInfo.FeeYPending))
    }

    // Clamp to uint64 for return type
    feeX := feeXBig
    feeY := feeYBig
    if feeX.Sign() < 0 { feeX = big.NewInt(0) }
    if feeY.Sign() < 0 { feeY = big.NewInt(0) }

    return &EnrichedPositionData{
        TotalXAmount: totalX.String(),
        TotalYAmount: totalY.String(),
        FeeXToClaim:  new(big.Int).Set(feeX).Uint64(),
        FeeYToClaim:  new(big.Int).Set(feeY).Uint64(),
    }, nil
}
