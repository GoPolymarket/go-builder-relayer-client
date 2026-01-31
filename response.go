package relayer

import (
	"context"

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