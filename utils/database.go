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

func TokenExistsInDb(token_hash string) bool {
	return false
}

func SaveTokenToDB(token models.Token) {

}

func getLastSuccessFullLedgerInDb() (uint32, error) {
	return 0, nil
}
