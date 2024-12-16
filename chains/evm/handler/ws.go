package handler

import (
	"context"
	"github.com/ClipFinance/relay-lib/common/utils"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"math/big"
	"time"
)

// StartWSSubscription starts the WebSocket subscription for relay and transfer events.
// It sets up the necessary subscriptions and starts handling events in a separate goroutine.
//
// Returns:
// - error: an error if any issue occurs during the subscription setup.
func (h *EventHandler) StartWSSubscription() error {
	if err := h.setupSubscriptions(h.solverAddress); err != nil {
		return errors.Wrap(err, "failed to setup subscriptions")
	}

	go h.handleEvents()

	return nil
}

// reconnectSubscription attempts to reconnect the specified subscription type (relay or transfer).
// It retries the connection up to a maximum number of attempts, with a delay between attempts.
//
// Parameters:
// - subscriptionType: the type of subscription to reconnect ("relay" or "transfer").
//
// Returns:
// - error: an error if the reconnection fails or the context is cancelled.
func (h *EventHandler) reconnectSubscription(subscriptionType string) error {
	// Close existing subscriptions
	if h.relaySubscription != nil {
		h.relaySubscription.Close()
	}
	if h.transferSubscription != nil {
		h.transferSubscription.Close()
	}

	ticker := time.NewTicker(retryTimeout)
	defer ticker.Stop()

	for {
		for attempt := 1; attempt <= maxReconnectAttempts; attempt++ {
			select {
			case <-h.ctx.Done():
				return errors.New("context cancelled during reconnection")
			default:
				h.logger.WithFields(logrus.Fields{
					"chain":   h.chainConfig.Name,
					"type":    subscriptionType,
					"attempt": attempt,
				}).Info("Attempting to reconnect subscription")

				if err := h.setupSubscriptions(h.solverAddress); err != nil {
					h.logger.WithField("chain", h.chainConfig.Name).WithError(err).Error("Failed to reconnect subscription")

					if attempt == maxReconnectAttempts {
						h.logger.WithFields(logrus.Fields{
							"chain": h.chainConfig.Name,
							"type":  subscriptionType,
						}).Warn("Max reconnect attempts reached, waiting for retry timeout")
						<-ticker.C
						attempt = 0
						continue
					}

					time.Sleep(reconnectTimeout)
					continue
				}

				h.logger.WithFields(logrus.Fields{
					"chain": h.chainConfig.Name,
					"type":  subscriptionType,
				}).Info("Successfully reconnected subscription")
				return nil
			}
		}
	}
}

// handleEvents handles incoming events from the relay and transfer subscriptions.
// It processes events and attempts to reconnect subscriptions in case of errors.
func (h *EventHandler) handleEvents() {
	for {
		select {
		case <-h.ctx.Done():
			return

		case err := <-h.relaySubscription.Subscription.Err():
			h.logger.WithError(err).Error("Relay subscription error")
			if err := h.reconnectSubscription("relay"); err != nil {
				h.logger.WithError(err).Error("Failed to reconnect relay subscription")
			}

		case err := <-h.transferSubscription.Subscription.Err():
			h.logger.WithError(err).Error("Transfer subscription error")
			if err := h.reconnectSubscription("transfer"); err != nil {
				h.logger.WithError(err).Error("Failed to reconnect transfer subscription")
			}

		case event := <-h.relaySubscription.EventChan:
			err := h.processEvent(utils.GetEventType(event), event)
			if err != nil {
				h.logger.WithError(err).Error("Failed to process relay event")
			}

		case event := <-h.transferSubscription.EventChan:
			err := h.processEvent(utils.GetEventType(event), event)
			if err != nil {
				h.logger.WithError(err).Error("Failed to process transfer event")
			}
		}
	}
}

// setupSubscriptions sets up the relay and transfer subscriptions using the provided solver address.
//
// Parameters:
// - solverAddress: the address used for setting up the subscriptions.
//
// Returns:
// - error: an error if any issue occurs during the subscription setup.
func (h *EventHandler) setupSubscriptions(solverAddress string) error {
	if h.relaySubscription.Subscription != nil {
		h.relaySubscription.Subscription.Unsubscribe()
	}
	if h.relaySubscription.Subscription != nil {
		h.transferSubscription.Subscription.Unsubscribe()
	}

	ctx, cancel := context.WithTimeout(h.ctx, contextTimeout)
	defer cancel()

	relayQuery := ethereum.FilterQuery{
		Addresses: []common.Address{common.HexToAddress(h.chainConfig.RelayReceiver)},
		Topics: [][]common.Hash{{
			crypto.Keccak256Hash([]byte("FundsForwarded()")),
			crypto.Keccak256Hash([]byte("FundsForwardedWithData(bytes)")),
		}},
	}

	transferQuery := ethereum.FilterQuery{
		Topics: [][]common.Hash{
			{crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))},
			nil,
			{common.BytesToHash(common.HexToAddress(solverAddress).Bytes())},
		},
	}

	var err error
	for attempt := 0; attempt < maxReconnectAttempts; attempt++ {
		if ctx.Err() != nil {
			return errors.New("context cancelled while setting up subscriptions")
		}

		if err = h.setupRelayReceiverSubscription(ctx, relayQuery); err != nil {
			h.logger.WithField("chain", h.chainConfig.Name).WithError(err).Error("Failed to setup relay subscription")
			time.Sleep(reconnectTimeout)
			continue
		}

		if err = h.setupTransferSubscription(ctx, transferQuery); err != nil {
			h.logger.WithField("chain", h.chainConfig.Name).WithError(err).Error("Failed to setup transfer subscription")
			time.Sleep(reconnectTimeout)
			continue
		}

		break
	}

	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"chain": h.chainConfig.Name,
		}).WithError(err).Error("Failed to setup subscriptions")
		return err
	}

	return nil
}

// setupRelayReceiverSubscription sets up the relay receiver subscription using the provided filter query.
//
// Parameters:
// - ctx: the context for managing the subscription setup.
// - query: the filter query for the relay subscription.
//
// Returns:
// - error: an error if any issue occurs during the subscription setup.
func (h *EventHandler) setupRelayReceiverSubscription(ctx context.Context, query ethereum.FilterQuery) error {
	h.relaySubscription.Lock()
	defer h.relaySubscription.Unlock()

	if h.relaySubscription.Subscription != nil {
		h.logger.WithField("chain", h.chainConfig.Name).Info("Closing relay subscription")
		h.relaySubscription.Subscription.Unsubscribe()
	}

	blockNumber, err := h.client.BlockNumber(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get block number")
	}

	query.FromBlock = new(big.Int).SetUint64(blockNumber)

	eventChan := make(chan ethtypes.Log)
	sub, err := h.client.SubscribeFilterLogs(ctx, query, eventChan)
	if err != nil {
		return errors.Wrap(err, "failed to subscribe to relay events")
	}

	h.relaySubscription.Subscription = sub
	h.relaySubscription.EventChan = eventChan

	h.logger.WithFields(logrus.Fields{
		"chain":       h.chainConfig.Name,
		"blockNumber": blockNumber,
	}).Info("Relay subscription established")

	return nil
}

// setupTransferSubscription sets up the transfer subscription using the provided filter query.
//
// Parameters:
// - ctx: the context for managing the subscription setup.
// - query: the filter query for the transfer subscription.
//
// Returns:
// - error: an error if any issue occurs during the subscription setup.
func (h *EventHandler) setupTransferSubscription(ctx context.Context, query ethereum.FilterQuery) error {
	h.transferSubscription.Lock()
	defer h.transferSubscription.Unlock()

	if h.transferSubscription.Subscription != nil {
		h.logger.WithField("chain", h.chainConfig.Name).Info("Closing transfer subscription")
		h.transferSubscription.Subscription.Unsubscribe()
	}

	blockNumber, err := h.client.BlockNumber(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get block number")
	}

	query.FromBlock = new(big.Int).SetUint64(blockNumber)

	eventChan := make(chan ethtypes.Log)
	sub, err := h.client.SubscribeFilterLogs(ctx, query, eventChan)
	if err != nil {
		return errors.Wrap(err, "failed to subscribe to transfer events")
	}

	h.transferSubscription.Subscription = sub
	h.transferSubscription.EventChan = eventChan

	h.logger.WithFields(logrus.Fields{
		"chain":       h.chainConfig.Name,
		"blockNumber": blockNumber,
	}).Info("Transfer subscription established")

	return nil
}
