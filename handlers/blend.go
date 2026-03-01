package tx_handlers

import (
	"fmt"
	"time"

	"github.com/celerfi/stellar-indexer-go/models"
	"github.com/celerfi/stellar-indexer-go/utils"
	"github.com/stellar/go/ingest"
)

func HandleBlendEvent(tx ingest.LedgerTransaction, seq uint32, blocktime time.Time) {
	events, err := tx.GetContractEvents()
	if err != nil {
		return
	}

	var blendEvents []models.BlendEvent

	for _, event := range events {
		body := event.Body.V0
		if body == nil || len(body.Topics) < 1 {
			continue
		}

		event_symbol, ok := body.Topics[0].GetSym()
		if !ok {
			continue
		}
		eventName := string(event_symbol)

		switch eventName {
		case "deposit", "withdraw", "borrow", "repay":
			if len(body.Topics) < 3 {
				continue
			}
			user_addr, _ := body.Topics[1].GetAddress()
			asset_addr, _ := body.Topics[2].GetAddress()
			
			user, _ := user_addr.String()
			asset, _ := asset_addr.String()
			
			var amount float64
			if i128, ok := body.Data.GetI128(); ok {
				amount = utils.Int128ToDecimalFloat(i128, 7)
			}

			blendEvents = append(blendEvents, models.BlendEvent{
				Timestamp:       blocktime,
				LedgerSequence:  seq,
				TransactionHash: tx.Result.TransactionHash.HexString(),
				ContractID:      fmt.Sprintf("%x", *event.ContractId),
				EventType:       eventName,
				User:            user,
				Asset:           asset,
				Amount:          amount,
			})

		case "liquidate":
			if len(body.Topics) < 3 {
				continue
			}
			liquidator_addr, _ := body.Topics[1].GetAddress()
			borrower_addr, _ := body.Topics[2].GetAddress()
			
			liquidator, _ := liquidator_addr.String()
			borrower, _ := borrower_addr.String()

			// Data: [collateral_asset: Address, debt_asset: Address, amount: i128]
			vec, ok := body.Data.GetVec()
			if !ok || vec == nil || len(*vec) < 3 {
				continue
			}

			collateral_addr, _ := (*vec)[0].GetAddress()
			debt_addr, _ := (*vec)[1].GetAddress()
			amount_i128, _ := (*vec)[2].GetI128()

			collateral, _ := collateral_addr.String()
			debt, _ := debt_addr.String()
			amount := utils.Int128ToDecimalFloat(amount_i128, 7)

			blendEvents = append(blendEvents, models.BlendEvent{
				Timestamp:       blocktime,
				LedgerSequence:  seq,
				TransactionHash: tx.Result.TransactionHash.HexString(),
				ContractID:      fmt.Sprintf("%x", *event.ContractId),
				EventType:       eventName,
				User:            borrower,
				Liquidator:      liquidator,
				CollateralAsset: collateral,
				DebtAsset:       debt,
				Amount:          amount,
			})
		}
	}

	if len(blendEvents) > 0 {
		fmt.Printf("Parsed %d Blend events\n", len(blendEvents))
		utils.InsertBlendEvents(blendEvents)
	}
}
