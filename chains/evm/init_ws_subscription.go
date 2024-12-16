package evm

import (
	"context"
	"github.com/ClipFinance/relay-lib/chains/evm/handler"
	"github.com/ClipFinance/relay-lib/common/types"
	"github.com/pkg/errors"
)

// InitWSSubscription initializes the WebSocket subscription for the EVM chain.
//
// Parameters:
// - ctx: the context for managing the initialization process.
// - eventChan: the channel to receive chain events.
//
// Returns:
// - error: an error if the client is not initialized, if the event handler creation fails, or if starting the WebSocket subscription fails.
func (e *evm) InitWSSubscription(ctx context.Context, eventChan chan types.ChainEvent) error {
	e.eventHandlerMutex.Lock()
	defer e.eventHandlerMutex.Unlock()

	e.clientMutex.RLock()
	client := e.client
	e.clientMutex.RUnlock()

	if client == nil {
		return errors.New("client not initialized")
	}

	if e.eventHandler != nil {
		e.eventHandler.Stop()
	}

	eventHandler, err := handler.NewEventHandler(
		ctx,
		e.config,
		e.logger,
		client,
		e.config.SolverAddress,
		eventChan,
	)
	if err != nil {
		return errors.Wrap(err, "failed to create event handler")
	}

	if err := eventHandler.StartWSSubscription(); err != nil {
		eventHandler.Stop()
		return errors.Wrap(err, "failed to start event handler")
	}

	e.eventHandler = eventHandler
	return nil
}
