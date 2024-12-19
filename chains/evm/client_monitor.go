package evm

import (
	"context"

	"github.com/ClipFinance/relay-lib/connectionmonitor"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"
)

// evmConnectionManager implements the BlockchainClient interface and manages the connection to the EVM chain.
type evmConnectionManager struct {
	chain *evm // Reference to the EVM chain instance.
}

// initMonitor initializes the connection monitor for the EVM chain.
//
// Parameters:
// - ctx: the context for managing the initialization process.
//
// Returns:
// - error: an error if there is an issue starting the connection monitor.
func (e *evm) initMonitor(ctx context.Context) error {
	e.monitorMutex.Lock()
	defer e.monitorMutex.Unlock()

	connectionManager := &evmConnectionManager{chain: e}
	e.monitor = connectionmonitor.NewConnectionMonitor(connectionManager, e.logger, e.config.Name)
	return e.monitor.Start(ctx)
}

// CheckConnection checks the connection to the Ethereum client by retrieving the current block number.
//
// Parameters:
// - ctx: the context for managing the connection check.
//
// Returns:
// - error: an error if the client is not initialized or if there is an issue retrieving the block number.
func (w *evmConnectionManager) CheckConnection(ctx context.Context) error {
	w.chain.clientMutex.RLock()
	client := w.chain.client
	w.chain.clientMutex.RUnlock()

	if client == nil {
		return errors.New("client not initialized")
	}

	_, err := client.BlockNumber(ctx)
	return err
}

// Reconnect re-establishes the connection to the Ethereum client and updates the event handler with the new client.
//
// Parameters:
// - ctx: the context for managing the reconnection process.
//
// Returns:
// - error: an error if there is an issue dialing the new client or updating the event handler.
func (w *evmConnectionManager) Reconnect(ctx context.Context) error {
	w.chain.clientMutex.Lock()
	defer w.chain.clientMutex.Unlock()

	if w.chain.client != nil {
		w.chain.client.Close()
	}

	client, err := ethclient.Dial(w.chain.config.RpcUrl)
	if err != nil {
		return err
	}

	w.chain.client = client

	w.chain.eventHandlerMutex.Lock()
	if w.chain.eventHandler != nil {
		w.chain.eventHandler.UpdateClient(ctx, client)
	}
	w.chain.eventHandlerMutex.Unlock()

	return nil
}
