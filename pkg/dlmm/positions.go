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

// PositionsResult is a generic result type for position queries with optional active bin info
type PositionsResult struct {
	ActiveBin *BinLiquidity // Optional active bin info
	Positions []LbPosition
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
// Supports filtering by lbPair and/or user public keys
func createPositionFilters(lbPair *solana.PublicKey, user *solana.PublicKey) []solanarpc.RPCFilter {
	filters := []solanarpc.RPCFilter{
		memcmpFilter(0, lb_clmm.Account_PositionV2[:]),
	}

	if lbPair != nil {
		filters = append(filters, memcmpFilter(8, lbPair.Bytes()))
	}

	if user != nil {
		filters = append(filters, memcmpFilter(8+32, user.Bytes()))
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
	keyOnlyOpts := &solanarpc.GetProgramAccountsOpts{
		Commitment: config.commitment,
		Filters:    filters,
	}

	accs, err := c.rpc.GetProgramAccountsWithOpts(ctx, config.programID, keyOnlyOpts)
	if err != nil {
		return nil, err
	}

	// Apply a client-side cap on number of accounts if a limit is provided
	if limit != nil && *limit < uint64(len(accs)) {
		accs = accs[:*limit]
	}

	// If nothing to fetch, return early
	if len(accs) == 0 {
		return []LbPosition{}, nil
	}

	// Phase 2: fetch full accounts for the selected keys
	keys := make([]solana.PublicKey, 0, len(accs))
	for _, a := range accs {
		keys = append(keys, a.Pubkey)
	}
	maOpts := &solanarpc.GetMultipleAccountsOpts{Commitment: config.commitment}
	multi, err := c.rpc.GetMultipleAccountsWithOpts(ctx, keys, maOpts)
	if err != nil {
		return nil, err
	}
	if multi == nil || multi.Value == nil {
		return nil, ErrAccountNotFound
	}

	positions := make([]LbPosition, 0, len(multi.Value))
	for i, acct := range multi.Value {
		if acct == nil {
			continue
		}
		ka := solanarpc.KeyedAccount{Pubkey: keys[i], Account: acct}
		pos, err := parsePositionAccount(ka)
		if err != nil {
			return nil, err
		}
		positions = append(positions, *pos)
	}

	return positions, nil
}

// fetchActiveBin fetches and computes the active bin information for a given lbPair
func (c *Client) fetchActiveBin(ctx context.Context, lbPair solana.PublicKey) (*BinLiquidity, error) {
	lb, err := c.fetchLbPairAccount(ctx, lbPair)
	if err != nil {
		return nil, err
	}

	activeID := lb.ActiveId
	price := getPriceOfBinByBinId(activeID, lb.BinStep)
	active := &BinLiquidity{BinID: activeID, Price: price, PricePerToken: price}
	
	return active, nil
}

// fetchPositionsWithOptionalActiveBin fetches positions and optionally includes active bin info
func (c *Client) fetchPositionsWithOptionalActiveBin(ctx context.Context, filters []solanarpc.RPCFilter, limit *uint64, includeActiveBin bool, lbPair *solana.PublicKey) (*PositionsResult, error) {
	// Fetch positions
	positions, err := c.fetchPositions(ctx, filters, limit)
	if err != nil {
		return nil, err
	}

	result := &PositionsResult{
		Positions: positions,
	}

	// Optionally fetch active bin info if lbPair is provided
	if includeActiveBin && lbPair != nil {
		activeBin, err := c.fetchActiveBin(ctx, *lbPair)
		if err != nil {
			return nil, err
		}
		result.ActiveBin = activeBin
	}

	return result, nil
}

// PositionsQueryOptions standardizes position queries.
type PositionsQueryOptions struct {
    LbPair           *solana.PublicKey
    User             *solana.PublicKey
    IncludeActiveBin bool
    Limit            uint64
}

// QueryPositions executes a standardized position query based on options.
func (c *Client) QueryPositions(ctx context.Context, opts PositionsQueryOptions) (*PositionsResult, error) {
    // default limit
    limit := opts.Limit
    if limit == 0 {
        limit = 25
    }

    filters := createPositionFilters(opts.LbPair, opts.User)
    return c.fetchPositionsWithOptionalActiveBin(ctx, filters, &limit, opts.IncludeActiveBin, opts.LbPair)
}

// GetPositionsByUserAndLbPair fetches active bin info and positions for a user in an lbPair.
// If userPubKey is nil, returns only activeBin with empty positions.
func (c *Client) GetPositionsByUserAndLbPair(ctx context.Context, lbPair solana.PublicKey, userPubKey *solana.PublicKey, includeActiveBin ...bool) (*PositionsByUserAndLbPairResult, error) {
	includeBin := len(includeActiveBin) == 0 || includeActiveBin[0]
	result, err := c.QueryPositions(ctx, PositionsQueryOptions{
		LbPair:           &lbPair,
		User:             userPubKey,
		IncludeActiveBin: includeBin,
		Limit:            25,
	})
	if err != nil {
		return nil, err
	}
	return &PositionsByUserAndLbPairResult{
		ActiveBin:     *result.ActiveBin,
		UserPositions: result.Positions,
	}, nil
}

// GetPositionsByLbPair returns all PositionV2 accounts that belong to the given lbPair.
// Optionally includes active bin information.
func (c *Client) GetPositionsByLbPair(ctx context.Context, lbPair solana.PublicKey, includeActiveBin ...bool) (*PositionsResult, error) {
	includeBin := len(includeActiveBin) == 0 || includeActiveBin[0]
	return c.QueryPositions(ctx, PositionsQueryOptions{
		LbPair:           &lbPair,
		IncludeActiveBin: includeBin,
		Limit:            25,
	})
}

// GetPositionsByUser returns all PositionV2 accounts that belong to the given user across all pools.
// Note: Active bin information is not available when searching across all pools.
func (c *Client) GetPositionsByUser(ctx context.Context, userPubKey solana.PublicKey) (*PositionsResult, error) {
	return c.QueryPositions(ctx, PositionsQueryOptions{
		User:  &userPubKey,
		Limit: 50,
	})
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

// PositionBinArraysAndActive encapsulates bin array range and active bin for a position
type PositionBinArraysAndActive struct {
    ActiveBin   BinLiquidity
    BinArrays   map[int64]*lb_clmm.BinArray
    LowerIndex  int64
    UpperIndex  int64
    LowerBinId  int32
    UpperBinId  int32
    LbPair      solana.PublicKey
}

// GetPositionBinArraysAndActiveBin returns the bin arrays covering a position's bin range
// along with the active bin info for the position's pool (lbPair).
func (c *Client) GetPositionBinArraysAndActiveBin(ctx context.Context, positionAddr solana.PublicKey) (*PositionBinArraysAndActive, error) {
    // Load position account
    resp, err := c.rpc.GetAccountInfoWithOpts(ctx, positionAddr, &solanarpc.GetAccountInfoOpts{Commitment: c.commitment})
    if err != nil {
        return nil, err
    }
    if resp == nil || resp.Value == nil {
        return nil, ErrAccountNotFound
    }

    posData := resp.Value.Data.GetBinary()
    pos, err := lb_clmm.ParseAccount_PositionV2(posData)
    if err != nil {
        return nil, fmt.Errorf("decode position: %w", err)
    }

    // Compute bin array index range for the position's bins
    lowerIdx := binIdToBinArrayIndex(pos.LowerBinId)
    upperIdx := binIdToBinArrayIndex(pos.UpperBinId)

    // Fetch bin arrays
    arrays, err := c.fetchBinArrays(ctx, pos.LbPair, lowerIdx, upperIdx)
    if err != nil {
        return nil, err
    }

    // Fetch active bin for the lbPair
    active, err := c.fetchActiveBin(ctx, pos.LbPair)
    if err != nil {
        return nil, err
    }

    return &PositionBinArraysAndActive{
        ActiveBin:  *active,
        BinArrays:  arrays,
        LowerIndex: lowerIdx,
        UpperIndex: upperIdx,
        LowerBinId: pos.LowerBinId,
        UpperBinId: pos.UpperBinId,
        LbPair:     pos.LbPair,
    }, nil
}
