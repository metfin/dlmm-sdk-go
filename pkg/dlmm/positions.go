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
	ActiveBin     BinLiquidity
	UserPositions []LbPosition
}

// positionQueryConfig holds common configuration for position queries
type positionQueryConfig struct {
	programID  solana.PublicKey
	commitment solanarpc.CommitmentType
}

// getQueryConfig returns the position query configuration with proper program ID fallback
func (c *Client) getQueryConfig() positionQueryConfig {
	programID := c.programID
	if programID.IsZero() {
		programID = lb_clmm.ProgramID
	}
	return positionQueryConfig{
		programID:  programID,
		commitment: c.commitment,
	}
}

// fetchLbPairAccount fetches and parses an LbPair account
func (c *Client) fetchLbPairAccount(ctx context.Context, lbPair solana.PublicKey) (*lb_clmm.LbPair, error) {
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
	return lb, nil
}

// createPositionFilters creates memcmp filters for position queries
func createPositionFilters(lbPair solana.PublicKey, userPubKey *solana.PublicKey) []solanarpc.RPCFilter {
	filters := []solanarpc.RPCFilter{
		memcmpFilter(0, lb_clmm.Account_PositionV2[:]),
		memcmpFilter(8, lbPair.Bytes()),
	}

	if userPubKey != nil {
		filters = append(filters, memcmpFilter(8+32, userPubKey.Bytes()))
	}

	return filters
}

// parsePositionAccount parses a position account and converts it to LbPosition
func parsePositionAccount(acc solanarpc.KeyedAccount) (*LbPosition, error) {
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

	return &LbPosition{PublicKey: acc.Pubkey, PositionData: pd, Version: 2}, nil
}

// fetchPositions fetches positions using the given filters
func (c *Client) fetchPositions(ctx context.Context, filters []solanarpc.RPCFilter, limit *uint64) ([]LbPosition, error) {
	config := c.getQueryConfig()
	opts := &solanarpc.GetProgramAccountsOpts{
		Commitment: config.commitment,
		Filters:    filters,
	}

	// Add limit if specified to reduce RPC usage
	if limit != nil {
		opts.DataSlice = &solanarpc.DataSlice{
			Length: limit,
		}
	}

	accs, err := c.rpc.GetProgramAccountsWithOpts(ctx, config.programID, opts)
	if err != nil {
		return nil, err
	}

	positions := make([]LbPosition, 0, len(accs))
	for _, acc := range accs {
		pos, err := parsePositionAccount(*acc)
		if err != nil {
			return nil, err
		}
		positions = append(positions, *pos)
	}

	return positions, nil
}

// GetPositionsByUserAndLbPair fetches active bin info and positions for a user in an lbPair.
// If userPubKey is nil, returns only activeBin with empty positions.
func (c *Client) GetPositionsByUserAndLbPair(ctx context.Context, lbPair solana.PublicKey, userPubKey *solana.PublicKey) (*PositionsByUserAndLbPairResult, error) {
	// Fetch lbPair account for activeId and binStep
	lb, err := c.fetchLbPairAccount(ctx, lbPair)
	if err != nil {
		return nil, err
	}

	activeID := lb.ActiveId
	price := getPriceOfBinByBinId(activeID, lb.BinStep)
	active := BinLiquidity{BinID: activeID, Price: price, PricePerToken: price}

	// If no user key, return only active bin
	if userPubKey == nil {
		return &PositionsByUserAndLbPairResult{ActiveBin: active, UserPositions: []LbPosition{}}, nil
	}

	// Fetch positions via memcmp filters with limit to reduce RPC usage
	filters := createPositionFilters(lbPair, userPubKey)
	limit := uint64(25) // Limit to 25 accounts to reduce RPC usage
	positions, err := c.fetchPositions(ctx, filters, &limit)
	if err != nil {
		return nil, err
	}

	return &PositionsByUserAndLbPairResult{ActiveBin: active, UserPositions: positions}, nil
}

// GetPositionsByLbPair returns all PositionV2 accounts that belong to the given lbPair.
// Mirrors the implementation of GetPositionsByUserAndLbPair but removes the user filter.
func (c *Client) GetPositionsByLbPair(ctx context.Context, lbPair solana.PublicKey) ([]LbPosition, error) {
	filters := createPositionFilters(lbPair, nil)
	limit := uint64(25) // Limit to 25 accounts to reduce RPC usage
	return c.fetchPositions(ctx, filters, &limit)
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

// fetchBinArrays fetches multiple bin array accounts
func (c *Client) fetchBinArrays(ctx context.Context, lbPair solana.PublicKey, lowerIdx, upperIdx int64) (map[int64]*lb_clmm.BinArray, error) {
	config := c.getQueryConfig()
	arrays := make(map[int64]*lb_clmm.BinArray)

	for idx := lowerIdx; idx <= upperIdx; idx++ {
		pda, _, err := deriveBinArray(lbPair, idx, config.programID)
		if err != nil {
			return nil, err
		}

		arResp, err := c.rpc.GetAccountInfoWithOpts(ctx, pda, &solanarpc.GetAccountInfoOpts{Commitment: config.commitment})
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

	return arrays, nil
}

// calculateBinAmounts calculates token amounts for a specific bin
func calculateBinAmounts(bin *lb_clmm.Bin, share *big.Int, liqSupply *big.Int) (*big.Int, *big.Int) {
	if share.Sign() == 0 || liqSupply.Sign() == 0 {
		return big.NewInt(0), big.NewInt(0)
	}

	// amountX * share / liqSupply
	xPart := new(big.Int).SetUint64(bin.AmountX)
	xPart.Mul(xPart, share)
	xPart.Quo(xPart, liqSupply)

	yPart := new(big.Int).SetUint64(bin.AmountY)
	yPart.Mul(yPart, share)
	yPart.Quo(yPart, liqSupply)

	return xPart, yPart
}

// calculateBinFees calculates claimable fees for a specific bin
func calculateBinFees(bin *lb_clmm.Bin, feeInfo *lb_clmm.FeeInfo, share *big.Int, scaleSquared *big.Int) (*big.Int, *big.Int) {
	feeXBig := big.NewInt(0)
	feeYBig := big.NewInt(0)

	// Fees: compute delta of per-liquidity accumulators and apply liquidity share
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

	return feeXBig, feeYBig
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
	_, err = c.fetchLbPairAccount(ctx, pos.LbPair)
	if err != nil {
		return nil, err
	}

	// Determine bin array indices range
	lowerIdx := binIdToBinArrayIndex(pos.LowerBinId)
	upperIdx := binIdToBinArrayIndex(pos.UpperBinId)

	// Fetch all required BinArray accounts
	arrays, err := c.fetchBinArrays(ctx, pos.LbPair, lowerIdx, upperIdx)
	if err != nil {
		return nil, err
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

		// Calculate amounts and fees for this bin
		xPart, yPart := calculateBinAmounts(&bin, share, liqSupply)
		totalX.Add(totalX, xPart)
		totalY.Add(totalY, yPart)

		feeInfo := pos.FeeInfos[posIdx]
		fx, fy := calculateBinFees(&bin, &feeInfo, share, scaleSquared)
		feeXBig.Add(feeXBig, fx)
		feeYBig.Add(feeYBig, fy)
	}

	// Clamp to uint64 for return type
	feeX := feeXBig
	feeY := feeYBig
	if feeX.Sign() < 0 {
		feeX = big.NewInt(0)
	}
	if feeY.Sign() < 0 {
		feeY = big.NewInt(0)
	}

	return &EnrichedPositionData{
		TotalXAmount: totalX.String(),
		TotalYAmount: totalY.String(),
		FeeXToClaim:  new(big.Int).Set(feeX).Uint64(),
		FeeYToClaim:  new(big.Int).Set(feeY).Uint64(),

		Owner:            pos.Owner,
		Operator:         pos.Operator,
		FeeOwner:         pos.FeeOwner,
		LbPair:           pos.LbPair,
		LockReleasePoint: pos.LockReleasePoint,
		LastUpdatedAt:    pos.LastUpdatedAt,

		LowerBinId: pos.LowerBinId,
		UpperBinId: pos.UpperBinId,
	}, nil
}
