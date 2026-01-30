package builder

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"

	"github.com/GoPolymarket/builder-relayer-go-client/internal/encoder"
	"github.com/GoPolymarket/builder-relayer-go-client/internal/utils"
	"github.com/GoPolymarket/builder-relayer-go-client/pkg/signer"
	"github.com/GoPolymarket/builder-relayer-go-client/pkg/types"
)

func aggregateSafeTransactions(txns []types.SafeTransaction, safeMultisend string) (types.SafeTransaction, error) {
	if len(txns) == 1 {
		return txns[0], nil
	}
	return encoder.CreateSafeMultisendTransaction(txns, safeMultisend)
}

func createSafeStructHash(chainID int64, safeAddress string, txn types.SafeTransaction, nonce string) ([]byte, error) {
	value, err := utils.ParseBigInt(txn.Value)
	if err != nil {
		return nil, fmt.Errorf("invalid value: %w", err)
	}
	nonceInt, err := utils.ParseBigInt(nonce)
	if err != nil {
		return nil, fmt.Errorf("invalid nonce: %w", err)
	}

	domain := apitypes.TypedDataDomain{
		ChainId:           (*math.HexOrDecimal256)(big.NewInt(chainID)),
		VerifyingContract: safeAddress,
	}
	typesMap := apitypes.Types{
		"EIP712Domain": {
			{Name: "chainId", Type: "uint256"},
			{Name: "verifyingContract", Type: "address"},
		},
		"SafeTx": {
			{Name: "to", Type: "address"},
			{Name: "value", Type: "uint256"},
			{Name: "data", Type: "bytes"},
			{Name: "operation", Type: "uint8"},
			{Name: "safeTxGas", Type: "uint256"},
			{Name: "baseGas", Type: "uint256"},
			{Name: "gasPrice", Type: "uint256"},
			{Name: "gasToken", Type: "address"},
			{Name: "refundReceiver", Type: "address"},
			{Name: "nonce", Type: "uint256"},
		},
	}
	message := apitypes.TypedDataMessage{
		"to":             txn.To,
		"value":          (*math.HexOrDecimal256)(value),
		"data":           txn.Data,
		"operation":      (*math.HexOrDecimal256)(big.NewInt(int64(txn.Operation))),
		"safeTxGas":      (*math.HexOrDecimal256)(big.NewInt(0)),
		"baseGas":        (*math.HexOrDecimal256)(big.NewInt(0)),
		"gasPrice":       (*math.HexOrDecimal256)(big.NewInt(0)),
		"gasToken":       types.ZeroAddress,
		"refundReceiver": types.ZeroAddress,
		"nonce":          (*math.HexOrDecimal256)(nonceInt),
	}

	typedData := apitypes.TypedData{
		Types:       typesMap,
		PrimaryType: "SafeTx",
		Domain:      domain,
		Message:     message,
	}

	hash, _, err := apitypes.TypedDataAndHash(typedData)
	if err != nil {
		return nil, fmt.Errorf("hash safe tx: %w", err)
	}
	return hash, nil
}

func BuildSafeTransactionRequest(s signer.Signer, args types.SafeTransactionArgs, safeContractConfig types.SafeContractConfig, metadata string) (*types.TransactionRequest, error) {
	transaction, err := aggregateSafeTransactions(args.Transactions, safeContractConfig.SafeMultisend)
	if err != nil {
		return nil, err
	}
	safeAddress, err := DeriveSafeAddress(args.From, safeContractConfig.SafeFactory)
	if err != nil {
		return nil, err
	}

	structHash, err := createSafeStructHash(args.ChainID, safeAddress, transaction, args.Nonce)
	if err != nil {
		return nil, err
	}

	sig, err := s.SignMessage(structHash)
	if err != nil {
		return nil, fmt.Errorf("sign safe tx: %w", err)
	}

	packedSig, err := utils.SplitAndPackSig(sig)
	if err != nil {
		return nil, err
	}

	sigParams := types.SignatureParams{
		GasPrice:       "0",
		Operation:      fmt.Sprintf("%d", transaction.Operation),
		SafeTxnGas:     "0",
		BaseGas:        "0",
		GasToken:       types.ZeroAddress,
		RefundReceiver: types.ZeroAddress,
	}

	return &types.TransactionRequest{
		Type:            string(types.TransactionTypeSafe),
		From:            args.From,
		To:              transaction.To,
		ProxyWallet:     safeAddress,
		Data:            transaction.Data,
		Nonce:           args.Nonce,
		Signature:       packedSig,
		SignatureParams: sigParams,
		Metadata:        metadata,
	}, nil
}
