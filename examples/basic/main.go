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

	chainID, _ := strconv.ParseInt(chainIDStr, 10, 64)
	signerInstance, err := signer.NewPrivateKeySigner(privateKey, chainID)
	if err != nil {
		panic(err)
	}

	builderCfg := &relayer.BuilderConfig{}
	if remoteHost := os.Getenv("BUILDER_REMOTE_HOST"); remoteHost != "" {
		builderCfg.Remote = &relayer.BuilderRemoteConfig{
			Host:  remoteHost,
			Token: os.Getenv("BUILDER_REMOTE_TOKEN"),
		}
	} else {
		builderCfg.Local = &relayer.BuilderCredentials{
			Key:        os.Getenv("BUILDER_API_KEY"),
			Secret:     os.Getenv("BUILDER_SECRET"),
			Passphrase: os.Getenv("BUILDER_PASS_PHRASE"),
		}
	}
	if !builderCfg.IsValid() {
		panic("builder authentication is required: set BUILDER_REMOTE_HOST or local BUILDER_* env vars")
	}

	client, err := relayer.NewRelayClient(relayerURL, chainID, signerInstance, builderCfg, types.RelayerTxSafe)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	txn := types.Transaction{
		To:    "0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174", // USDC
		Data:  "0x",
		Value: "0",
	}

	resp, err := client.Execute(ctx, []types.Transaction{txn}, "example execution")
	if err != nil {
		panic(err)
	}

	result, err := resp.Wait(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Println("Transaction mined:", result.TransactionHash)
}
