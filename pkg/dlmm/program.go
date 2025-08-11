package dlmm

import "github.com/gagliardetto/solana-go"

// Default DLMM program IDs per cluster. Replace with actual IDs as needed.
var (
    // MainnetProgramID should be set to the actual DLMM program ID on mainnet-beta.
    MainnetProgramID = solana.MustPublicKeyFromBase58("LBUZKhRxPF3XUpBCjp4YzTKgLccjZhTSDM9YuVaPwxo")
    // DevnetProgramID should be set to the actual DLMM program ID on devnet if available.
    DevnetProgramID  = solana.PublicKey{}
    // TestnetProgramID should be set to the actual DLMM program ID on testnet if available.
    TestnetProgramID = solana.PublicKey{}
)


