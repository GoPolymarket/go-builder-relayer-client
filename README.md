# Go Polymarket Builder Relayer Client

[![Go Reference](https://pkg.go.dev/badge/github.com/GoPolymarket/builder-relayer-go-client.svg)](https://pkg.go.dev/github.com/GoPolymarket/builder-relayer-go-client)
[![Go Report Card](https://goreportcard.com/badge/github.com/GoPolymarket/builder-relayer-go-client)](https://goreportcard.com/report/github.com/GoPolymarket/builder-relayer-go-client)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

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
go get github.com/GoPolymarket/builder-relayer-go-client
```

## 🛠 Configuration

The client can be configured using a struct or environment variables. Below are the standard environment variables used in examples:

| Variable | Description | Required | Example |
|----------|-------------|:--------:|---------|
| `POLYMARKET_RELAYER_URL` | The HTTP endpoint of the Relayer service | ✅ | `https://relayer-v2-staging.polymarket.dev` |
| `CHAIN_ID` | Network Chain ID (137 for Polygon, 80002 for Amoy) | ✅ | `137` |
| `PRIVATE_KEY` | Your Ethereum Private Key (Hex) | ✅ | `0x...` |
| `BUILDER_API_KEY` | Builder API Key for attribution | ❌ | `...` |
| `BUILDER_SECRET` | Builder API Secret | ❌ | `...` |
| `BUILDER_PASS_PHRASE` | Builder Passphrase | ❌ | `...` |

## ⚡ Quick Start

### 1. Initialize the Client

```go
package main

import (
    "os"
    relayer "github.com/GoPolymarket/builder-relayer-go-client"
    "github.com/GoPolymarket/builder-relayer-go-client/pkg/signer"
)

func main() {
    // 1. Create a Signer (manages your Private Key)
    pk := os.Getenv("PRIVATE_KEY")
    chainID := int64(137)
    signerInstance, _ := signer.NewPrivateKeySigner(pk, chainID)

    // 2. Configure Builder Attribution (Optional but Recommended)
    builderCfg := &relayer.BuilderConfig{
        Local: &relayer.BuilderCredentials{
            Key:        os.Getenv("BUILDER_API_KEY"),
            Secret:     os.Getenv("BUILDER_SECRET"),
            Passphrase: os.Getenv("BUILDER_PASS_PHRASE"),
        },
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
    "github.com/GoPolymarket/builder-relayer-go-client/pkg/types"
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
isDeployed, _ := client.GetDeployed(ctx, userAddress)
if !isDeployed {
    resp, err := client.Deploy(ctx)
    if err != nil {
        panic(err)
    }
    receipt, _ := resp.Wait(ctx)
    fmt.Printf("Safe deployed at: %s\n", receipt.ProxyAddress)
}
```

## 📚 Core Concepts

### Safe vs Proxy
- **Safe (`RelayerTxSafe`)**: Uses Gnosis Safe smart contracts. It is the modern standard for Polymarket accounts, supporting multisig features and batching. **Recommended for all new integrations.**
- **Proxy (`RelayerTxProxy`)**: Uses a custom proxy contract. Legacy standard, primarily supported on Polygon Mainnet (ChainID 137). Not available on Amoy Testnet.

### Builder Attribution
To participate in the Polymarket Rewards program, you must sign your requests with Builder Credentials.
- **Local**: You provide the API Key/Secret directly to the SDK.
- **Remote**: For higher security, you can run a separate signing service. The SDK will send the payload to your remote host for signing, keeping your secrets isolated.

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
