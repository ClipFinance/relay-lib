package handler

import (
	"context"
	commontypes "github.com/ClipFinance/relay-lib/common/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

// Constants for event handler timeouts and retry attempts.
const (
	contextTimeout       = 30 * time.Second // Timeout for context operations.
	reconnectTimeout     = 5 * time.Second  // Timeout for reconnect attempts.
	retryTimeout         = 5 * time.Minute  // Timeout for retry operations.
	maxReconnectAttempts = 3                // Maximum number of reconnect attempts.
)

// EventHandler handles chain events with thread-safe access.
// It manages subscriptions, polling, and client updates.
type EventHandler struct {
	ctx                  context.Context             // Context for managing lifecycle.
	cancel               context.CancelFunc          // Cancel function for context.
	chainConfig          *commontypes.ChainConfig    // Chain configuration.
	logger               *logrus.Logger              // Logger for logging events.
	client               *ethclient.Client           // Ethereum client.
	solverAddress        string                      // Solver address.
	eventChan            chan commontypes.ChainEvent // Channel for chain events.
	relaySubscription    *commontypes.Subscription   // Subscription for relay events.
	transferSubscription *commontypes.Subscription   // Subscription for transfer events.
	lastProcessedBlock   uint64                      // Last processed block number.
	lastBlockMutex       sync.RWMutex                // Mutex for last processed block.
	pollingTicker        *time.Ticker                // Ticker for polling.
}

// NewEventHandler creates a new event handler instance.
//
// Parameters:
// - ctx: context for managing the lifecycle of the event handler.
// - config: the chain configuration.
// - logger: the logger for logging events.
// - client: the Ethereum client.
// - solverAddr: the solver address.
// - eventChan: the channel to receive chain events.
//
// Returns:
// - *EventHandler: a new EventHandler instance.
// - error: an error if any issue occurs during creation.
func NewEventHandler(
	ctx context.Context,
	config *commontypes.ChainConfig,
	logger *logrus.Logger,
	client *ethclient.Client,
	solverAddr string,
	eventChan chan commontypes.ChainEvent,
) (*EventHandler, error) {
	handlerCtx, cancel := context.WithCancel(ctx)

	handler := &EventHandler{
		chainConfig:          config,
		logger:               logger,
		ctx:                  handlerCtx,
		cancel:               cancel,
		client:               client,
		solverAddress:        solverAddr,
		eventChan:            eventChan,
		relaySubscription:    &commontypes.Subscription{},
		transferSubscription: &commontypes.Subscription{},
	}

	return handler, nil
}

// UpdateClient updates the Ethereum client and restarts subscriptions and polling.
//
// Parameters:
// - client: the new Ethereum client.
func (h *EventHandler) UpdateClient(client *ethclient.Client) {
	h.cancel()

	if h.relaySubscription != nil {
		h.relaySubscription.Close()
		h.relaySubscription = &commontypes.Subscription{}
	}
	if h.transferSubscription != nil {
		h.transferSubscription.Close()
		h.transferSubscription = &commontypes.Subscription{}
	}

	handlerCtx, cancel := context.WithCancel(context.Background())
	h.ctx = handlerCtx
	h.cancel = cancel

	h.client = client

	if h.pollingTicker != nil {
		h.pollingTicker.Stop()
		if err := h.StartHTTPPolling(); err != nil {
			h.logger.WithField("chain", h.chainConfig.Name).WithError(err).Error("Failed to restart HTTP polling after client update")
		}
		return
	}

	if err := h.setupSubscriptions(h.solverAddress); err != nil {
		h.logger.WithField("chain", h.chainConfig.Name).WithError(err).Error("Failed to setup subscriptions after client update")
	}
}

// Stop stops the event handler and closes subscriptions and polling.
func (h *EventHandler) Stop() {
	h.cancel()
	if h.relaySubscription != nil {
		h.relaySubscription.Close()
	}
	if h.transferSubscription != nil {
		h.transferSubscription.Close()
	}
	if h.pollingTicker != nil {
		h.pollingTicker.Stop()
	}
}
