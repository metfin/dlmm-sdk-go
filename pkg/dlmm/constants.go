package dlmm

import (
	"math/big"

	"github.com/gagliardetto/solana-go"
)

// Network represents a Solana cluster name used by the DLMM SDK.
type Network string

const (
	NetworkMainnet Network = "mainnet-beta"
	NetworkTestnet Network = "testnet"
	NetworkDevnet  Network = "devnet"
	NetworkLocal   Network = "localhost"
)

var (
	SystemProgramAccount      = solana.MustPublicKeyFromBase58("11111111111111111111111111111111")
	EventAuthorityAccount     = solana.MustPublicKeyFromBase58("D1ZN9Wj1fRSUQfCjhvnu1hqDMT7hzjzBBpi12nVniYD6")
	AssosicatedTokenProgramID = solana.MustPublicKeyFromBase58("ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJA8knL")
	TokenProgramID            = solana.MustPublicKeyFromBase58("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA")
)

// ProgramIDs maps network to the DLMM program ID.
var ProgramIDs = map[Network]solana.PublicKey{
	NetworkDevnet:  solana.MustPublicKeyFromBase58("LBUZKhRxPF3XUpBCjp4YzTKgLccjZhTSDM9YuVaPwxo"),
	NetworkLocal:   solana.MustPublicKeyFromBase58("LbVRzDTvBDEcrthxfZ4RL6yiq3uZw8bS6MwtdY6UhFQ"),
	NetworkMainnet: solana.MustPublicKeyFromBase58("LBUZKhRxPF3XUpBCjp4YzTKgLccjZhTSDM9YuVaPwxo"),
}

// AdminByNetwork maps network to the admin authority.
var AdminByNetwork = map[Network]solana.PublicKey{
	NetworkDevnet: solana.MustPublicKeyFromBase58("6WaLrrRfReGKBYUSkmx2K6AuT21ida4j8at2SUiZdXu8"),
	NetworkLocal:  solana.MustPublicKeyFromBase58("bossj3JvwiNK7pvjr149DqdtJxf2gdygbcmEPTkb2F1"),
}

// Scalar and fee-related constants.
const (
	BasisPointMax = 10000
	ScaleOffset   = 64

	FeePrecision uint64 = 1_000_000_000 // 1e9
	MaxFeeRate   uint64 = 100_000_000   // 1e8
)

// Scale and precision as big integers (2^64); cannot fit in uint64.
var (
	Scale     = new(big.Int).Lsh(big.NewInt(1), 64)
	Precision = new(big.Int).Lsh(big.NewInt(1), 64)
)

// Lamport-denominated rent-exempt fees (pre-computed from SOL values in TS SDK).
const (
	BinArrayFeeLamports       uint64 = 71_437_440
	PositionFeeLamports       uint64 = 57_406_080
	TokenAccountFeeLamports   uint64 = 2_039_280
	PoolFeeLamports           uint64 = 7_182_720
	BinArrayBitmapFeeLamports uint64 = 11_804_160
)

// Constants mirrored from IDL constants used by the TS SDK helpers.
// Note: these duplicate values present in pkg/generated but are provided here
// for convenience within the user-facing dlmm package.
const (
	MaxBinArraySize             uint64 = 70   // MAX_BIN_PER_ARRAY
	DefaultBinPerPosition       uint64 = 70   // DEFAULT_BIN_PER_POSITION
	BinArrayBitmapSize          int32  = 512  // BIN_ARRAY_BITMAP_SIZE
	ExtensionBinArrayBitmapSize uint64 = 12   // EXTENSION_BINARRAY_BITMAP_SIZE
	PositionMaxLength           uint64 = 1400 // POSITION_MAX_LENGTH
	MaxResizeLength             uint64 = 70   // MAX_RESIZE_LENGTH
	MaxBinsPerPosition          uint64 = 1400 // Equal to POSITION_MAX_LENGTH
)

// Misc values from the TS SDK.
var (
	SimulationUser = solana.MustPublicKeyFromBase58("HrY9qR5TiB2xPzzvbBu5KrBorMfYGQXh9osXydz4jy9s")
	IlmBase        = solana.MustPublicKeyFromBase58("MFGQxwAmB91SwuYX36okv2Qmdc9aMuHTwWGUrp4AtB1")
)

const (
	MaxClaimAllAllowed         = 2
	MaxBinLengthAllowedInOneTx = 26
	MaxActiveBinSlippage       = 3
	MaxExtraBinArrays          = 3
)

// Max unsigned 64-bit integer as big.Int
var U64Max = func() *big.Int {
	v, ok := new(big.Int).SetString("18446744073709551615", 10)
	if !ok {
		panic("invalid U64_MAX")
	}
	return v
}()
