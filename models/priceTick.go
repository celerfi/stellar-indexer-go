package models

import "time"

type PriceTick struct {
	ID         uint64    `db:"id"`
	Timestamp  time.Time `db:"timestamp"`
	AssetID    string    `db:"asset_id"`
	SourceID   string    `db:"source_id"`
	SourceType string    `db:"source_type"`
	PriceUSD   float64   `db:"price_usd"`
	VolumeUSD  *float64  `db:"volume_usd"`
	LedgerSeq  uint32    `db:"ledger_seq"`
	TxHash     string    `db:"tx_hash"`
	IngestedAt time.Time `db:"ingested_at"`
}
