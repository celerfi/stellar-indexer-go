package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"time"

	tx_handlers "github.com/celerfi/stellar-indexer-go/handlers"
	"github.com/celerfi/stellar-indexer-go/models"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/xdr"
	"github.com/stellar/stellar-rpc/client"
	"github.com/stellar/go/network"
)

const (
	AQUARIUS_CONTRACT_ID = "GBNZILSTVQZ4R7IKQDGHYGY2QXL5QOFJYQMXPKWRRM5PAV7Y4M67AQUA"
)

func main() {
	ctx := context.Background()

	// Use the public SDF Testnet RPC for demo purpose
	// endpoint := "https://soroban-testnet.stellar.org"
	mainnet := "https://warmhearted-skilled-ensemble.stellar-mainnet.quiknode.pro/95b6d1245a6891a64c96706d267c4985209017ea"

	// Create a new RPC client
	rpcClient := client.NewClient(mainnet, nil)

	// Get the latest ledger sequence from the RPC server
	health, err := rpcClient.GetHealth(ctx)
	if err != nil {
		log.Fatalf("Failed to get RPC health: %v", err)
	}
	startSeq := health.LatestLedger

	// Configure the RPC Ledger Backend
	backend := ledgerbackend.NewRPCLedgerBackend(ledgerbackend.RPCLedgerBackendOptions{
		RPCServerURL: mainnet,
	})
	defer backend.Close()

	fmt.Printf("Prepare unbounded range starting with Testnet ledger sequence %d: \n", startSeq)
	// Prepare an unbounded range starting from the latest ledger
	if err := backend.PrepareRange(ctx, ledgerbackend.UnboundedRange(startSeq)); err != nil {
		log.Fatalf("Failed to prepare range: %v", err)
	}

	fmt.Println("Iterating over Testnet ledgers:")
	seq := startSeq
	for {
		ledger, err := backend.GetLedger(ctx, seq)
		if err != nil {
			fmt.Printf("No more ledgers or error at sequence %d: %v\n", seq, err)
			break
		}
		tx_reader, err := ingest.NewLedgerTransactionReaderFromLedgerCloseMeta(network.PublicNetworkPassphrase, ledger)
		if err != nil {
			panic("error reading transactions 1")
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
			if tx.Result.Result.Result.Code != xdr.TransactionResultCodeTxSuccess {
				// fmt.Println("skipped failed tx")
				continue // skip failed transactions
			}
			txResult := tx.Result.Result
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
					fmt.Println("found liquidity pool deposit")
				case xdr.OperationTypeLiquidityPoolWithdraw:
					fmt.Println("found liquidity pool withdraw")
				}

			}

		}

		seq++
	}

	fmt.Println("Done.")
}

func writeToFile(content string) error {
	// Open the file in append mode, create it if it doesn't exist
	file, err := os.OpenFile("sample.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("failed to open file ........")
		return err
	}
	defer file.Close()

	// Write the content to the end of the file
	_, err = file.WriteString(content)
	if err == nil {
		fmt.Println("saved ledger successfully.....")
	} else {
		fmt.Println("failed to save ledger ........")
	}
	return err
}

func HandleAquariusTx() {
	fmt.Println("handled an aquarius function ...............")
}

func formatAsset(a xdr.Asset) string {
	switch a.Type {
	case xdr.AssetTypeAssetTypeNative:
		return "XLM"
	case xdr.AssetTypeAssetTypeCreditAlphanum4:
		credit := a.MustAlphaNum4()
		return fmt.Sprintf("%s:%s", credit.AssetCode, credit.Issuer.Address())
	case xdr.AssetTypeAssetTypeCreditAlphanum12:
		credit := a.MustAlphaNum12()
		return fmt.Sprintf("%s:%s", credit.AssetCode, credit.Issuer.Address())
	default:
		return "Unknown"
	}
}

func printTransaction(t models.TransactionModels) {
	fmt.Printf("BlockTime: %v\n", t.BlockTime)
	fmt.Printf("LedgerSequence: %d\n", t.LedgerSequence)
	fmt.Printf("TransactionHash: %s\n", t.TransactionHash)
	fmt.Printf("OperationIndex: %d\n", t.OperationIndex)
	fmt.Printf("DexName: %s\n", t.DexName)
	fmt.Printf("SourceAccount: %s\n", t.SourceAccount)
	fmt.Printf("MarketSelling: %s\n", t.MarketSelling)
	fmt.Printf("MarketBuying: %s\n", t.MarketBuying)
	fmt.Printf("OfferID: %d\n", t.OfferID)
	fmt.Printf("MatchedOfferID: %d\n", t.MatchedOfferID)
	fmt.Printf("BuyerAccount: %s\n", t.BuyerAccount)
	fmt.Printf("SellerAccount: %s\n", t.SellerAccount)
	fmt.Printf("OfferBuyAmount: %v\n", t.OfferBuyAmount)
	fmt.Printf("OfferSellAmount: %v\n", t.OfferSellAmount)
	fmt.Printf("AmountBought: %v\n", t.AmountBought)
	fmt.Printf("AmountSold: %v\n", t.AmountSold)
	fmt.Printf("OfferPrice: %v\n", t.OfferPrice)
	fmt.Printf("Status: %s\n", t.Status)

	fmt.Println("OrderMatches:")
	for i, match := range t.OrderMatches {
		fmt.Printf("  Match #%d: %+v\n", i+1, match)
	}
}
