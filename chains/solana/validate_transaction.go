package solana

import (
	"context"
	"github.com/ClipFinance/relay-lib/common/types"
	"github.com/pkg/errors"
)

// ValidateTransaction validates a transaction based on the quote and the event
func (s *solana) ValidateTransaction(ctx context.Context, quote *types.Quote, event types.ChainEvent) error {
	// TODO: need to implement this function
	return errors.New("not implemented")
}
