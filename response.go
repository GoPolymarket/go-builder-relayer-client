package relayer

import (
	"context"
	"time"

	"github.com/GoPolymarket/go-builder-relayer-client/pkg/types"
)

// ClientRelayerTransactionResponse wraps a relayer transaction response.
type ClientRelayerTransactionResponse struct {
	TransactionID   string
	State           string
	TransactionHash string
	client          *RelayClient
}

func (r *ClientRelayerTransactionResponse) GetTransaction(ctx context.Context) ([]types.RelayerTransaction, error) {
	return r.client.GetTransaction(ctx, r.TransactionID)
}

func (r *ClientRelayerTransactionResponse) Wait(ctx context.Context) (*types.RelayerTransaction, error) {
	return r.client.PollUntilState(
		ctx,
		r.TransactionID,
		[]types.RelayerTransactionState{types.StateMined, types.StateConfirmed},
		types.StateFailed,
		100,
		0,
	)
}

// WaitOptions configures polling behaviour for WaitWithOptions.
type WaitOptions struct {
	MaxPolls      int
	PollFrequency time.Duration
}

// WaitWithOptions polls until the transaction reaches a terminal state using the provided options.
func (r *ClientRelayerTransactionResponse) WaitWithOptions(ctx context.Context, opts WaitOptions) (*types.RelayerTransaction, error) {
	maxPolls := opts.MaxPolls
	if maxPolls <= 0 {
		maxPolls = 100
	}
	return r.client.PollUntilState(
		ctx,
		r.TransactionID,
		[]types.RelayerTransactionState{types.StateMined, types.StateConfirmed},
		types.StateFailed,
		maxPolls,
		opts.PollFrequency,
	)
}