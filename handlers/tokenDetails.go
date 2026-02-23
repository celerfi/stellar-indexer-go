package tx_handlers

import (
	"log"
	"strings"

	"github.com/celerfi/stellar-indexer-go/utils"
)

func AddTokenData(token_hash string) {
	if utils.TokenExistsInDb(token_hash) {
		return
	}

	if strings.HasPrefix(token_hash, "G") {
		token, err := utils.GetClassicTokenInfo(token_hash)
		if err != nil {
			log.Printf("failed to get classic token info for %s: %v", token_hash, err)
			return
		}
		log.Println(token)
		return
	}

	token, err := utils.GetSorobanTokenInfo(token_hash)
	if err != nil {
		log.Printf("failed to get soroban token info for %s: %v", token_hash, err)
		return
	}
	log.Println(token)
}
