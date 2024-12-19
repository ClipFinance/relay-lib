package utils

import (
	"encoding/hex"
	"fmt"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
)

const (
	// minTransferInputLength is the minimum length of a transfer input data
	minTransferInputLength = 68 // 4 bytes function signature + 32 bytes quoteId + 32 bytes amount
)

// ExtractQuoteIDFromTxData extracts quote ID from transaction input data
func ExtractQuoteIDFromTxData(data []byte) (string, error) {
	if len(data) <= minTransferInputLength {
		return "", errors.New("invalid transaction input length, expected min than: " + fmt.Sprint(minTransferInputLength) + ", got: " + fmt.Sprint(len(data)))
	}

	quoteIDBytes := data[minTransferInputLength:]
	if len(quoteIDBytes) == 0 {
		return "", errors.New("quote ID is empty")
	}

	return hex.EncodeToString(quoteIDBytes), nil
}

// GetEventType determines event type from log topics
func GetEventType(log types.Log) string {
	if len(log.Topics) == 0 {
		return ""
	}

	eventSig := log.Topics[0].Hex()
	switch eventSig {
	case crypto.Keccak256Hash([]byte("FundsForwarded()")).Hex():
		return "FundsForwarded"
	case crypto.Keccak256Hash([]byte("FundsForwardedWithData(bytes)")).Hex():
		return "FundsForwardedWithData"
	case crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)")).Hex():
		return "Transfer"
	default:
		return ""
	}
}
