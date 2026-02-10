package relayer

import (
	"context"
	"math/big"
	"net/http"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/GoPolymarket/go-builder-relayer-client/pkg/types"
)

type contextKey string

type captureEstimateSigner struct {
	address     common.Address
	lastCtx     context.Context
	estimateGas uint64
}

func (s *captureEstimateSigner) Address() common.Address {
	return s.address
}

func (s *captureEstimateSigner) ChainID() *big.Int {
	return big.NewInt(137)
}

func (s *captureEstimateSigner) SignMessage(message []byte) ([]byte, error) {
	sig := make([]byte, 65)
	sig[64] = 27
	return sig, nil
}

func (s *captureEstimateSigner) SignTypedData(_ *apitypes.TypedDataDomain, _ apitypes.Types, _ apitypes.TypedDataMessage, _ string) ([]byte, error) {
	sig := make([]byte, 65)
	sig[64] = 27
	return sig, nil
}

func (s *captureEstimateSigner) EstimateGas(ctx context.Context, _ ethereum.CallMsg) (uint64, error) {
	s.lastCtx = ctx
	return s.estimateGas, nil
}

func TestExecuteProxy_PropagatesRequestContextToGasEstimate(t *testing.T) {
	t.Parallel()

	signer := &captureEstimateSigner{
		address:     common.HexToAddress("0x1234567890123456789012345678901234567890"),
		estimateGas: 21000,
	}

	builderCfg := &BuilderConfig{
		Local: &BuilderCredentials{
			Key:        "test-key",
			Secret:     "c2VjcmV0", // base64("secret")
			Passphrase: "test-pass",
		},
	}

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case GetRelayPayloadEndpoint:
			return newResponse(http.StatusOK, `{"address":"0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","nonce":"1"}`, nil), nil
		case SubmitTransactionEndpoint:
			return newResponse(http.StatusOK, `{"transactionID":"tx-1","state":"STATE_NEW","transactionHash":"0xabc"}`, nil), nil
		default:
			return newResponse(http.StatusNotFound, `{"error":"not found"}`, nil), nil
		}
	})

	client, err := NewRelayClient("https://example.test", 137, signer, builderCfg, types.RelayerTxProxy)
	require.NoError(t, err)
	client.SetHTTPClient(NewHTTPClient(&http.Client{Transport: transport}))

	ctx := context.WithValue(context.Background(), contextKey("trace"), "trace-value")
	_, err = client.Execute(ctx, []types.Transaction{{
		To:    "0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174",
		Data:  "0x",
		Value: "0",
	}}, "ctx-propagation-test")
	require.NoError(t, err)

	require.NotNil(t, signer.lastCtx, "expected EstimateGas to be called")
	assert.Equal(t, "trace-value", signer.lastCtx.Value(contextKey("trace")))
}
