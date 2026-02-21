package tx_handlers

import (
	"fmt"
	"time"

	"github.com/celerfi/stellar-indexer-go/models"
	"github.com/celerfi/stellar-indexer-go/utils"
)

func AddPoolDetails(poolAddress string) {
	if utils.PoolExistsInDb(poolAddress) {
		return
	}

	// todo - fetching actual pool details {tokens,fee & type}
	pool := models.LiquidityPool{
		PoolAddress: poolAddress,
		TokenA:      "UNKNOWN_TOKEN_A",
		TokenB:      "UNKNOWN_TOKEN_B",
		FeeBps:      30, // 0.3% for now
		Type:        "CONSTANT_PRODUCT",
		CreatedAt:   time.Now().UTC(),
	}

	utils.SavePoolToDB(pool)
	fmt.Printf("Saved new placeholder pool: %s\n", poolAddress)
}