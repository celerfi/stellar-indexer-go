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
	pairTokenCache = make(map[string][2]string)
	cacheMutex     sync.RWMutex
)

func getPairTokens(poolAddr string) (string, string, error) {
	cacheMutex.RLock()
	tokens, ok := pairTokenCache[poolAddr]
	cacheMutex.RUnlock()
	if ok {
		return tokens[0], tokens[1], nil
	}

	t0, t1, err := utils.GetSoroswapPairTokens(poolAddr)
	if err != nil {
		return "", "", err
	}

	cacheMutex.Lock()
	pairTokenCache[poolAddr] = [2]string{t0, t1}
	cacheMutex.Unlock()

	return t0, t1, nil
}

func ProcessSoroswapEvents(tx ingest.LedgerTransaction, seq uint32, blocktime time.Time) {
	events, err := tx.GetContractEvents()
	if err != nil {
		return
	}

	var tx_array []models.TransactionModels
	var liq_actions []models.LiquidityAction

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

		scAddr := xdr.ScAddress{
			Type:       xdr.ScAddressTypeScAddressTypeContract,
			ContractId: event.ContractId,
		}
		pool_addr, _ := scAddr.String()

		token0, token1, err := getPairTokens(pool_addr)
		if err != nil {
			continue
		}

		switch eventName {
		case "swap":
			if len(body.Topics) < 3 {
				continue
			}

			vec, ok := body.Data.GetVec()
			if !ok || vec == nil || len(*vec) < 4 {
				continue
			}

			a0in := utils.Int128ToDecimalFloat((*vec)[0].MustI128(), 7)
			a1in := utils.Int128ToDecimalFloat((*vec)[1].MustI128(), 7)
			a0out := utils.Int128ToDecimalFloat((*vec)[2].MustI128(), 7)
			a1out := utils.Int128ToDecimalFloat((*vec)[3].MustI128(), 7)

			tx_instance := models.TransactionModels{
				BlockTime:       blocktime,
				LedgerSequence:  seq,
				TransactionHash: tx.Result.TransactionHash.HexString(),
				DexName:         utils.DEX_NAME_SOROSWAP,
				Dex_type:        "AMM",
				PoolAddress:     pool_addr,
				Status:          "matched",
			}

			if a0in > 0 {
				tx_instance.TokenIn = token0
				tx_instance.TokenOut = token1
				tx_instance.AmountSold = a0in
				tx_instance.AmountBought = a1out
				if a1out > 0 {
					price := a0in / a1out
					utils.InsertPriceTicks([]models.PriceTick{{
						Timestamp:  blocktime,
						AssetID:    token1,
						SourceID:   utils.DEX_NAME_SOROSWAP,
						SourceType: "amm",
						PriceUSD:   price,
						LedgerSeq:  seq,
						TxHash:     tx_instance.TransactionHash,
					}})
				}
			} else {
				tx_instance.TokenIn = token1
				tx_instance.TokenOut = token0
				tx_instance.AmountSold = a1in
				tx_instance.AmountBought = a0out
				if a0out > 0 {
					price := a1in / a0out
					utils.InsertPriceTicks([]models.PriceTick{{
						Timestamp:  blocktime,
						AssetID:    token0,
						SourceID:   utils.DEX_NAME_SOROSWAP,
						SourceType: "amm",
						PriceUSD:   price,
						LedgerSeq:  seq,
						TxHash:     tx_instance.TransactionHash,
					}})
				}
			}

			tx_array = append(tx_array, tx_instance)

		case "mint":
			vec, ok := body.Data.GetVec()
			if !ok || vec == nil || len(*vec) < 2 {
				continue
			}

			sender_addr, _ := body.Topics[1].GetAddress()
			sender, _ := sender_addr.String()

			liq_actions = append(liq_actions, models.LiquidityAction{
				Timestamp:       blocktime,
				LedgerSequence:  seq,
				TransactionHash: tx.Result.TransactionHash.HexString(),
				PoolAddress:     pool_addr,
				ActionType:      "deposit",
				User:            sender,
				AmountA:         utils.Int128ToDecimalFloat((*vec)[0].MustI128(), 7),
				AmountB:         utils.Int128ToDecimalFloat((*vec)[1].MustI128(), 7),
				TokenA:          token0,
				TokenB:          token1,
			})

		case "burn":
			vec, ok := body.Data.GetVec()
			if !ok || vec == nil || len(*vec) < 2 {
				continue
			}

			sender_addr, _ := body.Topics[1].GetAddress()
			sender, _ := sender_addr.String()

			liq_actions = append(liq_actions, models.LiquidityAction{
				Timestamp:       blocktime,
				LedgerSequence:  seq,
				TransactionHash: tx.Result.TransactionHash.HexString(),
				PoolAddress:     pool_addr,
				ActionType:      "withdraw",
				User:            sender,
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
		fmt.Printf("Inserted %d Soroswap liquidity actions\n", len(liq_actions))
	}
}