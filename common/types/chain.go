package types

import (
	"context"
	"math/big"
)

// ChainConfig holds the configuration for a specific chain implementation.
//
// Fields:
// - Name: the name of the chain.
// - ChainType: the type of the chain.
// - ChainID: the unique identifier for the chain.
// - RpcUrl: the URL for the chain's RPC endpoint.
// - TxType: the type of transactions supported by the chain.
// - WaitNBlocks: the number of blocks to wait for transaction confirmation.
// - PrivateKey: the private key for signing transactions.
// - RelayReceiver: the address of the relay receiver.
type ChainConfig struct {
	Name          string
	ChainType     ChainType
	ChainID       uint64
	RpcUrl        string
	TxType        uint64
	WaitNBlocks   uint64
	PrivateKey    string
	SolverAddress string
	RelayReceiver string
}

// GasEstimator provides gas estimation functionality.
type GasEstimator interface {
	// EstimateGas estimates the gas required for a transaction.
	//
	// Parameters:
	// - ctx: the context for managing the request.
	// - to: the recipient address of the transaction.
	// - value: the amount of Ether to send with the transaction.
	// - data: the input data for the transaction.
	//
	// Returns:
	// - uint64: the estimated gas amount.
	// - error: an error if the gas estimation fails.
	EstimateGas(ctx context.Context, to string, value *big.Int, data []byte) (uint64, error)
}

// TransactionSender provides transaction sending functionality.
type TransactionSender interface {
	// SendAsset sends an asset (native or token) based on the provided transaction intent.
	//
	// Parameters:
	// - ctx: the context for managing the request.
	// - intent: the transaction intent containing details of the asset transfer.
	//
	// Returns:
	// - *Transaction: the transaction details.
	// - error: an error if the transaction sending fails.
	SendAsset(ctx context.Context, intent *TransactionIntent) (*Transaction, error)
}

// TransactionWatcher provides transaction confirmation functionality.
type TransactionWatcher interface {
	// WaitTransactionConfirmation waits for the confirmation of a transaction.
	//
	// Parameters:
	// - ctx: the context for managing the request.
	// - tx: the transaction to wait for confirmation.
	//
	// Returns:
	// - bool: true if the transaction is confirmed successfully, false otherwise.
	// - error: an error if the transaction confirmation fails.
	WaitTransactionConfirmation(ctx context.Context, tx *Transaction) (bool, error)
}

// EventHandler provides event subscription functionality.
type EventHandler interface {
	// InitWSSubscription initializes the WebSocket subscription for the chain.
	//
	// Parameters:
	// - ctx: the context for managing the initialization process.
	// - eventChan: the channel to receive chain events.
	//
	// Returns:
	// - error: an error if the WebSocket subscription initialization fails.
	InitWSSubscription(ctx context.Context, eventChan chan ChainEvent) error

	// InitHTTPPolling initializes the HTTP polling subscription for the chain.
	//
	// Parameters:
	// - ctx: the context for managing the initialization process.
	// - eventChan: the channel to receive chain events.
	//
	// Returns:
	// - error: an error if the HTTP polling subscription initialization fails.
	InitHTTPPolling(ctx context.Context, eventChan chan ChainEvent) error
}

// Chain combines all chain-specific functionality.
type Chain interface {
	GasEstimator
	TransactionSender
	TransactionWatcher
	EventHandler
}

// ChainRegistry manages multiple chains.
type ChainRegistry interface {
	// Add adds a new chain to the registry.
	//
	// Parameters:
	// - config: the configuration for the chain to add.
	//
	// Returns:
	// - error: an error if adding the chain fails.
	Add(config *ChainConfig) error

	// Get retrieves a chain from the registry by its chain ID.
	//
	// Parameters:
	// - chainID: the unique identifier for the chain to retrieve.
	//
	// Returns:
	// - Chain: the retrieved chain instance.
	Get(chainID uint64) Chain

	// Remove removes a chain from the registry by its chain ID.
	//
	// Parameters:
	// - chainID: the unique identifier for the chain to remove.
	Remove(chainID uint64)
}
