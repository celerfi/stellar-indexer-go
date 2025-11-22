package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/celerfi/stellar-indexer-go/config"
	tx_handlers "github.com/celerfi/stellar-indexer-go/handlers"
	"github.com/celerfi/stellar-indexer-go/utils"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/network"
	"github.com/stellar/go/xdr"
)

func main() {
	ctx := context.Background()
	fmt.Println("Starting up the Stellar Indexer ########")
	startSeq, err := utils.GetStartLedger()
	if err != nil {
		panic(err)
	}

	fmt.Println("Establishing the Indexer Connection ######## ", startSeq)
	backend := ledgerbackend.NewRPCLedgerBackend(ledgerbackend.RPCLedgerBackendOptions{
		RPCServerURL: config.RPC_URL,
	})
	defer backend.Close()
	if err := backend.PrepareRange(ctx, ledgerbackend.UnboundedRange(startSeq)); err != nil {
		log.Fatalf("Failed to prepare range: %v", err)
	}

	fmt.Println("Iterating over Testnet ledgers #########")
	seq := startSeq
	for {
		ledger, err := backend.GetLedger(ctx, seq)
		if err != nil {
			fmt.Printf("No more ledgers or error at sequence %d: %v\n", seq, err)
			// urgent error
			break
		}
		tx_reader, err := ingest.NewLedgerTransactionReaderFromLedgerCloseMeta(network.PublicNetworkPassphrase, ledger)
		if err != nil {
			// sendout the error
			
			startSeq++
			continue
		}
		closeTime := ledger.LedgerHeaderHistoryEntry().Header.ScpValue.CloseTime
		blockTime := time.Unix(int64(closeTime), 0).UTC()

		for {
			tx, readErr := tx_reader.Read()
			if errors.Is(readErr, io.EOF) {
				break
			}
			if readErr != nil {
				log.Fatalf("error reading transaction: %v", readErr)
			}
			txResult := tx.Result.Result
			tx_time := time.Unix(int64(tx_reader.GetHeader().Header.ScpValue.CloseTime), 0).UTC()
			if txResult.Result.Code != xdr.TransactionResultCodeTxSuccess {
				continue
			}
			for opIndex, op := range tx.Envelope.Operations() {
				opResults := txResult.Result.Results
				if opResults == nil || opIndex >= len(*opResults) {
					continue
				}
				switch op.Body.Type {
				case xdr.OperationTypeManageBuyOffer:
					go tx_handlers.HandleManageBuyTransaction(tx, op, seq, opIndex, opResults, blockTime)
				case xdr.OperationTypeManageSellOffer:
					go tx_handlers.HandleManageSellTransaction(tx, op, seq, opIndex, opResults, blockTime)
				case xdr.OperationTypeLiquidityPoolDeposit:
					// fmt.Println("found liquidity pool deposit")
				case xdr.OperationTypeLiquidityPoolWithdraw:
					// fmt.Println("found liquidity pool withdraw")
				case xdr.OperationTypeInvokeHostFunction:
					go tx_handlers.ProcessSorobanContracts(tx, seq, tx_time)

				}

			}

		}

		seq++
	}
}
