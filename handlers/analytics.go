package tx_handlers

import (
	"fmt"
	"time"

	"github.com/celerfi/stellar-indexer-go/utils"
)

func StartAnalyticsWorker() {
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		for {
			select {
			case <-ticker.C:
				err := utils.RefreshAnalytics()
				if err != nil {
					fmt.Printf("Failed to refresh analytics views: %v\n", err)
				} else {
					fmt.Println("Successfully refreshed analytics views.")
				}
			}
		}
	}()
}