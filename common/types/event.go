package types

import (
	"time"
)

// ChainEvent represents an event on a blockchain.
//
// Fields:
// - ChainID: the unique identifier for the chain where the event occurred.
// - EventType: the type of the event.
// - BlockNumber: the block number where the event was included.
// - TxHash: the transaction hash associated with the event.
// - LogIndex: the index of the log within the block.
// - Data: the event data.
// - FromTokenAddr: the address of the token that emitted the event.
// - FromAddress: the address that emitted the event.
// - ToAddress: the address that received the event.
// - TransactionHash: the hash of the transaction that emitted the event.
// - QuoteID: the unique identifier for the quote associated with the event.
// - FromTxMinedAt: the time when the transaction was mined.
// - FromNonce: the nonce of the transaction that emitted the event.
// - TransactionAmount: the amount of the transaction that emitted the event.
type ChainEvent struct {
	ChainID           uint64
	BlockNumber       uint64
	BlockHash         string
	FromTokenAddr     string
	FromAddress       string
	ToAddress         string
	TransactionHash   string
	QuoteID           string
	FromTxMinedAt     time.Time
	TransactionAmount string
	Metadata          interface{}
}
