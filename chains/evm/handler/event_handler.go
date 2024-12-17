package handler

import (
	"context"
	"github.com/ClipFinance/relay-lib/chains/evm/utils"
	relaytypes "github.com/ClipFinance/relay-lib/common/types"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"math/big"
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
	ctx                  context.Context            // Context for managing lifecycle.
	cancel               context.CancelFunc         // Cancel function for context.
	chainConfig          *relaytypes.ChainConfig    // Chain configuration.
	logger               *logrus.Logger             // Logger for logging events.
	client               *ethclient.Client          // Ethereum client.
	solverAddress        string                     // Solver address.
	eventChan            chan relaytypes.ChainEvent // Channel for chain events.
	relaySubscription    *relaytypes.Subscription   // Subscription for relay events.
	transferSubscription *relaytypes.Subscription   // Subscription for transfer events.
	lastProcessedBlock   uint64                     // Last processed block number.
	lastBlockMutex       sync.RWMutex               // Mutex for last processed block.
	pollingTicker        *time.Ticker               // Ticker for polling.
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
	config *relaytypes.ChainConfig,
	logger *logrus.Logger,
	client *ethclient.Client,
	solverAddr string,
	eventChan chan relaytypes.ChainEvent,
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
		relaySubscription:    &relaytypes.Subscription{},
		transferSubscription: &relaytypes.Subscription{},
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
		h.relaySubscription = &relaytypes.Subscription{}
	}
	if h.transferSubscription != nil {
		h.transferSubscription.Close()
		h.transferSubscription = &relaytypes.Subscription{}
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

// processEvent processes a single event log and sends it to the event channel.
//
// Parameters:
// - eventType: the type of the event (e.g., "relay", "transfer").
// - log: the event log to process.
//
// Returns:
// - error: an error if any issue occurs during event processing.
func (h *EventHandler) processEvent(eventType string, log ethtypes.Log) error {
	tx, _, err := h.client.TransactionByHash(h.ctx, log.TxHash)
	if err != nil {
		return errors.Wrap(err, "failed to get transaction by hash")
	}

	// Get Tx Sender
	signer := ethtypes.LatestSignerForChainID(tx.ChainId())
	fromAddress, err := ethtypes.Sender(signer, tx)
	if err != nil {
		return errors.Wrap(err, "failed to get transaction sender")
	}

	var quoteId string
	var amount string

	switch eventType {
	case "FundsForwardedWithData":
		quoteId, err = utils.ExtractQuoteIDFromTxData(tx.Data())
		if err != nil {
			return errors.Wrap(err, "failed to extract quoteId from FundsForwardedWithData event")
		}
		quoteId = "0x" + quoteId
		amount = tx.Value().String()

	case "FundsForwarded":
		quoteId = string(tx.Data())
		amount = tx.Value().String()

	case "Transfer":
		input := tx.Data()
		quoteId, err = utils.ExtractQuoteIDFromTxData(input)
		if err != nil {
			return errors.Wrap(err, "failed to extract quoteId from Transfer event")
		}
		quoteId = "0x" + quoteId

		amount = new(big.Int).SetBytes(log.Data).String()

	default:
		return errors.New("unknown event type: " + eventType)
	}

	block, err := h.client.HeaderByNumber(h.ctx, new(big.Int).SetUint64(log.BlockNumber))
	if err != nil {
		return errors.Wrap(err, "failed to get block time")
	}

	chainEvent := relaytypes.ChainEvent{
		ChainID:           h.chainConfig.ChainID,
		BlockNumber:       log.BlockNumber,
		BlockHash:         log.BlockHash.String(),
		FromTokenAddr:     log.Address.String(),
		FromAddress:       fromAddress.Hex(),
		ToAddress:         tx.To().Hex(),
		TransactionHash:   log.TxHash.String(),
		QuoteID:           quoteId,
		FromTxMinedAt:     time.Unix(int64(block.Time), 0),
		FromNonce:         tx.Nonce(),
		TransactionAmount: amount,
	}

	h.eventChan <- chainEvent

	return nil
}
