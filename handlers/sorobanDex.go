package tx_handlers

import (
	"time"

	"github.com/celerfi/stellar-indexer-go/models"
	"github.com/celerfi/stellar-indexer-go/utils"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/xdr"
)

func ProcessSorobanContracts(tx ingest.LedgerTransaction, seq uint32, blocktime time.Time) {
	var tx_array []models.TransactionModels
	events, err := tx.GetContractEvents()
	if err != nil {
		return
	}
	for _, event := range events {
		body := event.Body.V0
		if body == nil {
			continue
		}
		scAddr := xdr.ScAddress{
			Type:       xdr.ScAddressTypeScAddressTypeContract,
			ContractId: event.ContractId,
		}
		event_symbol, _ := body.Topics[0].GetSym()
		eventname := string(event_symbol)
		switch eventname {
		case "trade":
			token_in_sym, _ := body.Topics[1].GetAddress()
			token_out_sym, _ := body.Topics[2].GetAddress()
			pool_addr, _ := scAddr.String()
			token_in, _ := token_in_sym.String()
			token_out, _ := token_out_sym.String()
			tx_instance := models.TransactionModels{}
			tx_instance.BlockTime = blocktime
			tx_instance.LedgerSequence = seq
			tx_instance.TransactionHash = tx.Result.TransactionHash.HexString()
			tx_instance.DexName = aquarius_dex_name
			tx_instance.SourceAccount = tx.Envelope.SourceAccount().GoString()
			tx_instance.Dex_type = "AMM"
			tx_instance.PoolAddress = pool_addr
			tx_instance.TokenIn = token_in
			tx_instance.TokenOut = token_out

			if vec, ok := body.Data.GetVec(); ok && vec != nil && len(*vec) >= 3 {
				tx_instance.AmountSold = utils.Int128ToDecimalFloat((*vec)[0].MustI128(), 7)
				tx_instance.AmountBought = utils.Int128ToDecimalFloat((*vec)[1].MustI128(), 7)
				tx_instance.DexFee = utils.Int128ToDecimalFloat((*vec)[2].MustI128(), 7)
			}

			tx_array = append(tx_array, tx_instance)
			go AddTokenData(token_in)
			go AddTokenData(token_out)
			go AddPoolDetails(pool_addr)
		}
	}
	// for _, item := range tx_array {
	// 	utils.PrettyPrintTransaction(item)
	// 	fmt.Println("=======================")
	// }
	utils.InsertTransactionsToDb(tx_array)
}
