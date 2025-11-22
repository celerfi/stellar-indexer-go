package models

import "time"

type TokenInfo struct {
	ContractAddress string           `json:"contract_address"`
	Symbol          string           `json:"symbol"`
	Name            string           `json:"name"`
	Decimals        uint32           `json:"decimals"`
	TotalSupply     string           `json:"total_supply"`
	AdminAddress    string           `json:"admin_address"`
	IsAuthRevocable bool             `json:"is_auth_revocable"`
	IsMintable      bool             `json:"is_mintable"`
	IsSAC           bool             `json:"is_sac"` // Is Stellar Asset Contract
	NumAccounts     int              `json:"num_accounts,omitempty"`
	SupplyBreakdown *SupplyBreakdown `json:"supply_breakdown,omitempty"`
}

// SupplyBreakdown contains detailed supply information for SACs
type SupplyBreakdown struct {
	Authorized        float64 `json:"authorized"`         // In user wallets
	LiquidityPools    float64 `json:"liquidity_pools"`    // In liquidity pools
	Contracts         float64 `json:"contracts"`          // In smart contracts
	ClaimableBalances float64 `json:"claimable_balances"` // In claimable balances
	Total             float64 `json:"total"`
}

// Config for the helper
type GetTokenConfig struct {
	RPCUrl     string
	HorizonUrl string
	Timeout    time.Duration
}
