package builder

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/GoPolymarket/builder-relayer-go-client/pkg/types"
)

func DeriveProxyWalletAddress(eoa string, proxyFactory string) (string, error) {
	if proxyFactory == "" {
		return "", types.ErrConfigUnsupported
	}
	addr := common.HexToAddress(eoa)
	salt := crypto.Keccak256(addr.Bytes())
	initCodeHash, err := hexutil.Decode(types.ProxyInitCodeHash)
	if err != nil {
		return "", fmt.Errorf("invalid proxy init code hash: %w", err)
	}
	proxyAddr := crypto.CreateAddress2(common.HexToAddress(proxyFactory), common.BytesToHash(salt), initCodeHash)
	return proxyAddr.Hex(), nil
}

func DeriveSafeAddress(eoa string, safeFactory string) (string, error) {
	if safeFactory == "" {
		return "", types.ErrConfigUnsupported
	}
	addr := common.HexToAddress(eoa)
	padded := common.LeftPadBytes(addr.Bytes(), 32)
	salt := crypto.Keccak256(padded)
	initCodeHash, err := hexutil.Decode(types.SafeInitCodeHash)
	if err != nil {
		return "", fmt.Errorf("invalid safe init code hash: %w", err)
	}
	safeAddr := crypto.CreateAddress2(common.HexToAddress(safeFactory), common.BytesToHash(salt), initCodeHash)
	return safeAddr.Hex(), nil
}
