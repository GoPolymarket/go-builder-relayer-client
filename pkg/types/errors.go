package types

import sdkerrors "github.com/GoPolymarket/go-builder-relayer-client/pkg/errors"

var (
	ErrSignerUnavailable    = sdkerrors.ErrSignerUnavailable
	ErrSafeDeployed         = sdkerrors.ErrSafeDeployed
	ErrSafeNotDeployed      = sdkerrors.ErrSafeNotDeployed
	ErrConfigUnsupported    = sdkerrors.ErrConfigUnsupported
	ErrMissingBuilderConfig = sdkerrors.ErrMissingBuilderConfig
	ErrMissingGasEstimator  = sdkerrors.ErrMissingGasEstimator
	ErrNoTransactions       = sdkerrors.ErrNoTransactions
	ErrUnsupportedTxType    = sdkerrors.ErrUnsupportedTxType
	ErrInvalidNoncePayload  = sdkerrors.ErrInvalidNoncePayload
	ErrTransactionFailed    = sdkerrors.ErrTransactionFailed
	ErrTransactionTimeout   = sdkerrors.ErrTransactionTimeout
)
