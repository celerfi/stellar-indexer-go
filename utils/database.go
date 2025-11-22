package utils

import (
	"github.com/celerfi/stellar-indexer-go/models"
)

// var db = connectToDb()

// func connectToDb() *pgxpool.Pool {
// 	databaseUrl := fmt.Sprintf("postgres://%s:%s@%s:5432/%s", config.DB_USER, config.DB_PASSWORD, config.DB_HOST, config.DB_NAME)
// 	ctx := context.Background()

// 	poolConfig, err := pgxpool.ParseConfig(databaseUrl)
// 	if err != nil {
// 		panic(err)
// 	}

// 	poolConfig.MaxConns = 5
// 	dbPool, err := pgxpool.NewWithConfig(ctx, poolConfig)

// 	if err != nil {
// 		panic(err)
// 	}

// 	fmt.Println("Successfully connected to database!")
// 	return dbPool

// }

func InsertTransactionsToDb(transactions []models.TransactionModels) {
	// _ = db

	//TODO: do something here
}

func TokenExistsInDb(token_hash string) bool {
	return false
}

func SaveTokenToDB(token models.Token) {

}

func getLastSuccessFullLedgerInDb() (uint32, error){
	return 0, nil
}
