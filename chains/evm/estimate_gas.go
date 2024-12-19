package evm

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"math/big"
)

// GasPriceData represents the gas price data for EIP-1559 transactions.
type GasPriceData struct {
	MaxFeePerGas         *big.Int // The maximum fee per gas.
	MaxPriorityFeePerGas *big.Int // The maximum priority fee per gas.
	IsEIP1559            bool     // Indicates if the transaction is EIP-1559.
}

// EstimateGas estimates the gas required for a transaction.
//
// Parameters:
// - ctx: the context for managing the request.
// - toAddress: the recipient address of the transaction.
// - value: the amount of Ether to send with the transaction.
// - data: the input data for the transaction.
//
// Returns:
// - uint64: the estimated gas required for the transaction.
// - error: an error if the client or signer is not initialized or if the gas estimation fails.
func (e *evm) EstimateGas(ctx context.Context, toAddress string, value *big.Int, data []byte) (uint64, error) {
	e.clientMutex.RLock()
	client := e.client
	e.clientMutex.RUnlock()

	e.signerMutex.RLock()
	signer := e.signer
	e.signerMutex.RUnlock()

	if client == nil || signer == nil {
		return 0, errors.New("client or signer not initialized")
	}

	to := common.HexToAddress(toAddress)
	msg := ethereum.CallMsg{
		From:     signer.Address(),
		To:       &to,
		Value:    value,
		GasPrice: nil,
		Data:     data,
	}

	return client.EstimateGas(ctx, msg)
}

// getEIP1559GasPrice retrieves the gas price data for EIP-1559 transactions.
//
// Parameters:
// - ctx: the context for managing the request.
//
// Returns:
// - *GasPriceData: the gas price data for EIP-1559 transactions.
// - error: an error if the client is not initialized or if there is an issue retrieving the gas price data.
func (e *evm) getEIP1559GasPrice(ctx context.Context) (*GasPriceData, error) {
	e.clientMutex.RLock()
	client := e.client
	e.clientMutex.RUnlock()

	if client == nil {
		return nil, errors.New("client not initialized")
	}

	suggestedTip, err := client.SuggestGasTipCap(ctx)
	if err != nil {
		e.logger.WithError(err).Error("Failed to get suggested gas tip")
		suggestedTip = big.NewInt(1)
	}

	if suggestedTip.Cmp(big.NewInt(0)) == 0 {
		suggestedTip = big.NewInt(1)
	}

	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		e.logger.WithField("chain", e.config.Name).WithError(err).Warn("Failed to get header by number")
		return nil, errors.Wrap(err, "failed to get header by number")
	}

	baseFee := header.BaseFee
	if baseFee == nil {
		e.logger.WithField("chain", e.config.Name).Warn("Base fee is nil")
		return nil, errors.New("base fee is nil")
	}

	baseFeeBuf := new(big.Int).Mul(baseFee, big.NewInt(130))
	baseFeeBuf = baseFeeBuf.Div(baseFeeBuf, big.NewInt(100))
	maxFeePerGas := new(big.Int).Add(baseFeeBuf, suggestedTip)

	if maxFeePerGas.Cmp(suggestedTip) <= 0 {
		maxFeePerGas = new(big.Int).Add(suggestedTip, baseFee)
	}

	return &GasPriceData{
		MaxFeePerGas:         maxFeePerGas,
		MaxPriorityFeePerGas: suggestedTip,
		IsEIP1559:            true,
	}, nil
}

func (e *evm) estimateLegacyGasPrice(ctx context.Context, toAddress string, value *big.Int, data []byte) (*big.Int, error) {
	to := common.HexToAddress(toAddress)

	// TODO: refactor this using lineal_estimateGas to avoid if-else condition.
	if e.config.ChainID == 59144 {
		var gasEstimate map[string]string
		err := e.client.Client().CallContext(ctx, &gasEstimate, "linea_estimateGas", map[string]interface{}{
			"from":  e.signer.Address(),
			"to":    to.Hex(),
			"value": value,
			"data":  data,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to estimate gas using linea_estimateGas: %v", err)
		}

		baseFee, err := hexutil.DecodeUint64(gasEstimate["baseFeePerGas"])
		if err != nil {
			return nil, fmt.Errorf("failed to decode gas limit: %v", err)
		}

		baseFeeBI := new(big.Int).SetUint64(baseFee)

		return baseFeeBI, nil
	}

	return e.client.SuggestGasPrice(ctx)
}
