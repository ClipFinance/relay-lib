package solana

import (
	"context"
	"math/big"
)

// Implement required interface methods
func (s *solana) EstimateGas(ctx context.Context, to string, value *big.Int, data []byte) (uint64, error) {
	// TODO: Implement gas estimation for Solana
	return 0, nil
}
