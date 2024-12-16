package solana

import (
	"context"
	"github.com/ClipFinance/relay-lib/common/types"
)

func (s *solana) WaitTransactionConfirmation(ctx context.Context, tx *types.Transaction) (bool, error) {
	// TODO: Implement transaction confirmation waiting for Solana
	return false, nil
}
