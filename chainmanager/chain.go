package chainmanager

import (
	"context"
	relaytypes "github.com/ClipFinance/relay-lib/common/types"
	"math/big"
	"sync"
)

// Chain implements types.Chain interface with thread-safe access to dependencies.
// It provides methods to interact with the chain's gas estimator, transaction sender, transaction watcher, and event handler.
// Each dependency is protected by a read-write mutex to ensure thread-safe access.
type Chain struct {
	config    *relaytypes.ChainConfig       // Chain configuration.
	estimator relaytypes.GasEstimator       // Gas estimator implementation.
	sender    relaytypes.TransactionSender  // Transaction sender implementation.
	watcher   relaytypes.TransactionWatcher // Transaction watcher implementation.
	handler   relaytypes.EventHandler       // Event handler implementation.

	// Mutexes for thread-safe access to dependencies.
	estimatorMutex sync.RWMutex // Mutex for gas estimator.
	senderMutex    sync.RWMutex // Mutex for transaction sender.
	watcherMutex   sync.RWMutex // Mutex for transaction watcher.
	handlerMutex   sync.RWMutex // Mutex for event handler.
}

// NewChain creates a new Chain instance.
//
// Parameters:
// - config: the chain configuration.
// - estimator: the gas estimator implementation.
// - sender: the transaction sender implementation.
// - watcher: the transaction watcher implementation.
// - handler: the event handler implementation.
//
// Returns:
// - *Chain: a new Chain instance.
func NewChain(
	config *relaytypes.ChainConfig,
	estimator relaytypes.GasEstimator,
	sender relaytypes.TransactionSender,
	watcher relaytypes.TransactionWatcher,
	handler relaytypes.EventHandler,
) *Chain {
	return &Chain{
		config:    config,
		estimator: estimator,
		sender:    sender,
		watcher:   watcher,
		handler:   handler,
	}
}

// InitWSSubscription initializes websocket subscription with thread-safe access.
// It locks the handler mutex for reading to ensure safe concurrent access to the handler.
// If the handler is not implemented, it returns an error.
//
// Parameters:
// - ctx: context for managing the lifecycle of the websocket subscription.
// - eventChan: channel to receive chain events.
//
// Returns:
// - error: an error if the handler is not implemented or if any issue occurs during initialization.
func (c *Chain) InitWSSubscription(ctx context.Context, eventChan chan relaytypes.ChainEvent) error {
	c.handlerMutex.RLock()
	defer c.handlerMutex.RUnlock()

	if c.handler == nil {
		return ErrNotImplemented
	}
	return c.handler.InitWSSubscription(ctx, eventChan)
}

// InitHTTPPolling initializes HTTP polling for chain events with thread-safe access.
// It locks the handler mutex for reading to ensure safe concurrent access to the handler.
// If the handler is not implemented, it returns an error.
//
// Parameters:
// - ctx: context for managing the lifecycle of the HTTP polling.
// - eventChan: channel to receive chain events.
//
// Returns:
// - error: an error if the handler is not implemented or if any issue occurs during initialization.
func (c *Chain) InitHTTPPolling(ctx context.Context, eventChan chan relaytypes.ChainEvent) error {
	c.handlerMutex.RLock()
	defer c.handlerMutex.RUnlock()

	if c.handler == nil {
		return ErrNotImplemented
	}
	return c.handler.InitHTTPPolling(ctx, eventChan)
}

// ValidateTransaction validates a transaction based on the quote and the event.
//
// Parameters:
// - ctx: the context for managing the request.
// - quote: the quote containing transaction details.
// - event: the event containing chain event details.
//
// Returns:
// - error: an error if the transaction validation fails.
func (c *Chain) ValidateTransaction(ctx context.Context, quote *relaytypes.Quote, event relaytypes.ChainEvent) error {
	c.handlerMutex.RLock()
	defer c.handlerMutex.RUnlock()

	if c.handler == nil {
		return ErrNotImplemented
	}

	return c.handler.ValidateTransaction(ctx, quote, event)
}

// ShutdownListeners stops all active subscriptions and event handlers.
func (c *Chain) ShutdownListeners() {
	c.handlerMutex.RLock()
	defer c.handlerMutex.RUnlock()

	if c.handler != nil {
		c.handler.ShutdownListeners()
	}
}

// EstimateGas estimates transaction gas with thread-safe access.
// It locks the estimator mutex for reading to ensure safe concurrent access to the estimator.
// If the estimator is not implemented, it returns an error.
//
// Parameters:
// - ctx: context for managing the lifecycle of the gas estimation.
// - to: the recipient address of the transaction.
// - value: the amount of value to be sent in the transaction.
// - data: the input data for the transaction.
//
// Returns:
// - uint64: the estimated gas amount.
// - error: an error if the estimator is not implemented or if any issue occurs during estimation.
func (c *Chain) EstimateGas(ctx context.Context, to string, value *big.Int, data []byte) (uint64, error) {
	c.estimatorMutex.RLock()
	defer c.estimatorMutex.RUnlock()

	if c.estimator == nil {
		return 0, ErrNotImplemented
	}
	return c.estimator.EstimateGas(ctx, to, value, data)
}

// SendAsset sends asset with thread-safe access.
// It locks the sender mutex for reading to ensure safe concurrent access to the sender.
// If the sender is not implemented, it returns an error.
//
// Parameters:
// - ctx: context for managing the lifecycle of the asset sending.
// - intent: the transaction intent containing details of the asset to be sent.
//
// Returns:
// - *relaytypes.Transaction: the transaction instance.
// - error: an error if the sender is not implemented or if any issue occurs during sending.
func (c *Chain) SendAsset(ctx context.Context, intent *relaytypes.Intent) (*relaytypes.Transaction, error) {
	c.senderMutex.RLock()
	defer c.senderMutex.RUnlock()

	if c.sender == nil {
		return nil, ErrNotImplemented
	}
	return c.sender.SendAsset(ctx, intent)
}

// WaitTransactionConfirmation waits for transaction confirmation with thread-safe access.
// It locks the watcher mutex for reading to ensure safe concurrent access to the watcher.
// If the watcher is not implemented, it returns an error.
//
// Parameters:
// - ctx: context for managing the lifecycle of the transaction confirmation.
// - tx: the transaction to be confirmed.
//
// Returns:
// - bool: true if the transaction is confirmed, false otherwise.
// - error: an error if the watcher is not implemented or if any issue occurs during confirmation.
func (c *Chain) WaitTransactionConfirmation(ctx context.Context, tx *relaytypes.Transaction) (bool, error) {
	c.watcherMutex.RLock()
	defer c.watcherMutex.RUnlock()

	if c.watcher == nil {
		return false, ErrNotImplemented
	}
	return c.watcher.WaitTransactionConfirmation(ctx, tx)
}

// GetConfig returns chain configuration.
//
// Returns:
// - *relaytypes.ChainConfig: the chain configuration instance.
func (c *Chain) GetConfig() *relaytypes.ChainConfig {
	return c.config
}

// Helper methods with thread-safe access to dependencies

// GetEstimator returns the gas estimator with thread-safe access.
// It locks the estimator mutex for reading to ensure safe concurrent access to the estimator.
//
// Returns:
// - relaytypes.GasEstimator: the gas estimator instance.
func (c *Chain) GetEstimator() relaytypes.GasEstimator {
	c.estimatorMutex.RLock()
	defer c.estimatorMutex.RUnlock()
	return c.estimator
}

// GetSender returns the transaction sender with thread-safe access.
// It locks the sender mutex for reading to ensure safe concurrent access to the sender.
//
// Returns:
// - relaytypes.TransactionSender: the transaction sender instance.
func (c *Chain) GetSender() relaytypes.TransactionSender {
	c.senderMutex.RLock()
	defer c.senderMutex.RUnlock()
	return c.sender
}

// GetWatcher returns the transaction watcher with thread-safe access.
// It locks the watcher mutex for reading to ensure safe concurrent access to the watcher.
//
// Returns:
// - relaytypes.TransactionWatcher: the transaction watcher instance.
func (c *Chain) GetWatcher() relaytypes.TransactionWatcher {
	c.watcherMutex.RLock()
	defer c.watcherMutex.RUnlock()
	return c.watcher
}
