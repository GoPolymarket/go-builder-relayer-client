# Go Polymarket Builder Relayer Client

[![Go Reference](https://pkg.go.dev/badge/github.com/GoPolymarket/go-builder-relayer-client.svg)](https://pkg.go.dev/github.com/GoPolymarket/go-builder-relayer-client)
[![Go Report Card](https://goreportcard.com/badge/github.com/GoPolymarket/go-builder-relayer-client)](https://goreportcard.com/report/github.com/GoPolymarket/go-builder-relayer-client)
[![License: Apache-2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

**Official docs alignment:** Implements Polymarket Order Attribution (builder auth headers for leaderboard/grants) and the Relayer Client flow (gasless transactions + Safe/Proxy deployment; builder authentication required with remote signing recommended); official docs: [Order Attribution](https://docs.polymarket.com/developers/builders/order-attribution), [Relayer Client](https://docs.polymarket.com/developers/builders/relayer-client).

A robust, type-safe Go client library for interacting with the **Polymarket Relayer** infrastructure. This SDK enables developers to execute gasless meta-transactions on Polygon, seamlessly integrating with Polymarket's exchange protocol.

## 🚀 Features

- **Gasless Transactions**: Submit transactions via Polymarket's Relayer without holding MATIC for gas.
- **Dual Wallet Support**: Full support for both **Gnosis Safe** (Smart Account) and **Proxy** (EOA-like) wallet architectures.
- **Builder Attribution**: Native support for the Polymarket Builder Rewards program, supporting both:
  - **Local Signing**: Manage API keys locally.
  - **Remote Signing**: Delegate signing to a secure remote service.
- **Resilient Networking**: Built-in automatic retries with exponential backoff for network stability.
- **EIP-712 Compliance**: Automated handling of EIP-712 typed data signing and domain separation.
- **Developer Friendly**: Comprehensive error handling, context support, and helper utilities for address derivation.

## 📦 Installation

```bash
go get github.com/GoPolymarket/go-builder-relayer-client
```

## 🛠 Configuration

The client can be configured using a struct or environment variables. Below are the standard environment variables used in examples:

| Variable | Description | Required | Example |
|----------|-------------|:--------:|---------|
| `POLYMARKET_RELAYER_URL` | The HTTP endpoint of the Relayer service (prod: `https://relayer-v2.polymarket.com/`, staging: `https://relayer-v2-staging.polymarket.dev`) | ✅ | `https://relayer-v2.polymarket.com/` |
| `CHAIN_ID` | Network Chain ID (137 for Polygon, 80002 for Amoy) | ✅ | `137` |
| `PRIVATE_KEY` | Your Ethereum Private Key (Hex) | ✅ | `0x...` |
| `BUILDER_API_KEY` | Builder API Key for attribution (required for local signing; omit when using remote signing) | ✅* | `...` |
| `BUILDER_SECRET` | Builder API Secret (required for local signing; omit when using remote signing) | ✅* | `...` |
| `BUILDER_PASS_PHRASE` | Builder Passphrase (required for local signing; omit when using remote signing) | ✅* | `...` |
| `BUILDER_REMOTE_HOST` | Remote signing service endpoint (optional; if set, SDK uses remote signing) | ❌ | `https://your-signer-api.com/v1/sign-builder` |
| `BUILDER_REMOTE_TOKEN` | Bearer token for remote signer (optional) | ❌ | `...` |

*`BUILDER_API_KEY`, `BUILDER_SECRET`, and `BUILDER_PASS_PHRASE` are required only for local signing.*

## ⚡ Quick Start

### 1. Initialize the Client

```go
package main

import (
    "os"
    relayer "github.com/GoPolymarket/go-builder-relayer-client"
    "github.com/GoPolymarket/go-builder-relayer-client/pkg/signer"
)

func main() {
    // 1. Create a Signer (manages your Private Key)
    pk := os.Getenv("PRIVATE_KEY")
    chainID := int64(137)
    signerInstance, _ := signer.NewPrivateKeySigner(pk, chainID)

    // 2. Configure Builder Authentication (Required by Relayer)
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

    // 3. Initialize the Relay Client
    // Use relayer.RelayerTxSafe for Gnosis Safe (Recommended)
    // Use relayer.RelayerTxProxy for Proxy wallets
    client, err := relayer.NewRelayClient(
        os.Getenv("POLYMARKET_RELAYER_URL"),
        chainID,
        signerInstance,
        builderCfg,
        relayer.RelayerTxSafe, 
    )
    if err != nil {
        panic(err)
    }
}
```

### 2. Execute a Transaction

```go
import (
    "context"
    "fmt"
    "github.com/GoPolymarket/go-builder-relayer-client/pkg/types"
)

func executeTx(client *relayer.RelayClient) {
    ctx := context.Background()

    // Define the transaction (e.g., USDC Approval)
    tx := types.Transaction{
        To:    "0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174", // USDC Contract
        Data:  "0x095ea7b3...", // Approve(spender, amount)
        Value: "0",
    }

    // Submit to Relayer
    resp, err := client.Execute(ctx, []types.Transaction{tx}, "Approve USDC")
    if err != nil {
        panic(err)
    }

    // Wait for mining
    receipt, err := resp.Wait(ctx)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Transaction confirmed! Hash: %s\n", receipt.TransactionHash)
}
```

### 3. Deploy a Safe Wallet

If you are using the `SAFE` mode and the user doesn't have a Safe deployed yet, you can deploy it via the Relayer:

```go
safeAddress, _ := relayer.DeriveSafeAddress(chainID, signerInstance.Address().Hex())
isDeployed, _ := client.GetDeployed(ctx, safeAddress)
if !isDeployed {
    resp, err := client.Deploy(ctx)
    if err != nil {
        panic(err)
    }
    receipt, _ := resp.Wait(ctx)
    fmt.Printf("Safe deployed at: %s\n", receipt.ProxyAddress)
}
```

### 4. End-to-End Example (Safe deploy → execute → receipt → attribution call)

See the full example in `examples/end_to_end/main.go` and run it with:

```bash
go run ./examples/end_to_end
```

The flow includes:
- Deterministic Safe address derivation
- Deploy Safe if needed
- Execute a gasless transaction and wait for receipt
- Optional attribution-bearing call (`GetTransactions`) to verify builder auth is attached

## 📚 Core Concepts

### Safe vs Proxy
- **Safe (`RelayerTxSafe`)**: Uses Gnosis Safe smart contracts. It is the modern standard for Polymarket accounts, supporting multisig features and batching. **Recommended for all new integrations.**
- **Proxy (`RelayerTxProxy`)**: Uses a custom proxy contract. Legacy standard, primarily supported on Polygon Mainnet (ChainID 137). Not available on Amoy Testnet.

### Builder Attribution
To participate in the Polymarket Rewards program, you must sign your Relayer requests with Builder Credentials (builder authentication is required by the Relayer).
- **Local**: You provide the API Key/Secret directly to the SDK.
- **Remote**: For higher security, you can run a separate signing service. The SDK will send the payload to your remote host for signing, keeping your secrets isolated.

### Builder Authentication (Required by Relayer)
The client injects builder headers on all Relayer requests, including:
- `POST /submit` (Execute + Deploy)
- `GET /transactions` (list transactions)
- `GET /transaction`
- `GET /deployed`
- `GET /nonce`
- `GET /relay-payload`

Required headers:
- `POLY_BUILDER_API_KEY`
- `POLY_BUILDER_PASSPHRASE`
- `POLY_BUILDER_SIGNATURE`
- `POLY_BUILDER_TIMESTAMP`

Local signing message format:
```
<timestamp><method><path><body>
```
where `timestamp` is a Unix millisecond timestamp (e.g., `Date.now()`), `body` is the JSON payload (with `"` quotes), and the signature is `HMAC-SHA256` over that string using your Builder Secret, base64url-encoded.

When signing requests with query parameters, include the full path with query string (e.g., `/deployed?address=0x...`).

Debug headers (redacted example):
```go
body := `{"foo":"bar"}`
headers, _ := builderCfg.Headers(ctx, "POST", "/submit", &body, 1700000000000)

mask := func(s string) string {
    if len(s) <= 8 {
        return "********"
    }
    return s[:4] + "..." + s[len(s)-4:]
}

for _, k := range []string{
    relayer.HeaderPolyBuilderAPIKey,
    relayer.HeaderPolyBuilderPassphrase,
    relayer.HeaderPolyBuilderTimestamp,
    relayer.HeaderPolyBuilderSignature,
} {
    fmt.Printf("%s: %s\n", k, mask(headers.Get(k)))
}
```

Example output (redacted):
```
POLY_BUILDER_API_KEY: pk_...c123
POLY_BUILDER_PASSPHRASE: pass...9xyz
POLY_BUILDER_TIMESTAMP: 1700000000000
POLY_BUILDER_SIGNATURE: JqZQ...9Q==
```

Remote signer payload (sent to `BUILDER_REMOTE_HOST`):
```json
{
  "method": "POST",
  "path": "/submit",
  "body": "{...}"
}
```
Remote signer response should include the headers above (keys may be returned as either `POLY_BUILDER_*` or `poly_builder_*`), and should set `POLY_BUILDER_TIMESTAMP` in Unix milliseconds.
See `examples/remote_signer_server` for a Go implementation.
Start it locally with:

```bash
BUILDER_API_KEY=... BUILDER_SECRET=... BUILDER_PASS_PHRASE=... \\
  go run ./examples/remote_signer_server
```

Then point your client to `BUILDER_REMOTE_HOST=http://localhost:8080/sign-builder`.

For full details, see `docs/builder-auth.md`.

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the Apache-2.0 License - see the [LICENSE](LICENSE) file for details.
