package types

import (
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"sync"
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
type ChainEvent struct {
	ChainID     uint64
	EventType   string
	BlockNumber uint64
	TxHash      string
	LogIndex    uint
	Data        []byte
}

// Subscription wraps event subscription data.
//
// Fields:
// - Subscription: the event subscription.
// - EventChan: the channel to receive Ethereum logs.
// - sync.Mutex: the mutex to protect access to the subscription data.
type Subscription struct {
	Subscription event.Subscription
	EventChan    chan ethtypes.Log
	sync.Mutex
}

// Close closes the subscription and the event channel.
func (s *Subscription) Close() {
	s.Lock()
	defer s.Unlock()

	if s.Subscription != nil {
		s.Subscription.Unsubscribe()
	}

	if s.EventChan != nil {
		close(s.EventChan)
	}
}
