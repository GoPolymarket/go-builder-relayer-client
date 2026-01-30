# go-builder-relayer-client

Go client library for interacting with the Polymarket Relayer infrastructure.

## Installation

```bash
go get github.com/GoPolymarket/builder-relayer-go-client
```

## Configuration

Environment variables typically used:

```env
POLYMARKET_RELAYER_URL=https://relayer-v2-staging.polymarket.dev/
CHAIN_ID=80002
PRIVATE_KEY=your_private_key_here
BUILDER_API_KEY=your_api_key
BUILDER_SECRET=your_api_secret
BUILDER_PASS_PHRASE=your_passphrase
```

## Quick Start

### Basic Setup (SAFE)

```go
signer, _ := relayer.NewPrivateKeySigner(os.Getenv("PRIVATE_KEY"), 80002)

client, _ := relayer.NewRelayClient(
    os.Getenv("POLYMARKET_RELAYER_URL"),
    80002,
    signer,
    nil,
    relayer.RelayerTxSafe,
)
```

### With Builder Attribution (Local)

```go
builderCfg := &relayer.BuilderConfig{
    Local: &relayer.BuilderCredentials{
        Key:        os.Getenv("BUILDER_API_KEY"),
        Secret:     os.Getenv("BUILDER_SECRET"),
        Passphrase: os.Getenv("BUILDER_PASS_PHRASE"),
    },
}

client, _ := relayer.NewRelayClient(
    os.Getenv("POLYMARKET_RELAYER_URL"),
    137,
    signer,
    builderCfg,
    relayer.RelayerTxSafe,
)
```

### With Builder Attribution (Remote)

```go
builderCfg := &relayer.BuilderConfig{
    Remote: &relayer.BuilderRemoteConfig{
        Host:  "https://your-signer-api.com/v1/sign-builder",
        Token: os.Getenv("BUILDER_SIGNER_TOKEN"),
    },
}
```

### Execute Transactions (SAFE)

```go
ctx := context.Background()

approvalTx := relayer.Transaction{
    To:    "0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174", // USDC
    Data:  "0x...", // ABI-encoded calldata
    Value: "0",
}

resp, _ := client.Execute(ctx, []relayer.Transaction{approvalTx}, "usdc approval on the CTF")
result, _ := resp.Wait(ctx)
fmt.Println("Safe approval completed:", result.TransactionHash)
```

### Execute Transactions (PROXY)

```go
proxyClient, _ := relayer.NewRelayClient(
    os.Getenv("POLYMARKET_RELAYER_URL"),
    137,
    signer,
    builderCfg,
    relayer.RelayerTxProxy,
)

resp, _ := proxyClient.Execute(ctx, []relayer.Transaction{approvalTx}, "usdc approval on the CTF")
result, _ := resp.Wait(ctx)
fmt.Println("Proxy approval completed:", result.TransactionHash)
```

> Note: Proxy wallets are unsupported on Amoy (chainId 80002). They are available on Polygon (chainId 137).

### Deploy Safe

```go
resp, _ := client.Deploy(ctx)
result, _ := resp.Wait(ctx)
fmt.Println("Safe deployed:", result.ProxyAddress)
```

## Notes

- If a gas estimator is not configured on the signer, proxy transactions will fall back to a default gas limit of 10,000,000.
- `RelayerTxSafe` and `RelayerTxProxy` mirror the official TypeScript client behavior.
