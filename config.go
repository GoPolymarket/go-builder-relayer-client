package relayer

import (
	"github.com/GoPolymarket/builder-relayer-go-client/pkg/types"
)

var amoyConfig = types.ContractConfig{
	ProxyContracts: types.ProxyContractConfig{
		RelayHub:     "",
		ProxyFactory: "",
	},
	SafeContracts: types.SafeContractConfig{
		SafeFactory:   "0xaacFeEa03eb1561C4e67d661e40682Bd20E3541b",
		SafeMultisend: "0xA238CBeb142c10Ef7Ad8442C6D1f9E89e07e7761",
	},
}

var polygonConfig = types.ContractConfig{
	ProxyContracts: types.ProxyContractConfig{
		ProxyFactory: "0xaB45c5A4B0c941a2F231C04C3f49182e1A254052",
		RelayHub:     "0xD216153c06E857cD7f72665E0aF1d7D82172F494",
	},
	SafeContracts: types.SafeContractConfig{
		SafeFactory:   "0xaacFeEa03eb1561C4e67d661e40682Bd20E3541b",
		SafeMultisend: "0xA238CBeb142c10Ef7Ad8442C6D1f9E89e07e7761",
	},
}

func IsProxyContractConfigValid(config types.ProxyContractConfig) bool {
	return config.RelayHub != "" && config.ProxyFactory != ""
}

func IsSafeContractConfigValid(config types.SafeContractConfig) bool {
	return config.SafeFactory != "" && config.SafeMultisend != ""
}

func GetContractConfig(chainID int64) (types.ContractConfig, error) {
	switch chainID {
	case 137:
		return polygonConfig, nil
	case 80002:
		return amoyConfig, nil
	default:
		return types.ContractConfig{}, types.ErrConfigUnsupported
	}
}