package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	relayer "github.com/GoPolymarket/go-builder-relayer-client"
	"github.com/GoPolymarket/go-builder-relayer-client/pkg/signer"
	"github.com/GoPolymarket/go-builder-relayer-client/pkg/types"
)

func main() {
	relayerURL := os.Getenv("POLYMARKET_RELAYER_URL")
	chainIDStr := os.Getenv("CHAIN_ID")
	privateKey := os.Getenv("PRIVATE_KEY")
	if relayerURL == "" || chainIDStr == "" || privateKey == "" {
		panic("POLYMARKET_RELAYER_URL, CHAIN_ID, PRIVATE_KEY are required")
	}

	chainID, err := strconv.ParseInt(chainIDStr, 10, 64)
	if err != nil {
		panic(err)
	}

	signerInstance, err := signer.NewPrivateKeySigner(privateKey, chainID)
	if err != nil {
		panic(err)
	}

	builderCfg := buildBuilderConfig()
	if builderCfg == nil {
		panic("builder authentication is required: set BUILDER_REMOTE_HOST or local BUILDER_* env vars")
	}

	client, err := relayer.NewRelayClient(relayerURL, chainID, signerInstance, builderCfg, types.RelayerTxSafe)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	safeAddress, err := relayer.DeriveSafeAddress(chainID, signerInstance.Address().Hex())
	if err != nil {
		panic(err)
	}

	deployed, err := client.GetDeployed(ctx, safeAddress)
	if err != nil {
		panic(err)
	}
	if !deployed {
		fmt.Println("Safe not deployed, deploying...")
		deployResp, err := client.Deploy(ctx)
		if err != nil {
			panic(err)
		}
		receipt, err := deployResp.Wait(ctx)
		if err != nil {
			panic(err)
		}
		fmt.Println("Safe deployed at:", receipt.ProxyAddress)
	} else {
		fmt.Println("Safe already deployed:", safeAddress)
	}

	tx := types.Transaction{
		To:    "0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174", // USDC (Polygon)
		Data:  "0x",
		Value: "0",
	}

	execResp, err := client.Execute(ctx, []types.Transaction{tx}, "grant-demo: approve")
	if err != nil {
		panic(err)
	}
	receipt, err := execResp.Wait(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Println("Transaction confirmed:", receipt.TransactionHash)

	// Optional: attribution-bearing business call (also requires builder auth)
	txns, err := client.GetTransactions(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Println("Relayer transactions (authed):", len(txns))
}

func buildBuilderConfig() *relayer.BuilderConfig {
	if host := os.Getenv("BUILDER_REMOTE_HOST"); host != "" {
		return &relayer.BuilderConfig{Remote: &relayer.BuilderRemoteConfig{Host: host, Token: os.Getenv("BUILDER_REMOTE_TOKEN")}}
	}

	key := os.Getenv("BUILDER_API_KEY")
	secret := os.Getenv("BUILDER_SECRET")
	passphrase := os.Getenv("BUILDER_PASS_PHRASE")
	if key == "" || secret == "" || passphrase == "" {
		return nil
	}

	return &relayer.BuilderConfig{Local: &relayer.BuilderCredentials{Key: key, Secret: secret, Passphrase: passphrase}}
}
