package main

import (
	"context"
	"log"
	"os"

	"github.com/gagliardetto/solana-go"
	solanarpc "github.com/gagliardetto/solana-go/rpc"

	"github.com/metfin/dlmm-sdk-go/pkg/dlmm"
)

func main() {
	ctx := context.Background()

    rpcUrl := os.Getenv("RPC_URL")
	rpc := solanarpc.New(rpcUrl)

	client := dlmm.NewClient(
		rpc,
		dlmm.WithProgramID(solana.MustPublicKeyFromBase58("LBUZKhRxPF3XUpBCjp4YzTKgLccjZhTSDM9YuVaPwxo")),
	)

    lbPairStr := os.Getenv("LBPAIR")
    userStr := os.Getenv("POSITION_OWNER")

	if lbPairStr == "" {
		log.Println("LBPAIR env var not set; please export a valid lbPair public key. Exiting.")
		return
	}

	lbPair := solana.MustPublicKeyFromBase58(lbPairStr)

	// Example 1: fetch only active bin (no user)
	resNoUser, err := client.GetPositionsByUserAndLbPair(ctx, lbPair, nil)
	if err != nil {
		log.Fatalf("get positions (no user): %v", err)
	}
	log.Printf("Active bin (no user): ID=%d Price=%s", resNoUser.ActiveBin.BinID, resNoUser.ActiveBin.Price)

	// Example 2: fetch positions for a specific user if provided
	if userStr != "" {
		user := solana.MustPublicKeyFromBase58(userStr)
		resWithUser, err := client.GetPositionsByUserAndLbPair(ctx, lbPair, &user)
		if err != nil {
			log.Fatalf("get positions (with user): %v", err)
		}
		log.Printf("Active bin (with user): ID=%d Price=%s", resWithUser.ActiveBin.BinID, resWithUser.ActiveBin.Price)
		log.Printf("Found %d position(s) for user", len(resWithUser.UserPositions))
        for i, p := range resWithUser.UserPositions {
            log.Printf("%d) position=%s lower=%d upper=%d claimedX=%d claimedY=%d", i+1, p.PublicKey.String(), p.PositionData.LowerBinId, p.PositionData.UpperBinId, p.PositionData.TotalClaimedFeeXAmount, p.PositionData.TotalClaimedFeeYAmount)
        }
	}

	// Example 3: fetch positions for all users in a lbPair
	resAllUsers, err := client.GetPositionsByLbPair(ctx, lbPair)
	if err != nil {
		log.Fatalf("get positions (all users): %v", err)
	}
	log.Printf("Found %d position(s) for all users", len(resAllUsers.Positions))

	// Example 3b: fetch positions for all users in a lbPair with active bin info
	resAllUsersWithBin, err := client.GetPositionsByLbPair(ctx, lbPair, true)
	if err != nil {
		log.Fatalf("get positions (all users with bin): %v", err)
	}
	log.Printf("Found %d position(s) for all users with active bin ID=%d Price=%s", 
		len(resAllUsersWithBin.Positions), resAllUsersWithBin.ActiveBin.BinID, resAllUsersWithBin.ActiveBin.Price)

	// Example 4: fetch all positions for a specific user across all pools
	if userStr != "" {
		user := solana.MustPublicKeyFromBase58(userStr)
		resUserAllPools, err := client.GetPositionsByUser(ctx, user)
		if err != nil {
			log.Fatalf("get positions (user across all pools): %v", err)
		}
		log.Printf("Found %d position(s) for user across all pools", len(resUserAllPools.Positions))
		for i, p := range resUserAllPools.Positions {
			log.Printf("%d) position=%s pool=%s lower=%d upper=%d claimedX=%d claimedY=%d", 
				i+1, p.PublicKey.String(), p.PositionData.LbPair.String(), 
				p.PositionData.LowerBinId, p.PositionData.UpperBinId, 
				p.PositionData.TotalClaimedFeeXAmount, p.PositionData.TotalClaimedFeeYAmount)
		}
	}

	// Example 5: fetch positions for a specific user in a specific pool with optional active bin
	if userStr != "" {
		user := solana.MustPublicKeyFromBase58(userStr)
		resUserInPool, err := client.GetPositionsByUserInPool(ctx, lbPair, user, true)
		if err != nil {
			log.Fatalf("get positions (user in pool with bin): %v", err)
		}
		log.Printf("Found %d position(s) for user in pool with active bin ID=%d Price=%s", 
			len(resUserInPool.Positions), resUserInPool.ActiveBin.BinID, resUserInPool.ActiveBin.Price)
	}

	enrichedPositionData, err := client.GetEnrichedPositionData(ctx, solana.MustPublicKeyFromBase58("7zk9d39mHLHwBmNvoNbeZxwhZn1pWYGL1i7ZNvvco7JE"))
	if err != nil {
		log.Fatalf("get enriched position data: %v", err)
	}
	log.Printf("Enriched position data: %+v", enrichedPositionData)
}


