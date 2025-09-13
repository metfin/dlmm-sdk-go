package dlmm

// import (
// 	"context"
// 	"log"
// 	"math"
// 	"math/big"

// 	"github.com/gagliardetto/solana-go"
// 	"github.com/gagliardetto/solana-go/"

// 	lb_clmm "github.com/metfin/dlmm-sdk-go/pkg/generated"
// )

// type StrategyParameters struct {
// 	maxBinId     int32
// 	minBinId     int32
// 	strategyType StrategyType
// 	singleSidedX *bool
// }
// type StrategyType int

// const (
// 	Spot StrategyType = iota
// 	Curve
// 	BidAsk
// )
// func getAssociatedTokenAddressSync(mint solana.PublicKey , owner solana.PublicKey , allowOwnerOffCurve bool) (solana.PublicKey , error ) {
// 	if (!allowOwnerOffCurve && !solana.PublicKey.IsOnCurve()){
// 		return
// 	}
// }
// func (c *Client) createBinArraysIfNeeded(binaArrayIndex []int64, funder solana.PublicKey, lbpair solana.PublicKey) ([]solana.Instruction, error) {
// 	var ixs []solana.Instruction
// 	for _, idx := range binaArrayIndex {
// 		key, _, _ := deriveBinArray(lbpair, idx, MainnetProgramID)
// 		binArrayAccount, err := c.rpc.GetAccountInfo(context.TODO(), key)
// 		if err != nil {
// 			return nil, err
// 		}
// 		if binArrayAccount == nil {
// 			ix, err := lb_clmm.NewInitializeBinArrayInstruction(
// 				idx,
// 				lbpair,
// 				key,
// 				funder,
// 				SystemProgramAccount,
// 			)
// 			if err != nil {
// 				return nil, err
// 			}
// 			ixs = append(ixs, ix)
// 		}
// 	}
// 	return ixs, nil
// }

// func getBinArrayIndexesCoverage(lowerBinId, upperBinId int32) []int64 {
// 	lowerBinArrayIndex := binIdToBinArrayIndex(lowerBinId)
// 	upperBinArrayIndex := binIdToBinArrayIndex(upperBinId)

// 	binArrayIndexes := []int64{}

// 	for i := lowerBinArrayIndex; i <= upperBinArrayIndex; i++ {
// 		binArrayIndexes = append(binArrayIndexes, i)
// 	}

// 	return binArrayIndexes
// }

// func getBinArrayAccountMetasCoverage(lowerBinId, upperBinId int32, lbpair solana.PublicKey, programID solana.PublicKey) ([]solana.AccountMeta, error) {
// 	binArrayKeyscoverage, err := getBinArrayKeyCoverage(lowerBinId, upperBinId, lbpair, programID)
// 	if err != nil {
// 		return nil, err
// 	}
// 	accountMetaData := make([]*solana.AccountMeta, 0, len(binArrayKeyscoverage))
// 	for _, key := range binArrayKeyscoverage {
// 		accountMetaData = append(accountMetaData, &solana.AccountMeta{
// 			PublicKey:  key,
// 			IsWritable: true,
// 			IsSigner:   false,
// 		})

// 	}
// }

// func getBinArrayKeyCoverage(lowerBinId, upperBinId int32, lbpair solana.PublicKey, programID solana.PublicKey) ([]solana.PublicKey, error) {
// 	binArrayIndexCoverage := getBinArrayIndexesCoverage(lowerBinId, upperBinId)

// 	results := make([]solana.PublicKey, 0, len(binArrayIndexCoverage))

// 	for _, index := range binArrayIndexCoverage {
// 		derived, _, err := deriveBinArray(lbpair, index, programID)
// 		if err != nil {
// 			return results, err
// 		}

// 		results = append(results, derived)
// 	}
// 	return results, nil
// }

// func (c *Client) getorCreateATAInstruction(tokenMint solana.PublicKey , owner solana.PublicKey) {
// 	toAccount :=
// }
// func (c *Client) initializePositionAndAddLiquidityByStrategy(ctx context.Context, lbPairPubKey solana.PublicKey, positionPubKey solana.PublicKey, totalXamount big.Int, totalYamount big.Int, strategy StrategyParameters, user solana.PublicKey, slippage int) (*[]solana.Transaction, error) {
// 	maxBinId := strategy.maxBinId
// 	minBinId := strategy.minBinId
// 	lbpair, err := lb_clmm.ParseAccount_LbPair(lbPairPubKey[:])
// 	if err != nil {
// 		log.Print("error parsing lbpair account %s", lbPairPubKey)
// 		return nil, err
// 	}
// 	width := maxBinId - maxBinId + 1
// 	var maxActiveBinSlippage int
// 	if slippage != 0 {
// 		maxActiveBinSlippage = int(math.Ceil(
// 			float64(slippage) / (float64(lbpair.BinStep) / 100.0),
// 		))
// 	} else {
// 		maxActiveBinSlippage = dlmm.MaxActiveBinSlippage
// 	}
// 	var preInstructions []solana.Instruction

// 	initializePositionIx, err := lb_clmm.NewInitializePositionInstruction(
// 		minBinId,
// 		width,
// 		user,
// 		positionPubKey,
// 		lbPairPubKey,
// 		user,
// 		SystemProgramAccount,
// 		user,
// 		EventAuthorityAccount,
// 		MainnetProgramID,
// 	)
// 	if err != nil {
// 		log.Printf("error initializing position")
// 		return nil, err
// 	}
// 	preInstructions = append(preInstructions, initializePositionIx)

// 	binArrayIndexes := getBinArrayIndexesCoverage(minBinId, maxBinId)

// 	binArrayAccountMetaData, err := getBinArrayAccountMetasCoverage(minBinId, maxBinId, lbPairPubKey, MainnetProgramID)
// 	if err != nil {
// 		log.Printf("error getting bin array metadata")
// 		return nil, err
// 	}
// 	createBinarrayIxs , err := c.createBinArraysIfNeeded(binArrayIndexes , user ,lbPairPubKey)
// 	preInstructions = append(preInstructions, createBinarrayIxs...)

// }
