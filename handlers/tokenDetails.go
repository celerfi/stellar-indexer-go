package tx_handlers

import (
	"fmt"
	"strings"

	"github.com/celerfi/stellar-indexer-go/models"
	"github.com/celerfi/stellar-indexer-go/utils"
)

func AddTokenData(tokenHash string) {
	if utils.TokenExistsInDb(tokenHash) {
		return
	}

	var token *models.TokenInfo
	var err error

	if strings.HasPrefix(tokenHash, "C") {
		token, err = utils.GetSorobanTokenInfo(tokenHash)
	} else {
		token, err = utils.GetClassicTokenInfo(tokenHash)
	}

	if err != nil {
		fmt.Printf("failed to get token info for %s: %v\n", tokenHash, err)
		return
	}

	go utils.SaveTokenToDB(*token)
}
