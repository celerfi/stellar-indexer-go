package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/celerfi/stellar-indexer-go/config"
	"github.com/celerfi/stellar-indexer-go/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var db = connectToDb()

func connectToDb() *pgxpool.Pool {
	databaseUrl := fmt.Sprintf("postgres://%s:%s@%s:5432/%s", config.DB_USER, config.DB_PASSWORD, config.DB_HOST, config.DB_NAME)
	ctx := context.Background()

	poolConfig, err := pgxpool.ParseConfig(databaseUrl)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to parse database config: %v\n", err)
		os.Exit(1)
	}

	poolConfig.MaxConns = 5
	dbPool, err := pgxpool.NewWithConfig(ctx, poolConfig)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create connection pool: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Successfully connected to database!")
	return dbPool
}

func InsertTransactionsToDb(transactions []models.TransactionModels) {
	if len(transactions) == 0 {
		return
	}

	tx, err := db.Begin(context.Background())
	if err != nil {
		fmt.Printf("Error starting transaction: %v\n", err)
		return
	}
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Recovered from panic in InsertTransactionsToDb, rolling back: %v\n", r)
			tx.Rollback(context.Background())
		}
	}()

	err = func() error {
		_, err = tx.CopyFrom(
			context.Background(),
			pgx.Identifier{"transaction_models"},
			[]string{
				"block_time", "ledger_sequence", "transaction_hash", "operation_index",
				"dex_name", "source_account", "token_in", "token_out", "offer_id",
				"dex_type", "pool_address", "matched_offer_id", "buyer_account",
				"seller_account", "offer_buy_amount", "offer_sell_amount", "amount_bought",
				"amount_sold", "offer_price", "dex_fee", "status", "order_matches",
			},
			pgx.CopyFromSlice(len(transactions), func(i int) ([]interface{}, error) {
				transaction := transactions[i]
				orderMatchesJSON, err := json.Marshal(transaction.OrderMatches)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal order matches to JSON: %w", err)
				}

				return []interface{}{
					transaction.BlockTime, transaction.LedgerSequence, transaction.TransactionHash, transaction.OperationIndex,
					transaction.DexName, transaction.SourceAccount, transaction.TokenIn, transaction.TokenOut, transaction.OfferID,
					transaction.Dex_type, transaction.PoolAddress, transaction.MatchedOfferID, transaction.BuyerAccount,
					transaction.SellerAccount, transaction.OfferBuyAmount, transaction.OfferSellAmount, transaction.AmountBought,
					transaction.AmountSold, transaction.OfferPrice, transaction.DexFee, transaction.Status, orderMatchesJSON,
				}, nil
			}),
		)
		return err
	}()

	if err != nil {
		fmt.Printf("Error inserting transactions, rolling back: %v\n", err)
		tx.Rollback(context.Background())
		return
	}

	err = tx.Commit(context.Background())
	if err != nil {
		fmt.Printf("Error committing transaction: %v\n", err)
	}
}

func TokenExistsInDb(tokenHash string) bool {
	var exists bool
	err := db.QueryRow(context.Background(), "SELECT EXISTS(SELECT 1 FROM token_info WHERE contract_address = $1)", tokenHash).Scan(&exists)
	if err != nil {
		fmt.Printf("Error checking if token exists: %v\n", err)
		return false
	}
	return exists
}

func SaveTokenToDB(token models.TokenInfo) {
	supplyBreakdownJSON, err := json.Marshal(token.SupplyBreakdown)
	if err != nil {
		fmt.Printf("Error marshaling SupplyBreakdown to JSON: %v\n", err)
		return
	}

	_, err = db.Exec(
		context.Background(),
		`INSERT INTO token_info (
			contract_address, symbol, name, decimals, total_supply,
			admin_address, is_auth_revocable, is_mintable, is_sac,
			num_accounts, supply_breakdown
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (contract_address) DO UPDATE SET
			symbol = EXCLUDED.symbol,
			name = EXCLUDED.name,
			decimals = EXCLUDED.decimals,
			total_supply = EXCLUDED.total_supply,
			admin_address = EXCLUDED.admin_address,
			is_auth_revocable = EXCLUDED.is_auth_revocable,
			is_mintable = EXCLUDED.is_mintable,
			is_sac = EXCLUDED.is_sac,
			num_accounts = EXCLUDED.num_accounts,
			supply_breakdown = EXCLUDED.supply_breakdown`,
		token.ContractAddress, token.Symbol, token.Name, token.Decimals, token.TotalSupply,
		token.AdminAddress, token.IsAuthRevocable, token.IsMintable, token.IsSAC,
		token.NumAccounts, supplyBreakdownJSON,
	)
	if err != nil {
		fmt.Printf("Error saving token to database: %v\n", err)
	}
}

func getLastSuccessFullLedgerInDb() (uint32, error) {
	var lastLedger uint32
	row := db.QueryRow(context.Background(), "SELECT MAX(ledger_sequence) FROM transaction_models")
	err := row.Scan(&lastLedger)
	if err == pgx.ErrNoRows || lastLedger == 0 {
		return 0, nil
	} else if err != nil {
		return 0, fmt.Errorf("error getting last successful ledger: %w", err)
	}
	return lastLedger, nil
}

func PoolExistsInDb(poolAddress string) bool {
	var exists bool
	err := db.QueryRow(context.Background(), "SELECT EXISTS(SELECT 1 FROM liquidity_pools WHERE pool_address = $1)", poolAddress).Scan(&exists)
	if err != nil {
		fmt.Printf("Error checking if pool exists: %v\n", err)
		return false
	}
	return exists
}

func SavePoolToDB(pool models.LiquidityPool) {
	_, err := db.Exec(
		context.Background(),
		`INSERT INTO liquidity_pools (
			pool_address, token_a, token_b, fee_bps, type, created_at
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (pool_address) DO UPDATE SET
			token_a = EXCLUDED.token_a,
			token_b = EXCLUDED.token_b,
			fee_bps = EXCLUDED.fee_bps,
			type = EXCLUDED.type,
			created_at = EXCLUDED.created_at`,
		pool.PoolAddress, pool.TokenA, pool.TokenB, pool.FeeBps, pool.Type, pool.CreatedAt,
	)
	if err != nil {
		fmt.Printf("Error saving pool to database: %v\n", err)
	}
}

func InsertPriceTicks(ticks []models.PriceTick) {
	if len(ticks) == 0 {
		return
	}

	tx, err := db.Begin(context.Background())
	if err != nil {
		fmt.Printf("Error starting transaction: %v\n", err)
		return
	}
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Recovered from panic in InsertPriceTicks, rolling back: %v\n", r)
			tx.Rollback(context.Background())
		}
	}()

	err = func() error {
		_, err = tx.CopyFrom(
			context.Background(),
			pgx.Identifier{"price_ticks"},
			[]string{
				"ts", "asset_id", "source_id", "source_type",
				"price_usd", "volume_usd", "base_volume", "quote_volume",
				"ledger_seq", "tx_hash",
			},
			pgx.CopyFromSlice(len(ticks), func(i int) ([]interface{}, error) {
				t := ticks[i]
				return []interface{}{
					t.Timestamp, t.AssetID, t.SourceID, t.SourceType,
					t.PriceUSD, t.VolumeUSD, t.BaseVolume, t.QuoteVolume,
					t.LedgerSeq, t.TxHash,
				}, nil
			}),
		)
		return err
	}()

	if err != nil {
		fmt.Printf("Error inserting price ticks, rolling back: %v\n", err)
		tx.Rollback(context.Background())
		return
	}

	if err = tx.Commit(context.Background()); err != nil {
		fmt.Printf("Error committing price ticks: %v\n", err)
	}
}

func InsertBlendEvents(events []models.BlendEvent) {
	if len(events) == 0 {
		return
	}

	tx, err := db.Begin(context.Background())
	if err != nil {
		fmt.Printf("Error starting transaction: %v\n", err)
		return
	}
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Recovered from panic in InsertBlendEvents, rolling back: %v\n", r)
			tx.Rollback(context.Background())
		}
	}()

	err = func() error {
		_, err = tx.CopyFrom(
			context.Background(),
			pgx.Identifier{"blend_events"},
			[]string{
				"ts", "ledger_seq", "tx_hash", "contract_id",
				"event_type", "user_address", "asset_id", "amount",
				"liquidator_address", "collateral_asset", "debt_asset",
			},
			pgx.CopyFromSlice(len(events), func(i int) ([]interface{}, error) {
				e := events[i]
				return []interface{}{
					e.Timestamp, e.LedgerSequence, e.TransactionHash, e.ContractID,
					e.EventType, e.User, e.Asset, e.Amount,
					e.Liquidator, e.CollateralAsset, e.DebtAsset,
				}, nil
			}),
		)
		return err
	}()

	if err != nil {
		fmt.Printf("Error inserting blend events, rolling back: %v\n", err)
		tx.Rollback(context.Background())
		return
	}

	if err = tx.Commit(context.Background()); err != nil {
		fmt.Printf("Error committing blend events: %v\n", err)
	}
}

func InsertLiquidityActions(actions []models.LiquidityAction) {
	if len(actions) == 0 {
		return
	}

	tx, err := db.Begin(context.Background())
	if err != nil {
		fmt.Printf("Error starting transaction: %v\n", err)
		return
	}
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Recovered from panic in InsertLiquidityActions, rolling back: %v\n", r)
			tx.Rollback(context.Background())
		}
	}()

	err = func() error {
		_, err = tx.CopyFrom(
			context.Background(),
			pgx.Identifier{"liquidity_actions"},
			[]string{
				"ts", "ledger_seq", "tx_hash", "pool_address",
				"action_type", "user_address", "amount_a", "amount_b",
				"token_a", "token_b",
			},
			pgx.CopyFromSlice(len(actions), func(i int) ([]interface{}, error) {
				a := actions[i]
				return []interface{}{
					a.Timestamp, a.LedgerSequence, a.TransactionHash, a.PoolAddress,
					a.ActionType, a.User, a.AmountA, a.AmountB,
					a.TokenA, a.TokenB,
				}, nil
			}),
		)
		return err
	}()

	if err != nil {
		fmt.Printf("Error inserting liquidity actions, rolling back: %v\n", err)
		tx.Rollback(context.Background())
		return
	}

	if err = tx.Commit(context.Background()); err != nil {
		fmt.Printf("Error committing liquidity actions: %v\n", err)
	}
}

func InsertTransfers(transfers []models.Transfer) {
	if len(transfers) == 0 {
		return
	}

	tx, err := db.Begin(context.Background())
	if err != nil {
		fmt.Printf("Error starting transaction: %v\n", err)
		return
	}
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Recovered from panic in InsertTransfers, rolling back: %v\n", r)
			tx.Rollback(context.Background())
		}
	}()

	err = func() error {
		_, err = tx.CopyFrom(
			context.Background(),
			pgx.Identifier{"transfers"},
			[]string{
				"ts", "ledger_seq", "tx_hash", "operation_index",
				"from_address", "to_address", "asset_id", "amount",
			},
			pgx.CopyFromSlice(len(transfers), func(i int) ([]interface{}, error) {
				t := transfers[i]
				return []interface{}{
					t.Timestamp, t.LedgerSequence, t.TransactionHash, t.OperationIndex,
					t.From, t.To, t.Asset, t.Amount,
				}, nil
			}),
		)
		return err
	}()

	if err != nil {
		fmt.Printf("Error inserting transfers, rolling back: %v\n", err)
		tx.Rollback(context.Background())
		return
	}

	if err = tx.Commit(context.Background()); err != nil {
		fmt.Printf("Error committing transfers: %v\n", err)
	}
}

func RefreshAnalytics() error {
	_, err := db.Exec(context.Background(), "SELECT refresh_analytics_views()")
	return err
}

func GetTransactionByHash(hash string) (*models.TransactionModels, error) {
	var tx models.TransactionModels
	var orderMatchesJSON []byte
	err := db.QueryRow(context.Background(),
		"SELECT block_time, ledger_sequence, transaction_hash, operation_index, dex_name, token_in, token_out, amount_bought, amount_sold, status, order_matches FROM transaction_models WHERE transaction_hash = $1",
		hash).Scan(&tx.BlockTime, &tx.LedgerSequence, &tx.TransactionHash, &tx.OperationIndex, &tx.DexName, &tx.TokenIn, &tx.TokenOut, &tx.AmountBought, &tx.AmountSold, &tx.Status, &orderMatchesJSON)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(orderMatchesJSON, &tx.OrderMatches)
	return &tx, nil
}

func GetTransactions(limit, offset int) ([]models.TransactionModels, error) {
	rows, err := db.Query(context.Background(),
		"SELECT block_time, ledger_sequence, transaction_hash, operation_index, dex_name, token_in, token_out, amount_bought, amount_sold, status FROM transaction_models ORDER BY block_time DESC LIMIT $1 OFFSET $2",
		limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []models.TransactionModels
	for rows.Next() {
		var tx models.TransactionModels
		if err := rows.Scan(&tx.BlockTime, &tx.LedgerSequence, &tx.TransactionHash, &tx.OperationIndex, &tx.DexName, &tx.TokenIn, &tx.TokenOut, &tx.AmountBought, &tx.AmountSold, &tx.Status); err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}
	return txs, nil
}

func GetTokenByAddress(address string) (*models.TokenInfo, error) {
	var t models.TokenInfo
	var supplyBreakdownJSON []byte
	err := db.QueryRow(context.Background(),
		"SELECT contract_address, symbol, name, decimals, total_supply, is_sac, supply_breakdown FROM token_info WHERE contract_address = $1",
		address).Scan(&t.ContractAddress, &t.Symbol, &t.Name, &t.Decimals, &t.TotalSupply, &t.IsSAC, &supplyBreakdownJSON)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(supplyBreakdownJSON, &t.SupplyBreakdown)
	return &t, nil
}

func GetTokens(limit, offset int) ([]models.TokenInfo, error) {
	rows, err := db.Query(context.Background(),
		"SELECT contract_address, symbol, name, decimals, total_supply, is_sac FROM token_info LIMIT $1 OFFSET $2",
		limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []models.TokenInfo
	for rows.Next() {
		var t models.TokenInfo
		if err := rows.Scan(&t.ContractAddress, &t.Symbol, &t.Name, &t.Decimals, &t.TotalSupply, &t.IsSAC); err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	return tokens, nil
}

func GetPoolByAddress(address string) (map[string]interface{}, error) {
	var p struct {
		PoolAddress string
		TokenA      string
		TokenB      string
		FeeBps      int32
		Type        string
		TvlUsd      float64
		Volume24h   float64
	}
	err := db.QueryRow(context.Background(), `
		SELECT p.pool_address, p.token_a, p.token_b, p.fee_bps, p.type, 
		       COALESCE(t.tvl_usd, 0) as tvl_usd, 
		       COALESCE(v.volume_token_in, 0) as volume_24h
		FROM liquidity_pools p
		LEFT JOIN pool_tvl t ON p.pool_address = t.pool_address
		LEFT JOIN pool_volume_24h v ON p.pool_address = v.pool_address
		WHERE p.pool_address = $1`,
		address).Scan(&p.PoolAddress, &p.TokenA, &p.TokenB, &p.FeeBps, &p.Type, &p.TvlUsd, &p.Volume24h)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"poolAddress": p.PoolAddress,
		"tokenA":      p.TokenA,
		"tokenB":      p.TokenB,
		"feeBps":      p.FeeBps,
		"type":        p.Type,
		"tvlUsd":      p.TvlUsd,
		"volume24h":   p.Volume24h,
	}, nil
}

func GetPools(limit, offset int) ([]map[string]interface{}, error) {
	rows, err := db.Query(context.Background(), `
		SELECT p.pool_address, p.token_a, p.token_b, p.fee_bps, p.type, 
		       COALESCE(t.tvl_usd, 0) as tvl_usd, 
		       COALESCE(v.volume_token_in, 0) as volume_24h
		FROM liquidity_pools p
		LEFT JOIN pool_tvl t ON p.pool_address = t.pool_address
		LEFT JOIN pool_volume_24h v ON p.pool_address = v.pool_address
		LIMIT $1 OFFSET $2`,
		limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pools []map[string]interface{}
	for rows.Next() {
		var p struct {
			PoolAddress string
			TokenA      string
			TokenB      string
			FeeBps      int32
			Type        string
			TvlUsd      float64
			Volume24h   float64
		}
		if err := rows.Scan(&p.PoolAddress, &p.TokenA, &p.TokenB, &p.FeeBps, &p.Type, &p.TvlUsd, &p.Volume24h); err != nil {
			return nil, err
		}
		pools = append(pools, map[string]interface{}{
			"poolAddress": p.PoolAddress,
			"tokenA":      p.TokenA,
			"tokenB":      p.TokenB,
			"feeBps":      p.FeeBps,
			"type":        p.Type,
			"tvlUsd":      p.TvlUsd,
			"volume24h":   p.Volume24h,
		})
	}
	return pools, nil
}

func GetTokenOHLCV(address, interval string) ([]map[string]interface{}, error) {
	var table string
	switch interval {
	case "1m":
		table = "ohlcv_1min"
	case "1h":
		table = "ohlcv_1hour"
	case "1d":
		table = "ohlcv_1day"
	default:
		return nil, fmt.Errorf("invalid interval")
	}

	rows, err := db.Query(context.Background(),
		fmt.Sprintf("SELECT bucket, open, high, low, close, volume_usd FROM %s WHERE asset_id = $1 ORDER BY bucket DESC LIMIT 100", table),
		address)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var bucket time.Time
		var open, high, low, close, volume float64
		if err := rows.Scan(&bucket, &open, &high, &low, &close, &volume); err != nil {
			return nil, err
		}
		results = append(results, map[string]interface{}{
			"bucket":    bucket,
			"open":      open,
			"high":      high,
			"low":       low,
			"close":     close,
			"volumeUsd": volume,
		})
	}
	return results, nil
}
