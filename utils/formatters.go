package utils

import (
	"fmt"

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
	fmt.Printf("MarketSelling: %s\n", t.MarketSelling)
	fmt.Printf("MarketBuying: %s\n", t.MarketBuying)
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
