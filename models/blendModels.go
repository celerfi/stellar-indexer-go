package models

import "time"

type BlendEvent struct {
	Timestamp       time.Time
	LedgerSequence  uint32
	TransactionHash string
	ContractID      string
	EventType       string // deposit, withdraw, borrow, repay, liquidate
	User            string // borrower or depositor
	Asset           string // primary asset involved
	Amount          float64
	
	// Liquidation specific
	Liquidator      string
	CollateralAsset string
	DebtAsset       string
}
