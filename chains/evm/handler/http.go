package handler

import (
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

const (
	// defaultPollingInterval is the default interval for polling events.
	defaultPollingInterval = 5 * time.Second
	// maxBlockRange is the maximum number of blocks to fetch in a single poll.
	maxBlockRange = uint64(1000)
)

// StartHTTPPolling starts polling for FundsForwarded, FundsForwardedWithData, and Transfer events.
// It initializes a ticker to poll at regular intervals and processes events in a separate goroutine.
//
// Returns:
// - error: an error if any issue occurs during the polling setup.
func (h *EventHandler) StartHTTPPolling() error {
	h.pollingTicker = time.NewTicker(defaultPollingInterval)

	h.logger.WithFields(logrus.Fields{
		"chain":    h.chainConfig.Name,
		"interval": defaultPollingInterval,
	}).Info("Start polling FundsForwarded, FundsForwardedWithData and Transfer events")

	go func() {
		for {
			select {
			case <-h.ctx.Done():
				return
			case <-h.pollingTicker.C:
				if err := h.pollEvents(); err != nil {
					h.logger.WithError(err).Error("Error polling events")
				}
			}
		}
	}()

	return nil
}

// pollEvents polls for FundsForwarded, FundsForwardedWithData, and Transfer events.
// It retrieves the current block number and processes the block range for events.
//
// Returns:
// - error: an error if any issue occurs during event polling.
func (h *EventHandler) pollEvents() error {
	currentBlock, err := h.client.BlockNumber(h.ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get current block number")
	}

	h.lastBlockMutex.RLock()
	fromBlock := h.lastProcessedBlock
	h.lastBlockMutex.RUnlock()

	if fromBlock == 0 {
		h.lastBlockMutex.Lock()
		h.lastProcessedBlock = currentBlock
		h.lastBlockMutex.Unlock()
		return nil
	}

	if currentBlock <= fromBlock {
		return nil
	}

	toBlock := fromBlock + maxBlockRange
	if toBlock > currentBlock {
		toBlock = currentBlock
	}

	if err := h.processBlockRange(fromBlock+1, toBlock); err != nil {
		return errors.Wrap(err, "failed to process block range")
	}

	h.lastBlockMutex.Lock()
	h.lastProcessedBlock = toBlock
	h.lastBlockMutex.Unlock()

	return nil
}

// processBlockRange processes a block range and filters for FundsForwarded, FundsForwardedWithData, and Transfer events.
// It queries logs for the specified block range and processes the events.
//
// Parameters:
// - fromBlock: the starting block number.
// - toBlock: the ending block number.
//
// Returns:
// - error: an error if any issue occurs during block range processing.
func (h *EventHandler) processBlockRange(fromBlock, toBlock uint64) error {
	relayQuery := ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(fromBlock),
		ToBlock:   new(big.Int).SetUint64(toBlock),
		Addresses: []common.Address{common.HexToAddress(h.chainConfig.RelayReceiver)},
		Topics: [][]common.Hash{{
			crypto.Keccak256Hash([]byte("FundsForwarded()")),
			crypto.Keccak256Hash([]byte("FundsForwardedWithData(bytes)")),
		}},
	}

	transferQuery := ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(fromBlock),
		ToBlock:   new(big.Int).SetUint64(toBlock),
		Topics: [][]common.Hash{
			{crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))},
			nil,
			{common.BytesToHash(common.HexToAddress(h.solverAddress).Bytes())},
		},
	}

	// Get logs for both queries concurrently using goroutines.
	type queryResult struct {
		logs []ethtypes.Log
		err  error
	}

	relayResultChan := make(chan queryResult, 1)
	transferResultChan := make(chan queryResult, 1)

	go func() {
		logs, err := h.client.FilterLogs(h.ctx, relayQuery)
		relayResultChan <- queryResult{logs: logs, err: err}
	}()

	go func() {
		logs, err := h.client.FilterLogs(h.ctx, transferQuery)
		transferResultChan <- queryResult{logs: logs, err: err}
	}()

	// Wait for both queries to complete and handle any errors.
	relayResult := <-relayResultChan
	if relayResult.err != nil {
		return errors.Wrap(relayResult.err, "failed to get relay logs")
	}

	transferResult := <-transferResultChan
	if transferResult.err != nil {
		return errors.Wrap(transferResult.err, "failed to get transfer logs")
	}

	// Process all logs.
	for _, log := range relayResult.logs {
		if err := h.processEvent(utils.GetEventType(log), log); err != nil {
			h.logger.WithFields(logrus.Fields{
				"chain":     h.chainConfig.Name,
				"eventType": "relay",
				"txHash":    log.TxHash.Hex(),
				"block":     log.BlockNumber,
			}).WithError(err).Error("Failed to process relay log")
		}
	}

	for _, log := range transferResult.logs {
		if err := h.processEvent(utils.GetEventType(log), log); err != nil {
			h.logger.WithFields(logrus.Fields{
				"chain":     h.chainConfig.Name,
				"eventType": "transfer",
				"txHash":    log.TxHash.Hex(),
				"block":     log.BlockNumber,
			}).WithError(err).Error("Failed to process transfer log")
		}
	}

	return nil
}
