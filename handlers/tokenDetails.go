package tx_handlers

import (
	"fmt"

	// "github.com/celerfi/stellar-indexer-go/models"
	"github.com/celerfi/stellar-indexer-go/utils"
)

// TODO
func AddTokenData(token_hash string) {
	if utils.TokenExistsInDb(token_hash) {
		return
	}
	token, err := utils.GetSorobanTokenInfo(token_hash)
	if err != nil {
		fmt.Println("failed to get decimals: ", token_hash)
		panic(err)
	}
	// fetch the token from the blockchain

	go utils.SaveTokenToDB(*token)
}
