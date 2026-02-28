package utils

import (
	"context"
	"errors"

	"github.com/celerfi/stellar-indexer-go/config"
	client "github.com/stellar/go/clients/rpcclient"
)

func GetStartLedger() (uint32, error) {
	switch config.DEPLOYMENT_ENVIRONMENT {
	case "testing":
		return getNodeLatestLedger()
	case "production":
		lastLedger, err := getLastSuccessFullLedgerInDb()
		if err != nil {
			return 0, err
		}
		if lastLedger == 0 {
			return getNodeLatestLedger()
		}
		return lastLedger, nil
	default:
		return 0, errors.New("set the deployment environment config: options (testing, production)")
	}
}

func getNodeLatestLedger() (uint32, error) {
	rpcClient := client.NewClient(config.RPC_URL, nil)
	health, err := rpcClient.GetHealth(context.Background())
	if err != nil {
		return 0, err
	}
	return health.LatestLedger, nil
}
