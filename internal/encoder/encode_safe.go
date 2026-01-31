package encoder

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/GoPolymarket/go-builder-relayer-client/internal/utils"
	"github.com/GoPolymarket/go-builder-relayer-client/pkg/types"
)

var multisendABI abi.ABI

func init() {
	const multisendJSON = `[{"constant":false,"inputs":[{"internalType":"bytes","name":"transactions","type":"bytes"}],"name":"multiSend","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"}]`
	if err := json.Unmarshal([]byte(multisendJSON), &multisendABI); err != nil {
		panic(fmt.Sprintf("invalid multisend abi: %v", err))
	}
}

func CreateSafeMultisendTransaction(txns []types.SafeTransaction, safeMultisendAddress string) (types.SafeTransaction, error) {
	packed, err := encodePackedMultisend(txns)
	if err != nil {
		return types.SafeTransaction{}, err
	}
	data, err := multisendABI.Pack("multiSend", packed)
	if err != nil {
		return types.SafeTransaction{}, fmt.Errorf("pack multisend: %w", err)
	}

	return types.SafeTransaction{
		To:        safeMultisendAddress,
		Value:     "0",
		Data:      hexutil.Encode(data),
		Operation: types.OperationDelegateCall,
	}, nil
}

func encodePackedMultisend(txns []types.SafeTransaction) ([]byte, error) {
	out := make([]byte, 0)
	for _, tx := range txns {
		to := common.HexToAddress(tx.To)
		value, err := utils.ParseBigInt(tx.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid value: %w", err)
		}
		dataBytes, err := utils.DecodeHex(tx.Data)
		if err != nil {
			return nil, fmt.Errorf("invalid data: %w", err)
		}
		dataLen := big.NewInt(int64(len(dataBytes)))

		out = append(out, byte(tx.Operation))
		out = append(out, to.Bytes()...)
		out = append(out, utils.LeftPad32(value.Bytes())...)
		out = append(out, utils.LeftPad32(dataLen.Bytes())...)
		out = append(out, dataBytes...)
	}
	return out, nil
}
