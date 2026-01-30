package types

import "errors"

var (
	ErrSignerUnavailable    = errors.New("signer is needed to interact with this endpoint")
	ErrSafeDeployed         = errors.New("safe already deployed")
	ErrSafeNotDeployed      = errors.New("safe not deployed")
	ErrConfigUnsupported    = errors.New("config is not supported on the chainId")
	ErrMissingBuilderConfig = errors.New("builder config is required")
	ErrMissingGasEstimator  = errors.New("gas estimator is required")
)
