package utils

import (
	"fmt"
	"math"
	"math/big"

	"github.com/celerfi/stellar-indexer-go/models"
	"github.com/stellar/go/xdr"
)

func FormatAsset(a xdr.Asset) string {
	switch a.Type {
	case xdr.AssetTypeAssetTypeNative:
		return "XLM"
	case xdr.AssetTypeAssetTypeCreditAlphanum4:
		credit := a.MustAlphaNum4()
		return fmt.Sprintf("%s:%s", credit.AssetCode, credit.Issuer.Address())
	case xdr.AssetTypeAssetTypeCreditAlphanum12:
		credit := a.MustAlphaNum12()
		return fmt.Sprintf("%s:%s", credit.AssetCode, credit.Issuer.Address())
	default:
		return "Unknown"
	}
}

func PrettyPrintTransaction(t models.TransactionModels) {
	fmt.Printf("BlockTime: %v\n", t.BlockTime)
	fmt.Printf("LedgerSequence: %d\n", t.LedgerSequence)
	fmt.Printf("TransactionHash: %s\n", t.TransactionHash)
	fmt.Printf("OperationIndex: %d\n", t.OperationIndex)
	fmt.Printf("DexName: %s\n", t.DexName)
	fmt.Printf("SourceAccount: %s\n", t.SourceAccount)
	fmt.Printf("TokenIn: %s\n", t.TokenIn)
	fmt.Printf("TokenOut: %s\n", t.TokenOut)
	fmt.Printf("OfferID: %d\n", t.OfferID)
	fmt.Printf("MatchedOfferID: %d\n", t.MatchedOfferID)
	fmt.Printf("BuyerAccount: %s\n", t.BuyerAccount)
	fmt.Printf("SellerAccount: %s\n", t.SellerAccount)
	fmt.Printf("OfferBuyAmount: %v\n", t.OfferBuyAmount)
	fmt.Printf("OfferSellAmount: %v\n", t.OfferSellAmount)
	fmt.Printf("AmountBought: %v\n", t.AmountBought)
	fmt.Printf("AmountSold: %v\n", t.AmountSold)
	fmt.Printf("OfferPrice: %v\n", t.OfferPrice)
	fmt.Printf("Status: %s\n", t.Status)

	fmt.Println("OrderMatches:")
	for i, match := range t.OrderMatches {
		fmt.Printf("  Match #%d: %+v\n", i+1, match)
	}
}

func Int128ToDecimalFloat(parts xdr.Int128Parts, decimals int) float64 {
	// 1. Convert Hi and Lo parts to a *big.Int
	hi := big.NewInt(int64(parts.Hi))
	lo := new(big.Int)
	lo.SetUint64(uint64(parts.Lo))
	hi.Lsh(hi, 64) // Shift 'hi' 64 bits to the left
	hi.Add(hi, lo) // Add 'lo' to get the full 128-bit integer

	// 2. Convert the *big.Int to a *big.Float for division
	bigFloatValue := new(big.Float).SetInt(hi)

	// 3. Create the divisor (10^decimals)
	divisor := new(big.Float)
	divisor.SetInt(big.NewInt(int64(math.Pow10(decimals))))

	// 4. Perform high-precision division
	resultFloat := new(big.Float).Quo(bigFloatValue, divisor)

	// 5. Convert the final result to a standard float64
	//    This is where precision might be lost if the number is huge,
	//    but for token amounts, it's generally fine.
	finalVal, _ := resultFloat.Float64()
	return finalVal
}
