package main

import (
	"context"
	"log"

	"github.com/gagliardetto/solana-go"
	solanarpc "github.com/gagliardetto/solana-go/rpc"

	"github.com/metfin/dlmm-sdk-go/pkg/dlmm"
)

func main() {
    ctx := context.Background()

    rpc := solanarpc.New("https://api.mainnet-beta.solana.com")

    // Initialize DLMM client (thin wrapper around generated code)
    client := dlmm.NewClient(rpc, dlmm.WithProgramID(solana.MustPublicKeyFromBase58("LBUZKhRxPF3XUpBCjp4YzTKgLccjZhTSDM9YuVaPwxo")))

	poolPubkey := solana.MustPublicKeyFromBase58("HwEfZWKzMGqa6qgKqWLtsDXbx4bgexw5mQSBQj34asWt")
	pool, err := client.GetPool(ctx, poolPubkey)
	if err != nil {
		log.Fatalf("get pool: %v", err)
	}
	log.Printf("pool: %+v", pool)
}