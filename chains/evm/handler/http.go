package handler

import (
	"github.com/ClipFinance/relay-lib/common/types"
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
		if err := h.processEvent("relay", log); err != nil {
			h.logger.WithFields(logrus.Fields{
				"chain":     h.chainConfig.Name,
				"eventType": "relay",
				"txHash":    log.TxHash.Hex(),
				"block":     log.BlockNumber,
			}).WithError(err).Error("Failed to process relay log")
		}
	}

	for _, log := range transferResult.logs {
		if err := h.processEvent("transfer", log); err != nil {
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

	chainEvent := types.ChainEvent{
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

	h.logger.WithFields(logrus.Fields{
		"chain":       h.chainConfig.Name,
		"eventType":   eventType,
		"blockNumber": log.BlockNumber,
		"blockHash":   log.BlockHash.Hex(),
		"txHash":      log.TxHash.Hex(),
		"quoteId":     quoteId,
	}).Info("Successfully received event")

	return nil
}
