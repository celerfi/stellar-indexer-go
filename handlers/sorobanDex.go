package tx_handlers

import (
	"fmt"
	"time"

	"github.com/celerfi/stellar-indexer-go/utils"
	"github.com/stellar/go/ingest"
)

func ProcessSorobanContracts(tx ingest.LedgerTransaction, seq uint32, blocktime time.Time) {
	for _, op := range tx.Envelope.Operations() {
		if _, ok := IsReflectorInvocation(op); ok {
			go HandleReflectorSetPrice(tx, op, seq, blocktime)
			return
		}
	}

	go HandleBlendEvent(tx, seq, blocktime)

	events, err := tx.GetContractEvents()
	if err != nil {
		return
	}

	for _, event := range events {
		contractID := fmt.Sprintf("%x", *event.ContractId)

		if contractID == utils.AQUARIUS_CONTRACT_ID_HEX {
			go ProcessAquariusTransaction(tx, seq, blocktime)
			return
		}

		if contractID == utils.SOROSWAP_CONTRACT_ID_HEX {
			go ProcessSoroswapEvents(tx, seq, blocktime)
			return
		}
	}
}
