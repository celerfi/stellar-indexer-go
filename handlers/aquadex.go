package tx_handlers

import (
	"fmt"
	"sync"
	"time"

	"github.com/celerfi/stellar-indexer-go/models"
	"github.com/celerfi/stellar-indexer-go/utils"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/xdr"
)

var (
	aquariusPairCache = make(map[string][2]string)
	aquaCacheMutex    sync.RWMutex
)

func getAquariusPairTokens(poolAddr string) (string, string, error) {
	aquaCacheMutex.RLock()
	tokens, ok := aquariusPairCache[poolAddr]
	aquaCacheMutex.RUnlock()
	if ok {
		return tokens[0], tokens[1], nil
	}

	t0, t1, err := utils.GetSoroswapPairTokens(poolAddr)
	if err != nil {
		return "", "", err
	}

	aquaCacheMutex.Lock()
	aquariusPairCache[poolAddr] = [2]string{t0, t1}
	aquaCacheMutex.Unlock()

	return t0, t1, nil
}

func ProcessAquariusTransaction(tx ingest.LedgerTransaction, seq uint32, blocktime time.Time) {
	var tx_array []models.TransactionModels
	var liq_actions []models.LiquidityAction

	events, err := tx.GetContractEvents()
	if err != nil {
		return
	}

	for _, event := range events {
		body := event.Body.V0
		if body == nil || len(body.Topics) < 1 {
			continue
		}

		scAddr := xdr.ScAddress{
			Type:       xdr.ScAddressTypeScAddressTypeContract,
			ContractId: event.ContractId,
		}
		pool_addr, _ := scAddr.String()

		event_symbol, ok := body.Topics[0].GetSym()
		if !ok {
			continue
		}
		eventName := string(event_symbol)

		token0, token1, _ := getAquariusPairTokens(pool_addr)

		switch eventName {
		case "trade":
			token_in_sym, _ := body.Topics[1].GetAddress()
			token_out_sym, _ := body.Topics[2].GetAddress()
			token_in, _ := token_in_sym.String()
			token_out, _ := token_out_sym.String()

			tx_instance := models.TransactionModels{
				BlockTime:       blocktime,
				LedgerSequence:  seq,
				TransactionHash: tx.Result.TransactionHash.HexString(),
				DexName:         utils.DEX_NAME_AQUARIUS,
				SourceAccount:   tx.Envelope.SourceAccount().GoString(),
				Dex_type:        "AMM",
				PoolAddress:     pool_addr,
				TokenIn:         token_in,
				TokenOut:        token_out,
				Status:          "matched",
			}

			if vec, ok := body.Data.GetVec(); ok && vec != nil && len(*vec) >= 3 {
				tx_instance.AmountSold = utils.Int128ToDecimalFloat((*vec)[0].MustI128(), 7)
				tx_instance.AmountBought = utils.Int128ToDecimalFloat((*vec)[1].MustI128(), 7)
				tx_instance.DexFee = utils.Int128ToDecimalFloat((*vec)[2].MustI128(), 7)

				if tx_instance.AmountBought > 0 {
					price := tx_instance.AmountSold / tx_instance.AmountBought
					utils.InsertPriceTicks([]models.PriceTick{{
						Timestamp:  blocktime,
						AssetID:    token_out,
						SourceID:   utils.DEX_NAME_AQUARIUS,
						SourceType: "amm",
						PriceUSD:   price,
						LedgerSeq:  seq,
						TxHash:     tx_instance.TransactionHash,
					}})
				}
			}
			tx_array = append(tx_array, tx_instance)

		case "deposit", "withdraw":
			vec, ok := body.Data.GetVec()
			if !ok || vec == nil || len(*vec) < 2 {
				continue
			}

			user_addr, _ := body.Topics[1].GetAddress()
			user, _ := user_addr.String()

			liq_actions = append(liq_actions, models.LiquidityAction{
				Timestamp:       blocktime,
				LedgerSequence:  seq,
				TransactionHash: tx.Result.TransactionHash.HexString(),
				PoolAddress:     pool_addr,
				ActionType:      eventName,
				User:            user,
				AmountA:         utils.Int128ToDecimalFloat((*vec)[0].MustI128(), 7),
				AmountB:         utils.Int128ToDecimalFloat((*vec)[1].MustI128(), 7),
				TokenA:          token0,
				TokenB:          token1,
			})
		}
	}

	if len(tx_array) > 0 {
		utils.InsertTransactionsToDb(tx_array)
	}
	if len(liq_actions) > 0 {
		utils.InsertLiquidityActions(liq_actions)
		fmt.Printf("Inserted %d Aquarius liquidity actions\n", len(liq_actions))
	}
}