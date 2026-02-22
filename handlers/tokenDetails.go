package tx_handlers

import (
	"fmt"
	"strings"

	"github.com/celerfi/stellar-indexer-go/utils"
)

func AddTokenData(tokenHash string) {
	// only processing soroban 'C' addresses for now. learnt yhat classic 'G' addresses are not contracts and causes errors
	if !strings.HasPrefix(tokenHash, "C") {
		fmt.Printf("Skipping classic asset: %s\n", tokenHash)
		return
	}

	if utils.TokenExistsInDb(tokenHash) {
		return
	}

	token, err := utils.GetSorobanTokenInfo(tokenHash)
	if err != nil {
		fmt.Printf("Failed to get Soroban token info for %s: %v\n", tokenHash, err)
		return
	}

	go utils.SaveTokenToDB(*token)
}
