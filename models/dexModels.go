package models

import "time"

type TransactionModels struct {
	BlockTime       time.Time
	LedgerSequence  uint32
	TransactionHash string
	OperationIndex  int
	DexName         string
	SourceAccount   string // signer/source of the offer (instead of "Signature")
	TokenIn         string
	TokenOut        string
	OfferID         uint64
	Dex_type        string
	PoolAddress     string
	MatchedOfferID  uint64 // (optional; if specific counteroffer was matched)
	BuyerAccount    string
	SellerAccount   string
	OfferBuyAmount  float64
	OfferSellAmount float64
	AmountBought    float64
	AmountSold      float64
	OfferPrice      float64
	DexFee          float64
	Status          string
	OrderMatches    []OrderMatch // plural should be singular in struct definition
}

type OrderMatch struct {
	OrderType    string // e.g. "counter_offer"
	AmountBought float64
	AmountSold   float64
	AssetBought  string
	AssetSold    string
	Owner        string // owner of the counter offer
	OfferID      uint64 // matched offer ID
}

type Token struct {
	TokenHash       string
	Symbol          string
	Decimals        uint32
	TokenName       string
	TotalSupply     int
	TokenAuthority  string
	IsAuthRevocable bool
	IsMintable      bool
	Token_toml      map[string]any
}

type Transfers struct {
}
