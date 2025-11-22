package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/celerfi/stellar-indexer-go/models"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/strkey"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
)

// DefaultConfig returns sensible defaults for mainnet
var rpc_config = models.GetTokenConfig{
		RPCUrl:     "https://soroban-rpc.mainnet.stellar.gateway.fm",
		HorizonUrl: "https://horizon.stellar.org",
		Timeout:    10 * time.Second,
	}


// GetTokenInfo is the main function to get all token information
// It automatically detects SACs and fetches appropriate data
func GetSorobanTokenInfo(contractAddress string) (*models.TokenInfo, error) {
	info := &models.TokenInfo{
		ContractAddress: contractAddress,
	}

	// Create ScAddress from contract string
	scAddr, err := createScAddressFromString(contractAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid contract address: %w", err)
	}

	// Get basic token info from contract
	info.Symbol, err = getTokenSymbol(scAddr, rpc_config)
	if err != nil {
		return nil, fmt.Errorf("failed to get symbol: %w", err)
	}

	info.Name, err = getTokenName(scAddr, rpc_config)
	if err != nil {
		return nil, fmt.Errorf("failed to get name: %w", err)
	}

	info.Decimals, err = getTokenDecimals(scAddr, rpc_config)
	if err != nil {
		return nil, fmt.Errorf("failed to get decimals: %w", err)
	}

	// Try to get admin address (may fail for some contracts)
	info.AdminAddress, _ = getTokenAdmin(scAddr, rpc_config)

	// Try to get total supply from contract (works for custom tokens)
	info.TotalSupply, _ = getTokenTotalSupply(scAddr, rpc_config)

	// Check if it's a SAC by parsing the name format
	assetCode, issuer, isSAC := parseSACName(info.Name)
	info.IsSAC = isSAC

	// If it's a SAC and we don't have total supply, get it from Horizon
	if isSAC && info.TotalSupply == "" {
		supplyInfo, err := getClassicAssetSupply(assetCode, issuer, rpc_config)
		if err == nil {
			info.TotalSupply = fmt.Sprintf("%.7f", supplyInfo.Total)
			info.SupplyBreakdown = supplyInfo

			// Get number of holders
			assetInfo, err := getClassicAssetInfo(assetCode, issuer, rpc_config)
			if err == nil {
				info.NumAccounts = assetInfo.NumAccounts
				info.IsAuthRevocable = assetInfo.Flags.AuthRevocable
				info.IsMintable = !assetInfo.Flags.AuthImmutable
			}
		}
	}

	// For non-SACs, get issuer flags if admin is a G-address
	if !isSAC && strings.HasPrefix(info.AdminAddress, "G") {
		mintable, revocable, err := getIssuerFlags(info.AdminAddress, rpc_config)
		if err == nil {
			info.IsMintable = mintable
			info.IsAuthRevocable = revocable
		}
	}

	return info, nil
}

// --- Internal Helper Functions ---

func createScAddressFromString(addressStr string) (xdr.ScAddress, error) {
	var scAddr xdr.ScAddress

	if len(addressStr) == 0 {
		return scAddr, fmt.Errorf("empty address string")
	}

	if addressStr[0] == 'G' {
		rawBytes, err := strkey.Decode(strkey.VersionByteAccountID, addressStr)
		if err != nil {
			return scAddr, fmt.Errorf("failed to decode account address: %w", err)
		}

		var accountID xdr.AccountId
		var uint256 xdr.Uint256
		copy(uint256[:], rawBytes)
		accountID.Type = xdr.PublicKeyTypePublicKeyTypeEd25519
		accountID.Ed25519 = &uint256

		scAddr.Type = xdr.ScAddressTypeScAddressTypeAccount
		scAddr.AccountId = &accountID

	} else if addressStr[0] == 'C' {
		rawBytes, err := strkey.Decode(strkey.VersionByteContract, addressStr)
		if err != nil {
			return scAddr, fmt.Errorf("failed to decode contract address: %w", err)
		}

		var contractId xdr.ContractId
		copy(contractId[:], rawBytes)

		scAddr.Type = xdr.ScAddressTypeScAddressTypeContract
		scAddr.ContractId = &contractId
	} else {
		return scAddr, fmt.Errorf("invalid address format: must start with G or C")
	}

	return scAddr, nil
}

func callReadOnlyFunction(contractAddress xdr.ScAddress, functionName string, args xdr.ScVec, config models.GetTokenConfig) (xdr.ScVal, error) {
	invokeContractArgs := xdr.InvokeContractArgs{
		ContractAddress: contractAddress,
		FunctionName:    xdr.ScSymbol(functionName),
		Args:            args,
	}

	hostFunction := xdr.HostFunction{
		Type:           xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
		InvokeContract: &invokeContractArgs,
	}

	dummyAccountAddress := "GAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWHF"
	dummySource := txnbuild.NewSimpleAccount(dummyAccountAddress, 0)

	invokeHostFunctionOp := &txnbuild.InvokeHostFunction{
		HostFunction:  hostFunction,
		SourceAccount: dummySource.AccountID,
	}

	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &dummySource,
			IncrementSequenceNum: true,
			Operations:           []txnbuild.Operation{invokeHostFunctionOp},
			BaseFee:              100,
			Preconditions: txnbuild.Preconditions{
				TimeBounds: txnbuild.NewInfiniteTimeout(),
			},
		},
	)
	if err != nil {
		return xdr.ScVal{}, fmt.Errorf("failed to build transaction: %w", err)
	}

	txEnvelope := tx.ToXDR()
	txEnvelopeXDR, err := xdr.MarshalBase64(txEnvelope)
	if err != nil {
		return xdr.ScVal{}, fmt.Errorf("failed to marshal tx envelope: %w", err)
	}

	rpcRequest := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "simulateTransaction",
		"params": map[string]interface{}{
			"transaction": txEnvelopeXDR,
		},
	}

	requestBody, err := json.Marshal(rpcRequest)
	if err != nil {
		return xdr.ScVal{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", config.RPCUrl, bytes.NewBuffer(requestBody))
	if err != nil {
		return xdr.ScVal{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: config.Timeout}
	resp, err := client.Do(req)
	if err != nil {
		return xdr.ScVal{}, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return xdr.ScVal{}, fmt.Errorf("failed to read response: %w", err)
	}

	var rpcResponse struct {
		Result struct {
			Error   string `json:"error,omitempty"`
			Results []struct {
				XDR string `json:"xdr"`
			} `json:"results"`
		} `json:"result,omitempty"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}

	if err := json.Unmarshal(body, &rpcResponse); err != nil {
		return xdr.ScVal{}, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if rpcResponse.Error != nil {
		return xdr.ScVal{}, fmt.Errorf("RPC error: %s", rpcResponse.Error.Message)
	}
	if rpcResponse.Result.Error != "" {
		return xdr.ScVal{}, fmt.Errorf("simulation error: %s", rpcResponse.Result.Error)
	}
	if len(rpcResponse.Result.Results) == 0 {
		return xdr.ScVal{}, fmt.Errorf("no results returned")
	}

	var scVal xdr.ScVal
	if err := xdr.SafeUnmarshalBase64(rpcResponse.Result.Results[0].XDR, &scVal); err != nil {
		return xdr.ScVal{}, fmt.Errorf("failed to unmarshal result XDR: %w", err)
	}

	return scVal, nil
}

func getTokenSymbol(scAddr xdr.ScAddress, config models.GetTokenConfig) (string, error) {
	scVal, err := callReadOnlyFunction(scAddr, "symbol", xdr.ScVec{}, config)
	if err != nil {
		return "", err
	}
	if scVal.Type != xdr.ScValTypeScvString {
		return "", fmt.Errorf("unexpected result type")
	}
	return string(scVal.MustStr()), nil
}

func getTokenName(scAddr xdr.ScAddress, config models.GetTokenConfig) (string, error) {
	scVal, err := callReadOnlyFunction(scAddr, "name", xdr.ScVec{}, config)
	if err != nil {
		return "", err
	}
	if scVal.Type != xdr.ScValTypeScvString {
		return "", fmt.Errorf("unexpected result type")
	}
	return string(scVal.MustStr()), nil
}

func getTokenDecimals(scAddr xdr.ScAddress, config models.GetTokenConfig) (uint32, error) {
	scVal, err := callReadOnlyFunction(scAddr, "decimals", xdr.ScVec{}, config)
	if err != nil {
		return 0, err
	}
	if scVal.Type != xdr.ScValTypeScvU32 {
		return 0, fmt.Errorf("unexpected result type")
	}
	return uint32(*scVal.U32), nil
}

func getTokenTotalSupply(scAddr xdr.ScAddress, config models.GetTokenConfig) (string, error) {
	scVal, err := callReadOnlyFunction(scAddr, "total_supply", xdr.ScVec{}, config)
	if err != nil {
		return "", nil // Not an error, just means it's probably a SAC
	}
	if scVal.Type != xdr.ScValTypeScvI128 {
		return "", nil
	}
	return int128PartsToBigInt(scVal.MustI128()).String(), nil
}

func getTokenAdmin(scAddr xdr.ScAddress, config models.GetTokenConfig) (string, error) {
	scVal, err := callReadOnlyFunction(scAddr, "admin", xdr.ScVec{}, config)
	if err != nil {
		return "", err
	}
	if scVal.Type != xdr.ScValTypeScvAddress {
		return "", fmt.Errorf("unexpected result type")
	}
	adminAddr := scVal.MustAddress()
	return adminAddr.String()
}

func int128PartsToBigInt(parts xdr.Int128Parts) *big.Int {
	hi := big.NewInt(int64(parts.Hi))
	lo := new(big.Int)
	lo.SetUint64(uint64(parts.Lo))
	hi.Lsh(hi, 64)
	hi.Add(hi, lo)
	return hi
}

func parseSACName(name string) (assetCode, issuer string, isSAC bool) {
	parts := strings.Split(name, ":")
	if len(parts) != 2 {
		return "", "", false
	}

	assetCode = parts[0]
	issuer = parts[1]

	if !strings.HasPrefix(issuer, "G") {
		return "", "", false
	}

	return assetCode, issuer, true
}

type classicAssetInfo struct {
	NumAccounts int
	Flags       struct {
		AuthRevocable bool
		AuthImmutable bool
	}
}

func getClassicAssetInfo(assetCode, issuer string, config models.GetTokenConfig) (*classicAssetInfo, error) {
	client := &horizonclient.Client{HorizonURL: config.HorizonUrl}

	response, err := client.Assets(horizonclient.AssetRequest{
		ForAssetCode:   assetCode,
		ForAssetIssuer: issuer,
	})
	if err != nil {
		return nil, err
	}

	if len(response.Embedded.Records) == 0 {
		return nil, fmt.Errorf("asset not found")
	}

	record := response.Embedded.Records[0]

	// Calculate total number of accounts from all categories
	totalAccounts := int(record.Accounts.Authorized) +
		int(record.Accounts.AuthorizedToMaintainLiabilities) +
		int(record.Accounts.Unauthorized)

	return &classicAssetInfo{
		NumAccounts: totalAccounts,
		Flags: struct {
			AuthRevocable bool
			AuthImmutable bool
		}{
			AuthRevocable: record.Flags.AuthRevocable,
			AuthImmutable: record.Flags.AuthImmutable,
		},
	}, nil
}

func getClassicAssetSupply(assetCode, issuer string, config models.GetTokenConfig) (*models.SupplyBreakdown, error) {
	client := &horizonclient.Client{HorizonURL: config.HorizonUrl}

	response, err := client.Assets(horizonclient.AssetRequest{
		ForAssetCode:   assetCode,
		ForAssetIssuer: issuer,
	})
	if err != nil {
		return nil, err
	}

	if len(response.Embedded.Records) == 0 {
		return nil, fmt.Errorf("asset not found")
	}

	record := response.Embedded.Records[0]

	authorized := parseFloat(record.Balances.Authorized)
	liquidityPools := parseFloat(record.LiquidityPoolsAmount)
	contracts := parseFloat(record.ContractsAmount)
	claimable := parseFloat(record.ClaimableBalancesAmount)

	return &models.SupplyBreakdown{
		Authorized:        authorized,
		LiquidityPools:    liquidityPools,
		Contracts:         contracts,
		ClaimableBalances: claimable,
		Total:             authorized + liquidityPools + contracts + claimable,
	}, nil
}

func getIssuerFlags(adminAddress string, config models.GetTokenConfig) (isMintable, isAuthRevocable bool, err error) {
	client := &horizonclient.Client{HorizonURL: config.HorizonUrl}

	account, err := client.AccountDetail(horizonclient.AccountRequest{
		AccountID: adminAddress,
	})
	if err != nil {
		return false, false, err
	}

	// Check if account is locked (master weight = 0)
	isLocked := account.Thresholds.MedThreshold == 0 && account.Thresholds.HighThreshold == 0
	isMintable = !isLocked

	return isMintable, account.Flags.AuthRevocable, nil
}

func parseFloat(val string) float64 {
	if val == "" {
		return 0.0
	}
	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0.0
	}
	return f
}

func GetClassicTokenInfo(string) (*models.TokenInfo, error) {
	//
	return nil, nil
}