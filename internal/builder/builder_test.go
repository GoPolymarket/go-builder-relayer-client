package builder

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/GoPolymarket/go-builder-relayer-client/pkg/signer"
	"github.com/GoPolymarket/go-builder-relayer-client/pkg/types"
)

// Helper to create a test signer
func createTestSigner(t *testing.T) signer.Signer {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	s := &signer.PrivateKeySigner{}
	// Manually setting fields since we don't want to expose private fields in a real test but
	// for this internal test we might need a constructor that accepts *ecdsa.PrivateKey
	// or just use NewPrivateKeySigner with hex.

	keyBytes := crypto.FromECDSA(key)
	hexKey := common.Bytes2Hex(keyBytes)

	s, err = signer.NewPrivateKeySigner(hexKey, 80002)
	require.NoError(t, err)

	return s
}

func TestDeriveSafeAddress(t *testing.T) {
	// Known test vector (hypothetical, but deterministic)
	eoa := "0x1234567890123456789012345678901234567890"
	safeFactory := "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	// Should fail with empty factory
	_, err := DeriveSafeAddress(eoa, "")
	assert.ErrorIs(t, err, types.ErrConfigUnsupported)

	// Should succeed with valid inputs
	addr, err := DeriveSafeAddress(eoa, safeFactory)
	assert.NoError(t, err)
	assert.NotEmpty(t, addr)
	assert.True(t, common.IsHexAddress(addr))
}

func TestDeriveProxyWalletAddress(t *testing.T) {
	eoa := "0x1234567890123456789012345678901234567890"
	proxyFactory := "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

	// Should fail with empty factory
	_, err := DeriveProxyWalletAddress(eoa, "")
	assert.ErrorIs(t, err, types.ErrConfigUnsupported)

	// Should succeed with valid inputs
	addr, err := DeriveProxyWalletAddress(eoa, proxyFactory)
	assert.NoError(t, err)
	assert.NotEmpty(t, addr)
	assert.True(t, common.IsHexAddress(addr))
}

func TestBuildSafeCreateTransactionRequest(t *testing.T) {
	s := createTestSigner(t)

	safeConfig := types.SafeContractConfig{
		SafeFactory: "0x3333333333333333333333333333333333333333",
	}

	args := types.SafeCreateTransactionArgs{
		From:            s.Address().Hex(),
		ChainID:         80002,
		PaymentToken:    "0x4444444444444444444444444444444444444444",
		Payment:         "1000000",
		PaymentReceiver: "0x5555555555555555555555555555555555555555",
	}

	req, err := BuildSafeCreateTransactionRequest(s, safeConfig, args)
	require.NoError(t, err)
	require.NotNil(t, req)

	assert.Equal(t, string(types.TransactionTypeSafeCreate), req.Type)
	assert.Equal(t, args.From, req.From)
	assert.Equal(t, safeConfig.SafeFactory, req.To)
	assert.Equal(t, "0x", req.Data)

	// Verify signature params
	assert.Equal(t, args.PaymentToken, req.SignatureParams.PaymentToken)
	assert.Equal(t, args.Payment, req.SignatureParams.Payment)
	assert.Equal(t, args.PaymentReceiver, req.SignatureParams.PaymentReceiver)

	// Verify proxy wallet is derived and valid
	assert.NotEmpty(t, req.ProxyWallet)
	assert.True(t, common.IsHexAddress(req.ProxyWallet))

	// Verify signature presence
	assert.NotEmpty(t, req.Signature)
}

func TestBuildSafeCreateTransactionRequest_InvalidPayment(t *testing.T) {
	s := createTestSigner(t)
	safeConfig := types.SafeContractConfig{SafeFactory: "0x123"}

	args := types.SafeCreateTransactionArgs{
		From:    s.Address().Hex(),
		ChainID: 80002,
		Payment: "invalid-number",
	}

	_, err := BuildSafeCreateTransactionRequest(s, safeConfig, args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid payment")
}
