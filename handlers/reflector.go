package tx_handlers

import (
	"fmt"
	"log"
	"time"

	"github.com/celerfi/stellar-indexer-go/models"
	"github.com/celerfi/stellar-indexer-go/utils"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/xdr"
)

var reflectorContracts = map[string]bool{
	"CAFJZQWSED6YAWZU3GWRTOCNPPCGBN32L7QV43XX5LZLFTK6JLN34DLN": true,
	"CALI2BYU2JE6WVRUFYTS6MSBNEHGJ35P4AVCZYF3B6QOE3QKOB2PLE6M": true,
	"CBKGPWGKSKZF52CFHMTRR23TBWTPMRDIYZ4O2P5VS65BMHYH4DXMCJZC": true,
}

// contractAssets maps contract address - ordered asset list fetched from the contract.
var contractAssets = map[string][]string{}

const (
	reflectorDecimals = 14
	reflectorSourceID = "reflector"
	setPriceFuncName  = "set_price"
)

// InitReflectorAssets calls assets() on each Reflector contract
// the index - assetID mapping. Call this once before starting the ledger stream.
func InitReflectorAssets() {
	for contractAddr := range reflectorContracts {
		assets, err := utils.GetReflectorAssets(contractAddr)
		if err != nil {
			log.Printf("failed to fetch assets for reflector contract %s: %v", contractAddr, err)
			continue
		}
		contractAssets[contractAddr] = assets
	}
}

func IsReflectorInvocation(op xdr.Operation) (string, bool) {
	invokeOp, ok := op.Body.GetInvokeHostFunctionOp()
	if !ok {
		return "", false
	}

	if invokeOp.HostFunction.Type != xdr.HostFunctionTypeHostFunctionTypeInvokeContract {
		return "", false
	}

	args := invokeOp.HostFunction.MustInvokeContract()

	contractAddress := xdr.ScAddress{
		Type:       args.ContractAddress.Type,
		ContractId: args.ContractAddress.ContractId,
	}
	addr, err := contractAddress.String()
	if err != nil || !reflectorContracts[addr] {
		return "", false
	}

	if string(args.FunctionName) != setPriceFuncName {
		return "", false
	}

	return addr, true
}

func HandleReflectorSetPrice(tx ingest.LedgerTransaction, op xdr.Operation, seq uint32, blocktime time.Time) {
	invokeOp, ok := op.Body.GetInvokeHostFunctionOp()
	if !ok {
		return
	}

	args := invokeOp.HostFunction.MustInvokeContract()

	if len(args.Args) < 2 {
		return
	}

	// Identify which contract this is so we use the right asset list
	contractAddress := xdr.ScAddress{
		Type:       args.ContractAddress.Type,
		ContractId: args.ContractAddress.ContractId,
	}
	contractAddr, err := contractAddress.String()
	if err != nil {
		return
	}

	assets, ok := contractAssets[contractAddr]
	if !ok || len(assets) == 0 {
		log.Printf("no asset list for reflector contract %s", contractAddr)
		return
	}

	tsVal, ok := args.Args[1].GetU64()
	if !ok {
		return
	}
	priceTime := time.UnixMilli(int64(tsVal)).UTC()

	updatesVec, ok := args.Args[0].GetVec()
	if !ok || updatesVec == nil {
		return
	}

	var ticks []models.PriceTick

	for i, scVal := range *updatesVec {
		if i >= len(assets) {
			break
		}

		priceI128, ok := scVal.GetI128()
		if !ok {
			continue
		}

		priceFloat := utils.Int128ToDecimalFloat(priceI128, reflectorDecimals)
		if priceFloat == 0 {
			continue
		}

		ticks = append(ticks, models.PriceTick{
			Timestamp:  priceTime,
			AssetID:    assets[i],
			SourceID:   reflectorSourceID,
			SourceType: "oracle_onchain",
			PriceUSD:   priceFloat,
			VolumeUSD:  nil,
			LedgerSeq:  seq,
			TxHash:     tx.Result.TransactionHash.HexString(),
		})
	}

	if len(ticks) > 0 {
		fmt.Printf("TICKS: %+v\n", ticks)
		utils.InsertPriceTicks(ticks)
	}
}
