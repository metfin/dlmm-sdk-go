package dlmm

import (
	"github.com/gagliardetto/solana-go"

	lb_clmm "github.com/metfin/dlmm-sdk-go/pkg/generated"
)

// Pool is a user-facing decoded representation of a DLMM pool account.
// It wraps the generated `LbPair` account for convenience and forward-compat.
type Pool struct {
	Address solana.PublicKey
	LbPair  *lb_clmm.LbPair
}

// BinLiquidity mirrors the important fields returned by the TS SDK helper.
type BinLiquidity struct {
	BinID         int32
	Price         string
	PricePerToken string
}

// PositionData is a condensed representation of a processed position.
// It intentionally mirrors the TS PositionData shape for downstream use.
type PositionData struct {
	TotalXAmount           string
	TotalYAmount           string
	LastUpdatedAt          int64
	UpperBinId             int32
	LowerBinId             int32
	TotalClaimedFeeXAmount uint64
	TotalClaimedFeeYAmount uint64
	Owner                  solana.PublicKey
	LbPair                 solana.PublicKey
	Operator               solana.PublicKey
	FeeOwner               solana.PublicKey
	LockReleasePoint       uint64
	TotalClaimedRewards    [2]uint64
}

// LbPosition wraps a position address with its processed data and a version tag.
type LbPosition struct {
	PublicKey    solana.PublicKey
	PositionData PositionData
	Version      int // 2 for PositionV2
}

// EnrichedPositionData is a computed summary for a single position address.
// Values are raw token units (not adjusted for decimals).
type EnrichedPositionData struct {
	TotalXAmount string
	TotalYAmount string
	FeeXToClaim  uint64
	FeeYToClaim  uint64

	Owner            solana.PublicKey
	Operator         solana.PublicKey
	FeeOwner         solana.PublicKey
	LbPair           solana.PublicKey
	LockReleasePoint uint64
	LastUpdatedAt    int64

	LowerBinId int32
	UpperBinId int32
}
