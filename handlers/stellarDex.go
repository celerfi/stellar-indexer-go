package tx_handlers

import (
	"strings"
	"time"

	"github.com/celerfi/stellar-indexer-go/models"
	"github.com/celerfi/stellar-indexer-go/utils"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/xdr"
)

func HandleManageBuyTransaction(
	tx ingest.LedgerTransaction,
	op xdr.Operation,
	seq uint32,
	opIndex int,
	results *[]xdr.OperationResult,
	blockTime time.Time,
) {
	offer := op.Body.MustManageBuyOfferOp()
	if results == nil || opIndex >= len(*results) {
		return
	}

	result := (*results)[opIndex].Tr.ManageBuyOfferResult
	if result == nil {
		return
	}

	if result.Code == xdr.ManageBuyOfferResultCodeManageBuyOfferSuccess {
		success := result.Success
		numMatches := len(success.OffersClaimed)

		// Basic offer info
		clean_tx := models.TransactionModels{
			BlockTime:       blockTime,
			LedgerSequence:  seq,
			TransactionHash: tx.Result.TransactionHash.HexString(),
			OperationIndex:  opIndex,
			DexName:         utils.DEX_NAME_STELLAR_DEX,
			SourceAccount:   op.SourceAccount.Address(),
			TokenIn:         utils.FormatAsset(offer.Buying),
			TokenOut:        utils.FormatAsset(offer.Selling),
			OfferBuyAmount:  float64(offer.BuyAmount) / 1e7,
			OfferSellAmount: float64(offer.BuyAmount) / 1e7 * float64(offer.Price.N) / float64(offer.Price.D),
			OfferPrice:      float64(offer.Price.N) / float64(offer.Price.D),
		}

		// Determine status
		if numMatches == 0 {
			clean_tx.Status = utils.ORDERBOOK_TX_STATUS_POSTED
		} else if success.Offer.Offer == nil || success.Offer.Offer.OfferId == 0 {
			clean_tx.Status = utils.ORDERBOOK_TX_STATUS_MATCHED
		} else {
			clean_tx.Status = utils.ORDERBOOK_TX_STATUS_PARTIALLY_MATCHED
			clean_tx.OfferID = uint64(success.Offer.Offer.OfferId)
		}

		// Parse each matched offer
		for _, claim := range success.OffersClaimed {
			match := models.OrderMatch{
				OrderType:    "counter_offer",
				AmountBought: float64(claim.AmountBought()) / 1e7,
				AmountSold:   float64(claim.AmountSold()) / 1e7,
				AssetBought:  utils.FormatAsset(claim.AssetBought()),
				AssetSold:    utils.FormatAsset(claim.AssetSold()),
				Owner:        claim.SellerId().Address(),
				OfferID:      uint64(claim.OfferId()),
			}
			clean_tx.OrderMatches = append(clean_tx.OrderMatches, match)
		}
		// if clean_tx.Status == utils.ORDERBOOK_TX_STATUS_MATCHED || clean_tx.Status == utils.ORDERBOOK_TX_STATUS_PARTIALLY_MATCHED {
		// 	fmt.Printf("Tx hash: %v ||||| total number of matches: %v |||| status : %v\n", clean_tx.TransactionHash, numMatches, clean_tx.Status)
		// 	utils.PrettyPrintTransaction(clean_tx)
		// }
		token_buying_split := strings.Split(utils.FormatAsset(offer.Buying), ":")
		token_selling_split := strings.Split(utils.FormatAsset(offer.Buying), ":")
		if len(token_buying_split) > 1 {
			go AddTokenData(token_buying_split[1])
		}
		if len(token_selling_split) > 1 {
			go AddTokenData(token_selling_split[1])
		}
		utils.InsertTransactionsToDb([]models.TransactionModels{clean_tx})
	}
}

func HandleManageSellTransaction(
	tx ingest.LedgerTransaction,
	op xdr.Operation,
	seq uint32,
	opIndex int,
	results *[]xdr.OperationResult,
	blockTime time.Time,
) {
	offer := op.Body.MustManageSellOfferOp()
	if results == nil || opIndex >= len(*results) {
		return
	}

	result := (*results)[opIndex].Tr.ManageSellOfferResult
	if result == nil {
		return
	}

	if result.Code == xdr.ManageSellOfferResultCodeManageSellOfferSuccess {
		success := result.Success
		numMatches := len(success.OffersClaimed)

		// Build the transaction model
		clean_tx := models.TransactionModels{
			BlockTime:       blockTime,
			LedgerSequence:  seq,
			TransactionHash: tx.Result.TransactionHash.HexString(),
			OperationIndex:  opIndex,
			DexName:         utils.DEX_NAME_STELLAR_DEX,
			SourceAccount:   op.SourceAccount.Address(),
			TokenIn:         utils.FormatAsset(offer.Buying),
			TokenOut:        utils.FormatAsset(offer.Selling),
			OfferSellAmount: float64(offer.Amount) / 1e7,
			OfferBuyAmount:  (float64(offer.Amount) / 1e7) * (float64(offer.Price.N) / float64(offer.Price.D)),
			OfferPrice:      float64(offer.Price.N) / float64(offer.Price.D),
		}

		// Determine status
		if numMatches == 0 {
			clean_tx.Status = utils.ORDERBOOK_TX_STATUS_POSTED
		} else if success.Offer.Offer == nil || success.Offer.Offer.OfferId == 0 {
			clean_tx.Status = utils.ORDERBOOK_TX_STATUS_MATCHED
		} else {
			clean_tx.Status = utils.ORDERBOOK_TX_STATUS_PARTIALLY_MATCHED
			clean_tx.OfferID = uint64(success.Offer.Offer.OfferId)
		}

		// Parse matched offers
		for _, claim := range success.OffersClaimed {
			match := models.OrderMatch{
				OrderType:    "counter_offer",
				AmountBought: float64(claim.AmountBought()) / 1e7,
				AmountSold:   float64(claim.AmountSold()) / 1e7,
				AssetBought:  utils.FormatAsset(claim.AssetBought()),
				AssetSold:    utils.FormatAsset(claim.AssetSold()),
				Owner:        claim.SellerId().Address(),
				OfferID:      uint64(claim.OfferId()),
			}
			clean_tx.OrderMatches = append(clean_tx.OrderMatches, match)
		}

		// Print or persist
		// if clean_tx.Status == utils.ORDERBOOK_TX_STATUS_MATCHED || clean_tx.Status == utils.ORDERBOOK_TX_STATUS_PARTIALLY_MATCHED {
		// 	fmt.Printf("Tx hash: %v ||||| total matches: %v |||| status: %v\n", clean_tx.TransactionHash, numMatches, clean_tx.Status)
		// 	utils.PrettyPrintTransaction(clean_tx)
		// }
		token_buying_split := strings.Split(utils.FormatAsset(offer.Buying), ":")
		token_selling_split := strings.Split(utils.FormatAsset(offer.Buying), ":")
		if len(token_buying_split) > 1 {
			go AddTokenData(token_buying_split[1])
		}
		if len(token_selling_split) > 1 {
			go AddTokenData(token_selling_split[1])
		}
		utils.InsertTransactionsToDb([]models.TransactionModels{clean_tx})
	}
}
