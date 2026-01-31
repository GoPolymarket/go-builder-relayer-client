package relayer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/GoPolymarket/go-builder-relayer-client/internal/builder"
	"github.com/GoPolymarket/go-builder-relayer-client/internal/encoder"
	"github.com/GoPolymarket/go-builder-relayer-client/internal/utils"
	"github.com/GoPolymarket/go-builder-relayer-client/pkg/signer"
	"github.com/GoPolymarket/go-builder-relayer-client/pkg/types"
)

// RelayClient provides access to Polymarket relayer endpoints.
type RelayClient struct {
	relayerURL     string
	chainID        int64
	relayTxType    types.RelayerTxType
	contractConfig types.ContractConfig
	httpClient     *HTTPClient
	signer         signer.Signer
	builderConfig  *BuilderConfig
}

func NewRelayClient(relayerURL string, chainID int64, signer signer.Signer, builderConfig *BuilderConfig, relayTxType types.RelayerTxType) (*RelayClient, error) {
	cleanURL := strings.TrimRight(relayerURL, "/")
	if relayTxType == "" {
		relayTxType = types.RelayerTxSafe
	}
	config, err := GetContractConfig(chainID)
	if err != nil {
		return nil, err
	}

	return &RelayClient{
		relayerURL:     cleanURL,
		chainID:        chainID,
		relayTxType:    relayTxType,
		contractConfig: config,
		httpClient:     NewHTTPClient(nil),
		signer:         signer,
		builderConfig:  builderConfig,
	}, nil
}

// SetHTTPClient allows overriding the underlying HTTP client.
func (c *RelayClient) SetHTTPClient(client *HTTPClient) {
	if client != nil {
		c.httpClient = client
	}
}

func (c *RelayClient) GetNonce(ctx context.Context, signerAddress string, signerType string) (types.NoncePayload, error) {
	var resp types.NoncePayload
	err := c.send(ctx, GetNonceEndpoint, "GET", &RequestOptions{Params: map[string]string{"address": signerAddress, "type": signerType}}, &resp)
	return resp, err
}

func (c *RelayClient) GetRelayPayload(ctx context.Context, signerAddress string, signerType string) (types.RelayPayload, error) {
	var resp types.RelayPayload
	err := c.send(ctx, GetRelayPayloadEndpoint, "GET", &RequestOptions{Params: map[string]string{"address": signerAddress, "type": signerType}}, &resp)
	return resp, err
}

func (c *RelayClient) GetTransaction(ctx context.Context, transactionID string) ([]types.RelayerTransaction, error) {
	var resp []types.RelayerTransaction
	err := c.send(ctx, GetTransactionEndpoint, "GET", &RequestOptions{Params: map[string]string{"id": transactionID}}, &resp)
	return resp, err
}

func (c *RelayClient) GetTransactions(ctx context.Context) ([]types.RelayerTransaction, error) {
	var resp []types.RelayerTransaction
	err := c.sendAuthedRequest(ctx, "GET", GetTransactionsEndpoint, "", &resp)
	return resp, err
}

func (c *RelayClient) GetDeployed(ctx context.Context, safeAddress string) (bool, error) {
	var resp types.GetDeployedResponse
	err := c.send(ctx, GetDeployedEndpoint, "GET", &RequestOptions{Params: map[string]string{"address": safeAddress}}, &resp)
	return resp.Deployed, err
}

// Execute executes a batch of transactions.
func (c *RelayClient) Execute(ctx context.Context, txns []types.Transaction, metadata string) (*ClientRelayerTransactionResponse, error) {
	if c.signer == nil {
		return nil, types.ErrSignerUnavailable
	}
	if len(txns) == 0 {
		return nil, fmt.Errorf("no transactions to execute")
	}

	switch c.relayTxType {
	case types.RelayerTxSafe:
		safeTxns := make([]types.SafeTransaction, 0, len(txns))
		for _, tx := range txns {
			value := tx.Value
			if value == "" {
				value = "0"
			}
			safeTxns = append(safeTxns, types.SafeTransaction{To: tx.To, Operation: types.OperationCall, Data: tx.Data, Value: value})
		}
		return c.executeSafeTransactions(ctx, safeTxns, metadata)
	case types.RelayerTxProxy:
		proxyTxns := make([]types.ProxyTransaction, 0, len(txns))
		for _, tx := range txns {
			value := tx.Value
			if value == "" {
				value = "0"
			}
			proxyTxns = append(proxyTxns, types.ProxyTransaction{To: tx.To, TypeCode: types.CallTypeCall, Data: tx.Data, Value: value})
		}
		return c.executeProxyTransactions(ctx, proxyTxns, metadata)
	default:
		return nil, fmt.Errorf("unsupported relay transaction type: %s", c.relayTxType)
	}
}

func (c *RelayClient) executeProxyTransactions(ctx context.Context, txns []types.ProxyTransaction, metadata string) (*ClientRelayerTransactionResponse, error) {
	if c.signer == nil {
		return nil, types.ErrSignerUnavailable
	}
	if !IsProxyContractConfigValid(c.contractConfig.ProxyContracts) {
		return nil, types.ErrConfigUnsupported
	}
	from := c.signer.Address().Hex()
	relayPayload, err := c.GetRelayPayload(ctx, from, string(types.TransactionTypeProxy))
	if err != nil {
		return nil, err
	}
	data, err := encoder.EncodeProxyTransactionData(txns)
	if err != nil {
		return nil, err
	}

	args := types.ProxyTransactionArgs{
		From:     from,
		GasPrice: "0",
		Data:     data,
		Relay:    relayPayload.Address,
		Nonce:    relayPayload.Nonce,
	}

	request, err := builder.BuildProxyTransactionRequest(c.signer, args, c.contractConfig.ProxyContracts, metadata)
	if err != nil {
		return nil, err
	}

	payload, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}

	var resp types.RelayerTransactionResponse
	if err := c.sendAuthedRequest(ctx, "POST", SubmitTransactionEndpoint, string(payload), &resp); err != nil {
		return nil, err
	}
	return &ClientRelayerTransactionResponse{
		TransactionID:   resp.TransactionID,
		State:           resp.State,
		TransactionHash: resp.TransactionHash,
		client:          c,
	}, nil
}

func (c *RelayClient) executeSafeTransactions(ctx context.Context, txns []types.SafeTransaction, metadata string) (*ClientRelayerTransactionResponse, error) {
	if c.signer == nil {
		return nil, types.ErrSignerUnavailable
	}
	if !IsSafeContractConfigValid(c.contractConfig.SafeContracts) {
		return nil, types.ErrConfigUnsupported
	}
	safe, err := c.getExpectedSafe()
	if err != nil {
		return nil, err
	}
	deployed, err := c.GetDeployed(ctx, safe)
	if err != nil {
		return nil, err
	}
	if !deployed {
		return nil, types.ErrSafeNotDeployed
	}

	from := c.signer.Address().Hex()
	noncePayload, err := c.GetNonce(ctx, from, string(types.TransactionTypeSafe))
	if err != nil {
		return nil, err
	}
	if noncePayload.Nonce == "" {
		return nil, fmt.Errorf("invalid nonce payload received")
	}

	args := types.SafeTransactionArgs{
		From:         from,
		Nonce:        noncePayload.Nonce,
		ChainID:      c.chainID,
		Transactions: txns,
	}
	request, err := builder.BuildSafeTransactionRequest(c.signer, args, c.contractConfig.SafeContracts, metadata)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}

	var resp types.RelayerTransactionResponse
	if err := c.sendAuthedRequest(ctx, "POST", SubmitTransactionEndpoint, string(payload), &resp); err != nil {
		return nil, err
	}
	return &ClientRelayerTransactionResponse{
		TransactionID:   resp.TransactionID,
		State:           resp.State,
		TransactionHash: resp.TransactionHash,
		client:          c,
	}, nil
}

// Deploy deploys a Safe contract.
func (c *RelayClient) Deploy(ctx context.Context) (*ClientRelayerTransactionResponse, error) {
	if c.signer == nil {
		return nil, types.ErrSignerUnavailable
	}
	safe, err := c.getExpectedSafe()
	if err != nil {
		return nil, err
	}
	deployed, err := c.GetDeployed(ctx, safe)
	if err != nil {
		return nil, err
	}
	if deployed {
		return nil, types.ErrSafeDeployed
	}
	return c.deploySafe(ctx)
}

func (c *RelayClient) deploySafe(ctx context.Context) (*ClientRelayerTransactionResponse, error) {
	if !IsSafeContractConfigValid(c.contractConfig.SafeContracts) {
		return nil, types.ErrConfigUnsupported
	}
	from := c.signer.Address().Hex()
	args := types.SafeCreateTransactionArgs{
		From:            from,
		ChainID:         c.chainID,
		PaymentToken:    types.ZeroAddress,
		Payment:         "0",
		PaymentReceiver: types.ZeroAddress,
	}
	request, err := builder.BuildSafeCreateTransactionRequest(c.signer, c.contractConfig.SafeContracts, args)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}

	var resp types.RelayerTransactionResponse
	if err := c.sendAuthedRequest(ctx, "POST", SubmitTransactionEndpoint, string(payload), &resp); err != nil {
		return nil, err
	}
	return &ClientRelayerTransactionResponse{
		TransactionID:   resp.TransactionID,
		State:           resp.State,
		TransactionHash: resp.TransactionHash,
		client:          c,
	}, nil
}

func (c *RelayClient) PollUntilState(ctx context.Context, transactionID string, states []types.RelayerTransactionState, failState types.RelayerTransactionState, maxPolls int, pollFrequency time.Duration) (*types.RelayerTransaction, error) {
	stateSet := map[string]bool{}
	for _, s := range states {
		stateSet[string(s)] = true
	}

	if maxPolls <= 0 {
		maxPolls = 10
	}
	if pollFrequency < time.Second {
		pollFrequency = 2 * time.Second
	}

	for i := 0; i < maxPolls; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		txns, err := c.GetTransaction(ctx, transactionID)
		if err != nil {
			return nil, err
		}
		if len(txns) > 0 {
			txn := txns[0]
			if stateSet[txn.State] {
				return &txn, nil
			}
			if failState != "" && txn.State == string(failState) {
				return nil, fmt.Errorf("transaction failed onchain: %s", txn.TransactionHash)
			}
		}
		utils.Sleep(pollFrequency)
	}
	return nil, fmt.Errorf("transaction not found or not in desired state (timeout)")
}

func (c *RelayClient) send(ctx context.Context, path string, method string, options *RequestOptions, out interface{}) error {
	url := c.relayerURL + path
	return c.httpClient.Do(ctx, method, url, options, out)
}

func (c *RelayClient) sendAuthedRequest(ctx context.Context, method, path string, body string, out interface{}) error {
	headers := http.Header{}
	if c.builderConfig != nil && c.builderConfig.IsValid() {
		signBody := body
		headersToAdd, err := c.builderConfig.Headers(ctx, method, path, &signBody, 0)
		if err != nil {
			return err
		}
		for k, vals := range headersToAdd {
			for _, v := range vals {
				headers.Add(k, v)
			}
		}
	}
	opts := &RequestOptions{Headers: headers}
	if body != "" {
		opts.Body = []byte(body)
	}
	return c.send(ctx, path, method, opts, out)
}

func (c *RelayClient) getExpectedSafe() (string, error) {
	if c.signer == nil {
		return "", types.ErrSignerUnavailable
	}
	return builder.DeriveSafeAddress(c.signer.Address().Hex(), c.contractConfig.SafeContracts.SafeFactory)
}