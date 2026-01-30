package encoder

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/GoPolymarket/builder-relayer-go-client/internal/utils"
	"github.com/GoPolymarket/builder-relayer-go-client/pkg/types"
)

var proxyFactoryABI abi.ABI

func init() {
	const proxyJSON = `[{"constant":false,"inputs":[{"components":[{"name":"typeCode","type":"uint8"},{"name":"to","type":"address"},{"name":"value","type":"uint256"},{"name":"data","type":"bytes"}],"name":"calls","type":"tuple[]"}],"name":"proxy","outputs":[{"name":"returnValues","type":"bytes[]"}],"payable":true,"stateMutability":"payable","type":"function"}]`
	if err := json.Unmarshal([]byte(proxyJSON), &proxyFactoryABI); err != nil {
		panic(fmt.Sprintf("invalid proxy abi: %v", err))
	}
}

type proxyCall struct {
	TypeCode uint8
	To       common.Address
	Value    *big.Int
	Data     []byte
}

func EncodeProxyTransactionData(txns []types.ProxyTransaction) (string, error) {
	calls := make([]proxyCall, 0, len(txns))
	for _, tx := range txns {
		to := common.HexToAddress(tx.To)
		value, err := utils.ParseBigInt(tx.Value)
		if err != nil {
			return "", fmt.Errorf("invalid value: %w", err)
		}
		dataBytes, err := utils.DecodeHex(tx.Data)
		if err != nil {
			return "", fmt.Errorf("invalid data: %w", err)
		}
		calls = append(calls, proxyCall{
			TypeCode: uint8(tx.TypeCode),
			To:       to,
			Value:    value,
			Data:     dataBytes,
		})
	}

	data, err := proxyFactoryABI.Pack("proxy", calls)
	if err != nil {
		return "", fmt.Errorf("pack proxy data: %w", err)
	}
	return hexutil.Encode(data), nil
}
