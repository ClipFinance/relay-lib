package evm

import (
	"context"
	"github.com/ClipFinance/relay-lib/chains/evm/generated"
	"github.com/ClipFinance/relay-lib/common/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
	"math/big"
	"strings"
)

// SendAsset sends an asset (native or token) based on the provided transaction intent.
//
// Parameters:
// - ctx: the context for managing the request.
// - intent: the transaction intent containing details of the asset transfer.
//
// Returns:
// - *types.Transaction: the transaction details.
// - error: an error if the client is not initialized or if the transaction fails.
func (e *evm) SendAsset(ctx context.Context, intent *types.Intent) (*types.Transaction, error) {
	e.clientMutex.RLock()
	client := e.client
	e.clientMutex.RUnlock()

	if client == nil {
		return nil, errors.New("client not initialized")
	}

	nonce, err := client.PendingNonceAt(ctx, e.signer.Address())
	if err != nil {
		return nil, errors.Wrap(err, "failed to get nonce")
	}

	var tx *ethtypes.Transaction
	if intent.ToToken == ZeroAddress {
		tx, err = e.sendNativeAsset(ctx, intent, nonce)
	} else {
		tx, err = e.sendToken(ctx, intent, nonce)
	}
	if err != nil {
		return nil, err
	}

	return &types.Transaction{
		Hash:       tx.Hash().Hex(),
		From:       e.signer.Address().Hex(),
		To:         intent.RecipientAddress,
		FromAmount: intent.FromAmount.String(),
		ToAmount:   intent.ToAmount.String(),
		Token:      intent.ToToken,
		Nonce:      nonce,
		ChainID:    e.config.ChainID,
		QuoteID:    intent.QuoteID,
	}, nil
}

// sendNativeAsset sends a native asset (Ether) based on the provided transaction intent.
//
// Parameters:
// - ctx: the context for managing the request.
// - intent: the transaction intent containing details of the asset transfer.
// - nonce: the nonce for the transaction.
//
// Returns:
// - *ethtypes.Transaction: the transaction details.
// - error: an error if the transaction preparation or sending fails.
func (e *evm) sendNativeAsset(ctx context.Context, intent *types.Intent, nonce uint64) (*ethtypes.Transaction, error) {
	tx, err := e.prepareTransaction(ctx, nonce, intent.RecipientAddress, intent.ToAmount, nil)
	if err != nil {
		return nil, err
	}

	return e.signAndSendTransaction(ctx, tx)
}

// sendToken sends a token based on the provided transaction intent.
//
// Parameters:
// - ctx: the context for managing the request.
// - intent: the transaction intent containing details of the asset transfer.
// - nonce: the nonce for the transaction.
//
// Returns:
// - *ethtypes.Transaction: the transaction details.
// - error: an error if the token ABI parsing, data packing, transaction preparation, or sending fails.
func (e *evm) sendToken(ctx context.Context, intent *types.Intent, nonce uint64) (*ethtypes.Transaction, error) {
	tokenAbi, err := abi.JSON(strings.NewReader(generated.ERC20ABI))
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse token ABI")
	}

	data, err := tokenAbi.Pack("transfer", common.HexToAddress(intent.RecipientAddress), intent.ToAmount)
	if err != nil {
		return nil, errors.Wrap(err, "failed to pack transfer data")
	}

	tx, err := e.prepareTransaction(ctx, nonce, intent.ToToken, big.NewInt(0), data)
	if err != nil {
		return nil, err
	}

	return e.signAndSendTransaction(ctx, tx)
}

// prepareTransaction prepares a transaction with the given parameters.
//
// Parameters:
// - ctx: the context for managing the request.
// - nonce: the nonce for the transaction.
// - toAddress: the recipient address of the transaction.
// - value: the amount of Ether to send with the transaction.
// - data: the input data for the transaction.
//
// Returns:
// - *ethtypes.Transaction: the prepared transaction.
// - error: an error if the gas estimation, gas price retrieval, or client initialization fails.
func (e *evm) prepareTransaction(ctx context.Context, nonce uint64, toAddress string, value *big.Int, data []byte) (*ethtypes.Transaction, error) {
	estimatedGas, err := e.EstimateGas(ctx, toAddress, value, data)
	if err != nil {
		e.logger.WithField("chain", e.config.Name).WithError(err).Warn("Failed to estimate gas")
		return nil, errors.Wrap(err, "failed to estimate gas")
	}

	gasLimit := uint64(float64(estimatedGas) * 1.1)

	to := common.HexToAddress(toAddress)

	e.clientMutex.RLock()
	client := e.client
	e.clientMutex.RUnlock()

	if client == nil {
		return nil, errors.New("client not initialized")
	}

	if e.config.TxType == TxTypeEIP1559 {
		gasPriceData, err := e.getEIP1559GasPrice(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get EIP-1559 gas price")
		}

		return ethtypes.NewTx(&ethtypes.DynamicFeeTx{
			ChainID:    big.NewInt(0).SetUint64(e.config.ChainID),
			Nonce:      nonce,
			GasFeeCap:  gasPriceData.MaxFeePerGas,
			GasTipCap:  gasPriceData.MaxPriorityFeePerGas,
			Gas:        gasLimit,
			To:         &to,
			Value:      value,
			Data:       data,
			AccessList: nil,
		}), nil
	}

	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get gas price")
	}

	gasPrice = new(big.Int).Mul(gasPrice, big.NewInt(150))
	gasPrice = new(big.Int).Div(gasPrice, big.NewInt(100))

	return ethtypes.NewTransaction(
		nonce,
		to,
		value,
		gasLimit,
		gasPrice,
		data,
	), nil
}

// signAndSendTransaction signs and sends the prepared transaction.
//
// Parameters:
// - ctx: the context for managing the request.
// - tx: the prepared transaction to be signed and sent.
//
// Returns:
// - *ethtypes.Transaction: the signed and sent transaction.
// - error: an error if the client or signer is not initialized, or if the signing or sending fails.
func (e *evm) signAndSendTransaction(ctx context.Context, tx *ethtypes.Transaction) (*ethtypes.Transaction, error) {
	e.clientMutex.RLock()
	client := e.client
	e.clientMutex.RUnlock()

	e.signerMutex.RLock()
	signer := e.signer
	e.signerMutex.RUnlock()

	if client == nil || signer == nil {
		return nil, errors.New("client or signer not initialized")
	}

	chainID := big.NewInt(0).SetUint64(e.config.ChainID)

	signedTx, err := signer.SignTx(tx, chainID)
	if err != nil {
		e.logger.WithError(err).Error("Failed to sign transaction")
		return nil, errors.Wrap(err, "failed to sign transaction")
	}

	if err = client.SendTransaction(ctx, signedTx); err != nil {
		e.logger.WithError(err).Error("Failed to send transaction")
		return nil, errors.Wrap(err, "failed to send transaction")
	}

	return signedTx, nil
}
