package signer

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/GoPolymarket/builder-relayer-go-client/pkg/types"
)

// Signer defines the interface for an EIP-712 capable signing entity.
type Signer interface {
	Address() common.Address
	ChainID() *big.Int
	SignMessage(message []byte) ([]byte, error)
	SignTypedData(domain *apitypes.TypedDataDomain, types apitypes.Types, message apitypes.TypedDataMessage, primaryType string) ([]byte, error)
	EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error)
}

// GasEstimator allows injection of an RPC client for gas estimation.
type GasEstimator interface {
	EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error)
}

// PrivateKeySigner implements Signer using a local private key.
type PrivateKeySigner struct {
	key       *ecdsa.PrivateKey
	address   common.Address
	chainID   *big.Int
	estimator GasEstimator
}

// NewPrivateKeySigner creates a new signer from a hex-encoded private key.
func NewPrivateKeySigner(hexKey string, chainID int64) (*PrivateKeySigner, error) {
	if len(hexKey) > 2 && hexKey[:2] == "0x" {
		hexKey = hexKey[2:]
	}
	key, err := crypto.HexToECDSA(hexKey)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}
	return &PrivateKeySigner{
		key:     key,
		address: crypto.PubkeyToAddress(key.PublicKey),
		chainID: big.NewInt(chainID),
	}, nil
}

// WithGasEstimator attaches an estimator to the signer.
func (s *PrivateKeySigner) WithGasEstimator(est GasEstimator) *PrivateKeySigner {
	s.estimator = est
	return s
}

func (s *PrivateKeySigner) Address() common.Address {
	return s.address
}

func (s *PrivateKeySigner) ChainID() *big.Int {
	return s.chainID
}

// SignMessage signs a 32-byte hash with the EIP-191 prefix.
func (s *PrivateKeySigner) SignMessage(message []byte) ([]byte, error) {
	if len(message) == 0 {
		return nil, errors.New("message is required")
	}
	hash := accounts.TextHash(message)
	sig, err := crypto.Sign(hash, s.key)
	if err != nil {
		return nil, fmt.Errorf("sign message: %w", err)
	}
	return sig, nil
}

// SignTypedData signs EIP-712 typed data and normalizes V to 27/28.
func (s *PrivateKeySigner) SignTypedData(domain *apitypes.TypedDataDomain, types apitypes.Types, message apitypes.TypedDataMessage, primaryType string) ([]byte, error) {
	typedData := apitypes.TypedData{
		Types:       types,
		PrimaryType: primaryType,
		Domain:      *domain,
		Message:     message,
	}

	sighash, _, err := apitypes.TypedDataAndHash(typedData)
	if err != nil {
		return nil, fmt.Errorf("failed to hash typed data: %w", err)
	}

	signature, err := crypto.Sign(sighash, s.key)
	if err != nil {
		return nil, fmt.Errorf("failed to sign hash: %w", err)
	}

	if signature[64] < 27 {
		signature[64] += 27
	}

	return signature, nil
}

func (s *PrivateKeySigner) EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error) {
	if s.estimator == nil {
		return 0, types.ErrMissingGasEstimator
	}
	return s.estimator.EstimateGas(ctx, msg)
}
