package builder

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/GoPolymarket/builder-relayer-go-client/internal/utils"
	"github.com/GoPolymarket/builder-relayer-go-client/pkg/signer"
	"github.com/GoPolymarket/builder-relayer-go-client/pkg/types"
)

const defaultGasLimit = 10_000_000

func createProxyStructHash(
	from string,
	to string,
	data string,
	txFee string,
	gasPrice string,
	gasLimit string,
	nonce string,
	relayHub string,
	relay string,
) ([]byte, error) {
	fromAddr := common.HexToAddress(from)
	toAddr := common.HexToAddress(to)
	relayHubAddr := common.HexToAddress(relayHub)
	relayAddr := common.HexToAddress(relay)

	dataBytes, err := utils.DecodeHex(data)
	if err != nil {
		return nil, fmt.Errorf("decode data: %w", err)
	}

	txFeeInt, err := utils.ParseBigInt(txFee)
	if err != nil {
		return nil, fmt.Errorf("invalid txFee: %w", err)
	}
	gasPriceInt, err := utils.ParseBigInt(gasPrice)
	if err != nil {
		return nil, fmt.Errorf("invalid gasPrice: %w", err)
	}
	gasLimitInt, err := utils.ParseBigInt(gasLimit)
	if err != nil {
		return nil, fmt.Errorf("invalid gasLimit: %w", err)
	}
	nonceInt, err := utils.ParseBigInt(nonce)
	if err != nil {
		return nil, fmt.Errorf("invalid nonce: %w", err)
	}

	encodedTxFee := utils.LeftPad32(txFeeInt.Bytes())
	encodedGasPrice := utils.LeftPad32(gasPriceInt.Bytes())
	encodedGasLimit := utils.LeftPad32(gasLimitInt.Bytes())
	encodedNonce := utils.LeftPad32(nonceInt.Bytes())

	dataToHash := make([]byte, 0, 4+20+20+len(dataBytes)+32*4+20+20)
	dataToHash = append(dataToHash, []byte("rlx:")...)
	dataToHash = append(dataToHash, fromAddr.Bytes()...)
	dataToHash = append(dataToHash, toAddr.Bytes()...)
	dataToHash = append(dataToHash, dataBytes...)
	dataToHash = append(dataToHash, encodedTxFee...)
	dataToHash = append(dataToHash, encodedGasPrice...)
	dataToHash = append(dataToHash, encodedGasLimit...)
	dataToHash = append(dataToHash, encodedNonce...)
	dataToHash = append(dataToHash, relayHubAddr.Bytes()...)
	dataToHash = append(dataToHash, relayAddr.Bytes()...)

	hash := crypto.Keccak256(dataToHash)
	return hash, nil
}

func BuildProxyTransactionRequest(s signer.Signer, args types.ProxyTransactionArgs, proxyContractConfig types.ProxyContractConfig, metadata string) (*types.TransactionRequest, error) {
	proxyFactory := proxyContractConfig.ProxyFactory
	proxyWallet, err := DeriveProxyWalletAddress(args.From, proxyFactory)
	if err != nil {
		return nil, err
	}

	gasLimitStr, err := getGasLimit(context.Background(), s, proxyFactory, args)
	if err != nil {
		gasLimitStr = fmt.Sprintf("%d", defaultGasLimit)
	}

	relayerFee := "0"
	sigParams := types.SignatureParams{
		GasPrice:   args.GasPrice,
		GasLimit:   gasLimitStr,
		RelayerFee: relayerFee,
		RelayHub:   proxyContractConfig.RelayHub,
		Relay:      args.Relay,
	}

	txHash, err := createProxyStructHash(
		args.From,
		proxyFactory,
		args.Data,
		relayerFee,
		args.GasPrice,
		gasLimitStr,
		args.Nonce,
		proxyContractConfig.RelayHub,
		args.Relay,
	)
	if err != nil {
		return nil, err
	}

	sig, err := s.SignMessage(txHash)
	if err != nil {
		return nil, fmt.Errorf("sign proxy tx: %w", err)
	}

	return &types.TransactionRequest{
		Type:            string(types.TransactionTypeProxy),
		From:            args.From,
		To:              proxyFactory,
		ProxyWallet:     proxyWallet,
		Data:            args.Data,
		Nonce:           args.Nonce,
		Signature:       hexutil.Encode(sig),
		SignatureParams: sigParams,
		Metadata:        metadata,
	}, nil
}

func getGasLimit(ctx context.Context, s signer.Signer, to string, args types.ProxyTransactionArgs) (string, error) {
	if args.GasLimit != "" && args.GasLimit != "0" {
		return args.GasLimit, nil
	}
	toAddr := common.HexToAddress(to)
	dataBytes, err := utils.DecodeHex(args.Data)
	if err != nil {
		return "", err
	}

	msg := ethereum.CallMsg{
		From: common.HexToAddress(args.From),
		To:   &toAddr,
		Data: dataBytes,
	}
	gasLimit, err := s.EstimateGas(ctx, msg)
	if err != nil {
		return "", err
	}
	return new(big.Int).SetUint64(gasLimit).String(), nil
}
