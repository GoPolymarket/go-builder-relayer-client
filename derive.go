package relayer

import "github.com/GoPolymarket/go-builder-relayer-client/internal/builder"

// DeriveSafeAddress returns the deterministic Safe address for an EOA on a supported chain.
func DeriveSafeAddress(chainID int64, eoa string) (string, error) {
	config, err := GetContractConfig(chainID)
	if err != nil {
		return "", err
	}
	return builder.DeriveSafeAddress(eoa, config.SafeContracts.SafeFactory)
}

// DeriveProxyAddress returns the deterministic Proxy address for an EOA on a supported chain.
func DeriveProxyAddress(chainID int64, eoa string) (string, error) {
	config, err := GetContractConfig(chainID)
	if err != nil {
		return "", err
	}
	return builder.DeriveProxyWalletAddress(eoa, config.ProxyContracts.ProxyFactory)
}
