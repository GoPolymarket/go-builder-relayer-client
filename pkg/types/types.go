package types

import "time"

type RelayerTxType string

const (
	RelayerTxSafe  RelayerTxType = "SAFE"
	RelayerTxProxy RelayerTxType = "PROXY"
)

type TransactionType string

const (
	TransactionTypeSafe       TransactionType = "SAFE"
	TransactionTypeProxy      TransactionType = "PROXY"
	TransactionTypeSafeCreate TransactionType = "SAFE-CREATE"
)

type SignatureParams struct {
	GasPrice string `json:"gasPrice,omitempty"`

	// Proxy RelayHub sig params
	RelayerFee string `json:"relayerFee,omitempty"`
	GasLimit   string `json:"gasLimit,omitempty"`
	RelayHub   string `json:"relayHub,omitempty"`
	Relay      string `json:"relay,omitempty"`

	// SAFE sig params
	Operation      string `json:"operation,omitempty"`
	SafeTxnGas     string `json:"safeTxnGas,omitempty"`
	BaseGas        string `json:"baseGas,omitempty"`
	GasToken       string `json:"gasToken,omitempty"`
	RefundReceiver string `json:"refundReceiver,omitempty"`

	// SAFE CREATE sig params
	PaymentToken    string `json:"paymentToken,omitempty"`
	Payment         string `json:"payment,omitempty"`
	PaymentReceiver string `json:"paymentReceiver,omitempty"`
}

type AddressPayload struct {
	Address string `json:"address"`
}

type NoncePayload struct {
	Nonce string `json:"nonce"`
}

type RelayPayload struct {
	Address string `json:"address"`
	Nonce   string `json:"nonce"`
}

type TransactionRequest struct {
	Type            string          `json:"type"`
	From            string          `json:"from"`
	To              string          `json:"to"`
	ProxyWallet     string          `json:"proxyWallet,omitempty"`
	Data            string          `json:"data"`
	Nonce           string          `json:"nonce,omitempty"`
	Signature       string          `json:"signature"`
	SignatureParams SignatureParams `json:"signatureParams"`
	Metadata        string          `json:"metadata,omitempty"`
}

type CallType uint8

const (
	CallTypeInvalid      CallType = 0
	CallTypeCall         CallType = 1
	CallTypeDelegateCall CallType = 2
)

type ProxyTransaction struct {
	To       string   `json:"to"`
	TypeCode CallType `json:"typeCode"`
	Data     string   `json:"data"`
	Value    string   `json:"value"`
}

// Safe Transactions

type OperationType uint8

const (
	OperationCall         OperationType = 0
	OperationDelegateCall OperationType = 1
)

type SafeTransaction struct {
	To        string        `json:"to"`
	Operation OperationType `json:"operation"`
	Data      string        `json:"data"`
	Value     string        `json:"value"`
}

type Transaction struct {
	To    string `json:"to"`
	Data  string `json:"data"`
	Value string `json:"value"`
}

type SafeTransactionArgs struct {
	From         string
	Nonce        string
	ChainID      int64
	Transactions []SafeTransaction
}

type SafeCreateTransactionArgs struct {
	From            string
	ChainID         int64
	PaymentToken    string
	Payment         string
	PaymentReceiver string
}

type ProxyTransactionArgs struct {
	From     string
	Nonce    string
	GasPrice string
	GasLimit string
	Data     string
	Relay    string
}

type RelayerTransactionState string

const (
	StateNew       RelayerTransactionState = "STATE_NEW"
	StateExecuted  RelayerTransactionState = "STATE_EXECUTED"
	StateMined     RelayerTransactionState = "STATE_MINED"
	StateInvalid   RelayerTransactionState = "STATE_INVALID"
	StateConfirmed RelayerTransactionState = "STATE_CONFIRMED"
	StateFailed    RelayerTransactionState = "STATE_FAILED"
)

type RelayerTransaction struct {
	TransactionID   string    `json:"transactionID"`
	TransactionHash string    `json:"transactionHash"`
	From            string    `json:"from"`
	To              string    `json:"to"`
	ProxyAddress    string    `json:"proxyAddress"`
	Data            string    `json:"data"`
	Nonce           string    `json:"nonce"`
	Value           string    `json:"value"`
	State           string    `json:"state"`
	Type            string    `json:"type"`
	Metadata        string    `json:"metadata"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type RelayerTransactionResponse struct {
	TransactionID   string `json:"transactionID"`
	State           string `json:"state"`
	Hash            string `json:"hash"`
	TransactionHash string `json:"transactionHash"`
}

type GetDeployedResponse struct {
	Deployed bool `json:"deployed"`
}

type ProxyContractConfig struct {
	RelayHub     string
	ProxyFactory string
}

type SafeContractConfig struct {
	SafeFactory   string
	SafeMultisend string
}

type ContractConfig struct {
	ProxyContracts ProxyContractConfig
	SafeContracts  SafeContractConfig
}