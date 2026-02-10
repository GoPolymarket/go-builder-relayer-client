package relayer

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/GoPolymarket/go-builder-relayer-client/pkg/types"
)

func newPollTestClient() *RelayClient {
	builderCfg := &BuilderConfig{
		Local: &BuilderCredentials{
			Key:        "test-key",
			Secret:     "c2VjcmV0", // base64("secret")
			Passphrase: "test-pass",
		},
	}

	client, err := NewRelayClient("https://example.test", 137, nil, builderCfg, types.RelayerTxSafe)
	if err != nil {
		panic(err)
	}
	client.SetHTTPClient(NewHTTPClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == GetTransactionEndpoint {
			return newResponse(http.StatusOK, `[]`, nil), nil
		}
		return newResponse(http.StatusNotFound, `{"error":"not found"}`, nil), nil
	})}))
	return client
}

func TestPollUntilState_DoesNotSleepAfterFinalPoll(t *testing.T) {
	t.Parallel()

	client := newPollTestClient()

	start := time.Now()
	_, err := client.PollUntilState(
		context.Background(),
		"tx-id",
		[]types.RelayerTransactionState{types.StateMined},
		types.StateFailed,
		1,
		time.Second,
	)
	elapsed := time.Since(start)

	require.ErrorIs(t, err, types.ErrTransactionTimeout)
	assert.Less(t, elapsed, 250*time.Millisecond, "final iteration should not sleep")
}

func TestPollUntilState_CancelDuringSleepReturnsPromptly(t *testing.T) {
	t.Parallel()

	client := newPollTestClient()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := client.PollUntilState(
		ctx,
		"tx-id",
		[]types.RelayerTransactionState{types.StateMined},
		types.StateFailed,
		3,
		time.Second,
	)
	elapsed := time.Since(start)

	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, elapsed, 400*time.Millisecond, "cancelled context should interrupt sleep")
}
