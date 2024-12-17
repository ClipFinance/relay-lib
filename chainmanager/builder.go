package chainmanager

import (
	"github.com/ClipFinance/relay-lib/common/types"
)

// ChainBuilder is a builder pattern implementation for chain configuration.
// It allows setting various components of the chain such as gas estimator,
// transaction sender, transaction watcher, and event handler.
type ChainBuilder struct {
	config    *types.ChainConfig       // Chain configuration.
	estimator types.GasEstimator       // Gas estimator implementation.
	sender    types.TransactionSender  // Transaction sender implementation.
	watcher   types.TransactionWatcher // Transaction watcher implementation.
	handler   types.EventHandler       // Event handler implementation.
	provider  types.BalanceProvider    // Balance provider implementation.
}

// NewChainBuilder creates a new chain builder instance.
//
// Parameters:
// - config: the chain configuration.
//
// Returns:
// - *ChainBuilder: a new ChainBuilder instance.
func NewChainBuilder(config *types.ChainConfig) *ChainBuilder {
	return &ChainBuilder{
		config: config,
	}
}

// WithGasEstimator sets gas estimator implementation.
//
// Parameters:
// - estimator: the gas estimator implementation.
//
// Returns:
// - *ChainBuilder: the updated ChainBuilder instance.
func (b *ChainBuilder) WithGasEstimator(estimator types.GasEstimator) *ChainBuilder {
	b.estimator = estimator
	return b
}

// WithTransactionSender sets transaction sender implementation.
//
// Parameters:
// - sender: the transaction sender implementation.
//
// Returns:
// - *ChainBuilder: the updated ChainBuilder instance.
func (b *ChainBuilder) WithTransactionSender(sender types.TransactionSender) *ChainBuilder {
	b.sender = sender
	return b
}

// WithTransactionWatcher sets transaction watcher implementation.
//
// Parameters:
// - watcher: the transaction watcher implementation.
//
// Returns:
// - *ChainBuilder: the updated ChainBuilder instance.
func (b *ChainBuilder) WithTransactionWatcher(watcher types.TransactionWatcher) *ChainBuilder {
	b.watcher = watcher
	return b
}

// WithEventHandler sets event handler implementation.
//
// Parameters:
// - handler: the event handler implementation.
//
// Returns:
// - *ChainBuilder: the updated ChainBuilder instance.
func (b *ChainBuilder) WithEventHandler(handler types.EventHandler) *ChainBuilder {
	b.handler = handler
	return b
}

// WithBalanceProvider sets balance provider implementation.
//
// Parameters:
// - provider: the balance provider implementation.
//
// Returns:
// - *ChainBuilder: the updated ChainBuilder instance.
func (b *ChainBuilder) WithBalanceProvider(provider types.BalanceProvider) *ChainBuilder {
	b.provider = provider
	return b
}

// Build creates a new chain instance with configured implementations.
//
// Returns:
// - types.Chain: a new Chain instance with the configured implementations.
func (b *ChainBuilder) Build() types.Chain {
	return NewChain(b.config, b.estimator, b.sender, b.watcher, b.handler, b.provider)
}
