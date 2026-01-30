package builder

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"

	"github.com/GoPolymarket/builder-relayer-go-client/internal/utils"
	"github.com/GoPolymarket/builder-relayer-go-client/pkg/signer"
	"github.com/GoPolymarket/builder-relayer-go-client/pkg/types"
)

func BuildSafeCreateTransactionRequest(s signer.Signer, safeContractConfig types.SafeContractConfig, args types.SafeCreateTransactionArgs) (*types.TransactionRequest, error) {
	safeFactory := safeContractConfig.SafeFactory

	domain := apitypes.TypedDataDomain{
		Name:              types.SafeFactoryName,
		ChainId:           (*math.HexOrDecimal256)(big.NewInt(args.ChainID)),
		VerifyingContract: safeFactory,
	}
	typesMap := apitypes.Types{
		"EIP712Domain": {
			{Name: "name", Type: "string"},
			{Name: "chainId", Type: "uint256"},
			{Name: "verifyingContract", Type: "address"},
		},
		"CreateProxy": {
			{Name: "paymentToken", Type: "address"},
			{Name: "payment", Type: "uint256"},
			{Name: "paymentReceiver", Type: "address"},
		},
	}
	payment, err := utils.ParseBigInt(args.Payment)
	if err != nil {
		return nil, fmt.Errorf("invalid payment: %w", err)
	}
	message := apitypes.TypedDataMessage{
		"paymentToken":    args.PaymentToken,
		"payment":         (*math.HexOrDecimal256)(payment),
		"paymentReceiver": args.PaymentReceiver,
	}

	sig, err := s.SignTypedData(&domain, typesMap, message, "CreateProxy")
	if err != nil {
		return nil, fmt.Errorf("sign safe create: %w", err)
	}

	safeAddress, err := DeriveSafeAddress(args.From, safeFactory)
	if err != nil {
		return nil, err
	}

	sigParams := types.SignatureParams{
		PaymentToken:    args.PaymentToken,
		Payment:         args.Payment,
		PaymentReceiver: args.PaymentReceiver,
	}

	return &types.TransactionRequest{
		Type:            string(types.TransactionTypeSafeCreate),
		From:            args.From,
		To:              safeFactory,
		ProxyWallet:     safeAddress,
		Data:            "0x",
		Signature:       hexutil.Encode(sig),
		SignatureParams: sigParams,
	}, nil
}
