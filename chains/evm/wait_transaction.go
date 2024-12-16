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
	"time"
)

// WaitTransactionConfirmation waits for the confirmation of a transaction.
//
// Parameters:
// - ctx: the context for managing the request.
// - tx: the transaction to wait for confirmation.
//
// Returns:
// - bool: true if the transaction is confirmed successfully, false otherwise.
// - error: an error if the client is not initialized, if there is an issue getting the block number, or if the transaction receipt retrieval fails.
func (e *evm) WaitTransactionConfirmation(ctx context.Context, tx *types.Transaction) (bool, error) {
	e.clientMutex.RLock()
	client := e.client
	e.clientMutex.RUnlock()

	if client == nil {
		return false, errors.New("client not initialized")
	}

	start := time.Now()
	blockNumber, err := client.BlockNumber(ctx)
	if err != nil {
		return false, errors.Wrap(err, "failed to get current block number")
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			e.logger.WithField("txHash", tx.Hash).Error("Context done waiting for transaction receipt")
			return false, ctx.Err()

		case <-ticker.C:
			if time.Since(start) > waitTimeout {
				currentBlock, err := client.BlockNumber(ctx)
				if err != nil {
					return false, errors.Wrap(err, "failed to get current block number")
				}
				// TODO: use SubscribeNewHead

				if currentBlock > blockNumber+2 {
					newTx, err := e.replaceTransaction(ctx, tx)
					if err != nil {
						if cancelTx, err := e.cancelTransaction(ctx, tx); err == nil {
							e.logger.WithFields(logrus.Fields{
								"originalTx": tx.Hash,
								"cancelTx":   cancelTx.Hash,
							}).Info("Transaction cancelled successfully")
							return false, errors.New("transaction cancelled due to timeout")
						}
						return false, errors.New("failed to cancel stuck transaction")
					}
					tx = &types.Transaction{
						Hash:       newTx.Hash().Hex(),
						From:       e.signer.Address().Hex(),
						To:         tx.To,
						FromAmount: tx.FromAmount,
						ToAmount:   tx.ToAmount,
						Token:      tx.Token,
						Nonce:      tx.Nonce,
						ChainID:    e.config.ChainID,
						QuoteID:    tx.QuoteID,
					}
					start = time.Now()
					continue
				}
			}

			receipt, err := client.TransactionReceipt(ctx, common.HexToHash(tx.Hash))
			if err != nil {
				if errors.Is(err, ethereum.NotFound) {
					continue
				}
				return false, errors.Wrap(err, "failed to get transaction receipt")
			}

			currentBlock, err := client.BlockNumber(ctx)
			if err != nil {
				return false, errors.Wrap(err, "failed to get current block number")
			}

			if currentBlock < receipt.BlockNumber.Uint64()+e.config.WaitNBlocks {
				continue
			}

			return receipt.Status == ethtypes.ReceiptStatusSuccessful, nil
		}
	}
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

	newGasPrice := new(big.Int).Mul(oldTx.GasPrice(), big.NewInt(110))
	newGasPrice = newGasPrice.Div(newGasPrice, big.NewInt(100))

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
