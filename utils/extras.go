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
	"sync"
	"time"

	"github.com/celerfi/stellar-indexer-go/config"
	"github.com/celerfi/stellar-indexer-go/models"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/strkey"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
)

var (
	tokenCache   sync.Map
	rpcSemaphore = make(chan struct{}, 5)
)

func getRPCConfig() models.GetTokenConfig {
	return models.GetTokenConfig{
		RPCUrl:     config.RPC_URL,
		HorizonUrl: config.HORIZON_URL,
		Timeout:    10 * time.Second,
	}
}

func GetSorobanTokenInfo(contractAddress string) (*models.TokenInfo, error) {
	if val, ok := tokenCache.Load(contractAddress); ok {
		return val.(*models.TokenInfo), nil
	}

	rpcSemaphore <- struct{}{}
	defer func() { <-rpcSemaphore }()

	rpcConfig := getRPCConfig()
	info := &models.TokenInfo{
		ContractAddress: contractAddress,
	}

	scAddr, err := createScAddressFromString(contractAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid contract address: %w", err)
	}

	info.Symbol, err = getTokenSymbol(scAddr, rpcConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get symbol: %w", err)
	}

	info.Name, err = getTokenName(scAddr, rpcConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get name: %w", err)
	}

	info.Decimals, err = getTokenDecimals(scAddr, rpcConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get decimals: %w", err)
	}

	info.AdminAddress, _ = getTokenAdmin(scAddr, rpcConfig)
	info.TotalSupply, _ = getTokenTotalSupply(scAddr, rpcConfig)

	assetCode, issuer, isSAC := parseSACName(info.Name)
	info.IsSAC = isSAC

	if isSAC && info.TotalSupply == "" {
		supplyInfo, err := getClassicAssetSupply(assetCode, issuer, rpcConfig)
		if err == nil {
			info.TotalSupply = fmt.Sprintf("%.7f", supplyInfo.Total)
			info.SupplyBreakdown = supplyInfo

			assetInfo, err := getClassicAssetInfo(assetCode, issuer, rpcConfig)
			if err == nil {
				info.NumAccounts = assetInfo.NumAccounts
				info.IsAuthRevocable = assetInfo.Flags.AuthRevocable
				info.IsMintable = !assetInfo.Flags.AuthImmutable
			}
		}
	}

	if !isSAC && strings.HasPrefix(info.AdminAddress, "G") {
		mintable, revocable, err := getIssuerFlags(info.AdminAddress, rpcConfig)
		if err == nil {
			info.IsMintable = mintable
			info.IsAuthRevocable = revocable
		}
	}

	tokenCache.Store(contractAddress, info)
	return info, nil
}

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

func callReadOnlyFunction(contractAddress xdr.ScAddress, functionName string, args xdr.ScVec, cfg models.GetTokenConfig) (xdr.ScVal, error) {
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

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", cfg.RPCUrl, bytes.NewBuffer(requestBody))
	if err != nil {
		return xdr.ScVal{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: cfg.Timeout}
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

func getTokenSymbol(scAddr xdr.ScAddress, cfg models.GetTokenConfig) (string, error) {
	scVal, err := callReadOnlyFunction(scAddr, "symbol", xdr.ScVec{}, cfg)
	if err != nil {
		return "", err
	}
	if scVal.Type != xdr.ScValTypeScvString {
		return "", fmt.Errorf("unexpected result type")
	}
	return string(scVal.MustStr()), nil
}

func getTokenName(scAddr xdr.ScAddress, cfg models.GetTokenConfig) (string, error) {
	scVal, err := callReadOnlyFunction(scAddr, "name", xdr.ScVec{}, cfg)
	if err != nil {
		return "", err
	}
	if scVal.Type != xdr.ScValTypeScvString {
		return "", fmt.Errorf("unexpected result type")
	}
	return string(scVal.MustStr()), nil
}

func getTokenDecimals(scAddr xdr.ScAddress, cfg models.GetTokenConfig) (uint32, error) {
	scVal, err := callReadOnlyFunction(scAddr, "decimals", xdr.ScVec{}, cfg)
	if err != nil {
		return 0, err
	}
	if scVal.Type != xdr.ScValTypeScvU32 {
		return 0, fmt.Errorf("unexpected result type")
	}
	return uint32(*scVal.U32), nil
}

func getTokenTotalSupply(scAddr xdr.ScAddress, cfg models.GetTokenConfig) (string, error) {
	scVal, err := callReadOnlyFunction(scAddr, "total_supply", xdr.ScVec{}, cfg)
	if err != nil {
		return "", nil
	}
	if scVal.Type != xdr.ScValTypeScvI128 {
		return "", nil
	}
	return int128PartsToBigInt(scVal.MustI128()).String(), nil
}

func getTokenAdmin(scAddr xdr.ScAddress, cfg models.GetTokenConfig) (string, error) {
	scVal, err := callReadOnlyFunction(scAddr, "admin", xdr.ScVec{}, cfg)
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

func getClassicAssetInfo(assetCode, issuer string, cfg models.GetTokenConfig) (*classicAssetInfo, error) {
	client := &horizonclient.Client{HorizonURL: cfg.HorizonUrl}

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

func getClassicAssetSupply(assetCode, issuer string, cfg models.GetTokenConfig) (*models.SupplyBreakdown, error) {
	client := &horizonclient.Client{HorizonURL: cfg.HorizonUrl}

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

func getIssuerFlags(adminAddress string, cfg models.GetTokenConfig) (isMintable, isAuthRevocable bool, err error) {
	client := &horizonclient.Client{HorizonURL: cfg.HorizonUrl}

	account, err := client.AccountDetail(horizonclient.AccountRequest{
		AccountID: adminAddress,
	})
	if err != nil {
		return false, false, err
	}

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

func GetClassicTokenInfo(issuerAddress string) (*models.TokenInfo, error) {
	cfg := getRPCConfig()
	client := &horizonclient.Client{HorizonURL: cfg.HorizonUrl}

	account, err := client.AccountDetail(horizonclient.AccountRequest{
		AccountID: issuerAddress,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch account: %w", err)
	}

	info := &models.TokenInfo{
		AdminAddress: issuerAddress,
		IsSAC:        false,
	}

	info.Name = account.HomeDomain

	assets, err := client.Assets(horizonclient.AssetRequest{
		ForAssetIssuer: issuerAddress,
	})
	if err != nil || len(assets.Embedded.Records) == 0 {
		return info, nil
	}

	record := assets.Embedded.Records[0]
	info.Symbol = record.Code

	info.ContractAddress = strings.Join([]string{record.Code, issuerAddress}, ":")

	supply, err := getClassicAssetSupply(record.Code, issuerAddress, cfg)
	if err == nil {
		info.TotalSupply = fmt.Sprintf("%.7f", supply.Total)
		info.SupplyBreakdown = supply
	}

	assetInfo, err := getClassicAssetInfo(record.Code, issuerAddress, cfg)
	if err == nil {
		info.NumAccounts = assetInfo.NumAccounts
		info.IsAuthRevocable = assetInfo.Flags.AuthRevocable
		info.IsMintable = !assetInfo.Flags.AuthImmutable
	}

	return info, nil
}

func GetReflectorAssets(contractAddr string) ([]string, error) {
	cfg := getRPCConfig()
	scAddr, err := createScAddressFromString(contractAddr)
	if err != nil {
		return nil, fmt.Errorf("invalid contract address: %w", err)
	}

	result, err := callReadOnlyFunction(scAddr, "assets", xdr.ScVec{}, cfg)
	if err != nil {
		return nil, fmt.Errorf("assets() call failed: %w", err)
	}

	vec, ok := result.GetVec()
	if !ok || vec == nil {
		return nil, fmt.Errorf("unexpected result type from assets()")
	}

	var assets []string
	for i, item := range *vec {
		assetID, err := decodeReflectorAsset(item)
		if err != nil {
			return nil, fmt.Errorf("decode asset[%d]: %w", i, err)
		}
		assets = append(assets, assetID)
	}

	return assets, nil
}

func decodeReflectorAsset(val xdr.ScVal) (string, error) {
	vec, ok := val.GetVec()
	if !ok || vec == nil || len(*vec) < 2 {
		return "", fmt.Errorf("expected ScVec with 2 elements")
	}

	variant, ok := (*vec)[0].GetSym()
	if !ok {
		return "", fmt.Errorf("expected Symbol as first element")
	}

	switch string(variant) {
	case "Other":
		sym, ok := (*vec)[1].GetSym()
		if !ok {
			return "", fmt.Errorf("expected Symbol value in Other variant")
		}
		return string(sym), nil

	case "Stellar":
		addr, ok := (*vec)[1].GetAddress()
		if !ok {
			return "", fmt.Errorf("expected Address value in Stellar variant")
		}
		addrStr, err := addr.String()
		if err != nil {
			return "", err
		}
		return addrStr, nil

	default:
		return "", fmt.Errorf("unknown Asset variant: %s", string(variant))
	}
}

func GetSoroswapPairTokens(contractAddr string) (string, string, error) {
	cfg := getRPCConfig()
	scAddr, err := createScAddressFromString(contractAddr)
	if err != nil {
		return "", "", err
	}

	t0Val, err := callReadOnlyFunction(scAddr, "token0", xdr.ScVec{}, cfg)
	if err != nil {
		return "", "", err
	}
	t0, ok := t0Val.GetAddress()
	if !ok {
		return "", "", fmt.Errorf("token0 not an address")
	}
	token0, _ := t0.String()

	t1Val, err := callReadOnlyFunction(scAddr, "token1", xdr.ScVec{}, cfg)
	if err != nil {
		return "", "", err
	}
	t1, ok := t1Val.GetAddress()
	if !ok {
		return "", "", fmt.Errorf("token1 not an address")
	}
	token1, _ := t1.String()

	return token0, token1, nil
}

func Ptr[T any](v T) *T {
	return &v
}

type classicAssetInfo struct {
	NumAccounts int
	Flags       struct {
		AuthRevocable bool
		AuthImmutable bool
	}
}
