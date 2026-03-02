package tx_handlers

import (
	"strings"
	"time"

	"github.com/celerfi/stellar-indexer-go/models"
	"github.com/celerfi/stellar-indexer-go/utils"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/xdr"
)

func HandleTransferOperation(
	tx ingest.LedgerTransaction,
	op xdr.Operation,
	seq uint32,
	opIndex int,
	blockTime time.Time,
) {
	var from, to, asset string
	var amount float64

	sourceAddr := op.SourceAccount.Address()
	if sourceAddr == "" {
		sourceAddr = tx.Envelope.SourceAccount().Address()
	}

	switch op.Body.Type {
	case xdr.OperationTypePayment:
		p := op.Body.MustPaymentOp()
		from = sourceAddr
		to = p.Destination.Address()
		asset = utils.FormatAsset(p.Asset)
		amount = float64(p.Amount) / 1e7

	case xdr.OperationTypePathPaymentStrictReceive:
		p := op.Body.MustPathPaymentStrictReceiveOp()
		from = sourceAddr
		to = p.Destination.Address()
		asset = utils.FormatAsset(p.DestAsset)
		amount = float64(p.DestAmount) / 1e7

	case xdr.OperationTypePathPaymentStrictSend:
		p := op.Body.MustPathPaymentStrictSendOp()
		from = sourceAddr
		to = p.Destination.Address()
		asset = utils.FormatAsset(p.DestAsset)
		amount = float64(p.DestinationAmount) / 1e7

	default:
		return
	}

	transfer := models.Transfer{
		Timestamp:       blockTime,
		LedgerSequence:  seq,
		TransactionHash: tx.Result.TransactionHash.HexString(),
		OperationIndex:  opIndex,
		From:            from,
		To:              to,
		Asset:           asset,
		Amount:          amount,
	}

	utils.InsertTransfers([]models.Transfer{transfer})

	if asset != "XLM" {
		parts := strings.Split(asset, ":")
		if len(parts) > 1 {
			go AddTokenData(parts[1])
		}
	}
}
