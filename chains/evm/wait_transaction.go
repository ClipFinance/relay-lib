package evm

import (
	"context"
	"github.com/ClipFinance/relay-lib/common/types"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"math/big"
	"sync"
	"time"
)

const ( // minProfitPercentage defines minimum profit percentage for transaction to be considered profitable.
	minProfitPercentage = 1
	// gasIncreaseFactor defines minimum gas increase percentage for replacement transaction to be considered profitable.
	gasIncreaseFactor = 110 // 110%
)

// subscriptionHandler manages block header subscriptions
type subscriptionHandler struct {
	subscription ethereum.Subscription
	headerChan   chan *ethtypes.Header
	sync.RWMutex
}

// close safely closes subscription and channel
func (h *subscriptionHandler) close() {
	h.Lock()
	defer h.Unlock()
	if h.subscription != nil {
		h.subscription.Unsubscribe()
		h.subscription = nil
	}
	if h.headerChan != nil {
		close(h.headerChan)
		h.headerChan = nil
	}
}

// WaitTransactionConfirmation waits for the confirmation of a transaction.
//
// Parameters:
// - ctx: the context for managing the request.
// - tx: the transaction to wait for confirmation.
//
// Returns:
// - bool: true if the transaction is confirmed successfully, false otherwise.
// - error: an error if the client is not initialized, if there is an issue getting the block number, or if the transaction receipt retrieval fails.
func (e *evm) WaitTransactionConfirmation(ctx context.Context, tx *types.Transaction) (types.TransactionStatus, error) {
	e.clientMutex.RLock()
	client := e.client
	e.clientMutex.RUnlock()

	if client == nil {
		return types.TxNeedsRetry, errors.New("client not initialized")
	}

	start := time.Now()
	blockNumber, err := client.BlockNumber(ctx)
	if err != nil {
		return types.TxNeedsRetry, errors.Wrap(err, "failed to get current block number")
	}

	// Use subscription based on RPC URL type
	if types.GetSubscriptionMode(e.config.RpcUrl) == types.WebSocketMode {
		return e.waitTransactionConfirmationWS(ctx, tx, blockNumber, start)
	}
	return e.waitTransactionConfirmationHTTP(ctx, tx, blockNumber, start)
}

// waitTransactionConfirmationWS waits for transaction confirmation using WebSocket subscription
func (e *evm) waitTransactionConfirmationWS(ctx context.Context, tx *types.Transaction, startBlock uint64, startTime time.Time) (types.TransactionStatus, error) {
	e.clientMutex.RLock()
	client := e.client
	e.clientMutex.RUnlock()

	handler := &subscriptionHandler{
		headerChan: make(chan *ethtypes.Header),
	}
	defer handler.close()

	// Subscribe to new block headers
	sub, err := client.SubscribeNewHead(ctx, handler.headerChan)
	if err != nil {
		return types.TxNeedsRetry, errors.Wrap(err, "failed to subscribe to new headers")
	}

	handler.Lock()
	handler.subscription = sub
	handler.Unlock()

	for {
		select {
		case <-ctx.Done():
			e.logger.WithField("txHash", tx.Hash).Error("WaitTransactionConfirmation: context done")
			return types.TxFailed, ctx.Err()

		case err := <-sub.Err():
			return types.TxNeedsRetry, errors.Wrap(err, "subscription error")

		case header := <-handler.headerChan:
			if header == nil {
				continue
			}

			// Check for stuck transaction
			if time.Since(startTime) > waitTimeout {
				currentBlock := header.Number.Uint64()
				if currentBlock > startBlock+2 {
					status, err := e.handleStuckTransaction(ctx, tx)
					if err != nil {
						return status, err
					}
					// Reset timer and block number for new transaction
					startTime = time.Now()
					startBlock = currentBlock
					continue
				}
			}

			// Check transaction receipt
			receipt, err := client.TransactionReceipt(ctx, common.HexToHash(tx.Hash))
			if err != nil {
				if errors.Is(err, ethereum.NotFound) {
					continue
				}
				return types.TxFailed, errors.Wrap(err, "failed to get transaction receipt")
			}

			// Wait for required block confirmations
			if header.Number.Uint64() < receipt.BlockNumber.Uint64()+e.config.WaitNBlocks {
				continue
			}

			if receipt.Status == ethtypes.ReceiptStatusSuccessful {
				return types.TxDone, nil
			}
			return types.TxFailed, nil
		}
	}
}

// waitTransactionConfirmationHTTP waits for transaction confirmation using HTTP polling
func (e *evm) waitTransactionConfirmationHTTP(ctx context.Context, tx *types.Transaction, startBlock uint64, startTime time.Time) (types.TransactionStatus, error) {
	e.clientMutex.RLock()
	client := e.client
	e.clientMutex.RUnlock()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			e.logger.WithField("txHash", tx.Hash).Error("WaitTransactionConfirmation: context done")
			return types.TxFailed, ctx.Err()

		case <-ticker.C:
			// Check for stuck transaction
			if time.Since(startTime) > waitTimeout {
				currentBlock, err := client.BlockNumber(ctx)
				if err != nil {
					return types.TxFailed, errors.Wrap(err, "failed to get current block number")
				}

				if currentBlock > startBlock+2 {
					status, err := e.handleStuckTransaction(ctx, tx)
					if err != nil {
						return status, err
					}
					// Reset timer and block number for new transaction
					startTime = time.Now()
					startBlock = currentBlock
					continue
				}
			}

			receipt, err := client.TransactionReceipt(ctx, common.HexToHash(tx.Hash))
			if err != nil {
				if errors.Is(err, ethereum.NotFound) {
					continue
				}
				return types.TxFailed, errors.Wrap(err, "failed to get transaction receipt")
			}

			currentBlock, err := client.BlockNumber(ctx)
			if err != nil {
				return types.TxFailed, errors.Wrap(err, "failed to get current block number")
			}

			if currentBlock < receipt.BlockNumber.Uint64()+e.config.WaitNBlocks {
				continue
			}

			if receipt.Status == ethtypes.ReceiptStatusSuccessful {
				return types.TxDone, nil
			}
			return types.TxFailed, nil
		}
	}
}

// handleStuckTransaction handles stuck transaction by attempting to replace or cancel it
func (e *evm) handleStuckTransaction(ctx context.Context, tx *types.Transaction) (types.TransactionStatus, error) {
	newTx, err := e.replaceTransaction(ctx, tx)
	if err != nil {
		if cancelTx, err := e.cancelTransaction(ctx, tx); err == nil {
			e.logger.WithFields(logrus.Fields{
				"originalTx": tx.Hash,
				"cancelTx":   cancelTx.Hash(),
			}).Info("Transaction cancelled successfully")
			return types.TxFailed, errors.New("transaction cancelled due to timeout")
		}
		return types.TxFailed, errors.New("failed to cancel stuck transaction")
	}

	// Update transaction details with new transaction
	tx.Hash = newTx.Hash().Hex()
	tx.From = e.signer.Address().Hex()
	return types.TxNeedsRetry, nil
}

// replaceTransaction replaces a pending transaction with a new one with a higher gas price.
//
// Parameters:
// - ctx: the context for managing the request.
// - tx: the transaction to be replaced.
//
// Returns:
// - *ethtypes.Transaction: the new transaction details.
// - error: an error if the client is not initialized, if the transaction retrieval fails, or if the transaction is not pending.
func (e *evm) replaceTransaction(ctx context.Context, tx *types.Transaction) (*ethtypes.Transaction, error) {
	e.clientMutex.RLock()
	client := e.client
	e.clientMutex.RUnlock()

	if client == nil {
		return nil, errors.New("client not initialized")
	}

	txHash := common.HexToHash(tx.Hash)

	oldTx, isPending, err := client.TransactionByHash(ctx, txHash)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get transaction by hash")
	}
	if !isPending {
		e.logger.WithFields(logrus.Fields{
			"txHash": tx.Hash,
			"chain":  e.config.Name,
		}).Warn("transaction is not pending")
		return nil, nil
	}

	// Get optimal gas price for replacement
	newGasPrice, err := e.getNewGasPrice(ctx, oldTx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to calculate new gas price")
	}

	oldGas := oldTx.Gas()

	// Check if transaction remains profitable with new gas price
	if !e.calculateTransactionProfitability(tx, new(big.Int).SetUint64(oldGas), newGasPrice) {
		if cancelTx, err := e.cancelTransaction(ctx, tx); err == nil {
			e.logger.WithFields(logrus.Fields{
				"originalTx": tx.Hash,
				"cancelTx":   cancelTx.Hash(),
			}).Info("Transaction cancelled due to unprofitability")
			return nil, nil
		}
	}

	var newTx *ethtypes.Transaction

	if e.config.TxType == TxTypeEIP1559 {
		newTx = ethtypes.NewTx(&ethtypes.DynamicFeeTx{
			ChainID:   oldTx.ChainId(),
			Nonce:     oldTx.Nonce(),
			GasTipCap: oldTx.GasTipCap(),
			GasFeeCap: newGasPrice,
			Gas:       oldTx.Gas(),
			To:        oldTx.To(),
			Value:     oldTx.Value(),
			Data:      oldTx.Data(),
		})
	} else {
		newTx = ethtypes.NewTransaction(
			oldTx.Nonce(),
			*oldTx.To(),
			oldTx.Value(),
			oldTx.Gas(),
			newGasPrice,
			oldTx.Data(),
		)
	}

	return e.signAndSendTransaction(ctx, newTx)
}

// cancelTransaction cancels a pending transaction by sending a new transaction with the same nonce and higher gas price.
//
// Parameters:
// - ctx: the context for managing the request.
// - tx: the transaction to be cancelled.
//
// Returns:
// - *ethtypes.Transaction: the new transaction details.
// - error: an error if the client is not initialized, if the transaction retrieval fails, or if the transaction is not pending.
func (e *evm) cancelTransaction(ctx context.Context, tx *types.Transaction) (*ethtypes.Transaction, error) {
	e.clientMutex.RLock()
	client := e.client
	e.clientMutex.RUnlock()

	if client == nil {
		return nil, errors.New("client not initialized")
	}

	txHash := common.HexToHash(tx.Hash)
	transaction, pending, err := client.TransactionByHash(ctx, txHash)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get transaction by hash")
	}
	if !pending {
		e.logger.WithFields(logrus.Fields{
			"txHash": tx.Hash,
			"chain":  e.config.Name,
		}).Warn("transaction is not pending")
		return nil, nil
	}

	gasPrice := transaction.GasPrice()
	gasPrice = new(big.Int).Mul(gasPrice, big.NewInt(150))
	gasPrice = new(big.Int).Div(gasPrice, big.NewInt(100))

	chainID := new(big.Int).SetUint64(e.config.ChainID)

	e.signerMutex.RLock()
	toAddress := e.signer.Address()
	e.signerMutex.RUnlock()

	var newTx *ethtypes.Transaction

	if e.config.TxType == TxTypeEIP1559 {
		newTx = ethtypes.NewTx(&ethtypes.DynamicFeeTx{
			ChainID:   chainID,
			Nonce:     transaction.Nonce(),
			GasTipCap: transaction.GasTipCap(),
			GasFeeCap: gasPrice,
			Gas:       21000,
			To:        &toAddress,
			Value:     big.NewInt(0),
			Data:      nil,
		})
	} else {
		newTx = ethtypes.NewTransaction(
			transaction.Nonce(),
			toAddress,
			big.NewInt(0),
			21000,
			gasPrice,
			nil,
		)
	}

	return e.signAndSendTransaction(ctx, newTx)
}

// getNewGasPrice calculates optimal gas price for replacement transaction
func (e *evm) getNewGasPrice(ctx context.Context, oldTx *ethtypes.Transaction) (*big.Int, error) {
	e.clientMutex.RLock()
	client := e.client
	e.clientMutex.RUnlock()

	var currentGasPrice *big.Int
	var err error

	if e.config.TxType == TxTypeEIP1559 {
		gasPriceData, err := e.getEIP1559GasPrice(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get EIP-1559 gas price")
		}
		currentGasPrice = gasPriceData.MaxFeePerGas
	} else {
		currentGasPrice, err = client.SuggestGasPrice(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get current gas price")
		}
	}

	oldGasPrice := oldTx.GasPrice()

	// Calculate minimum required gas price (110% of old gas price)
	minGasPrice := new(big.Int).Div(
		new(big.Int).Mul(oldGasPrice, big.NewInt(gasIncreaseFactor)),
		big.NewInt(100),
	)

	// If current gas price is higher than minimum required, use it
	if currentGasPrice.Cmp(minGasPrice) > 0 {
		return currentGasPrice, nil
	}

	// Otherwise use minimum required gas price
	return minGasPrice, nil
}
