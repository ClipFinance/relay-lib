package solana

import (
	"context"
	"encoding/base64"
	"math/big"

	sol "github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/pkg/errors"
)

const (
	defaultComputeUnits = 60_000 // for create ATA account and transfer tokens is 50,512 units,and we will use default with ~20% buffer
	defaultPriorityFee  = 10_000 // minimum priority fee in microlamports (1_000_000 microlamports = 1 lamports)
	computeUnitBuffer   = 110    // 10% buffer
)

// EstimateGas estimates the gas required to execute a transaction with the given parameters.
func (s *solana) EstimateGas(ctx context.Context, to string, value *big.Int, data []byte) (uint64, error) {
	// Convert binary data to base64
	messageBase64 := base64.StdEncoding.EncodeToString(data)

	// Simulate transaction
	feeResponse, err := s.client.GetFeeForMessage(
		ctx,
		messageBase64, // Now passing base64 encoded message
		rpc.CommitmentProcessed,
	)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get fee for message")
	}
	if feeResponse.Value == nil {
		return 0, errors.New("fee value is nil, check transaction validity")
	}

	return *feeResponse.Value, nil
}

func (s *solana) estimateTransactionCost(ctx context.Context, instructions []sol.Instruction, feePayer sol.PublicKey, latestBlockHash sol.Hash) (uint64, error) {
	// Get base fee
	tx, err := sol.NewTransaction(
		instructions,
		latestBlockHash,
		sol.TransactionPayer(feePayer),
	)
	if err != nil {
		return 0, errors.Wrap(err, "failed to create transaction")
	}

	// Marshal transaction to binary
	message, err := tx.Message.MarshalBinary()
	if err != nil {
		return 0, errors.Wrap(err, "failed to marshal transaction")
	}

	return s.EstimateGas(ctx, "", nil, message)
}

func (s *solana) getPriorityFee(ctx context.Context) uint64 {
	priorityFeesResp, err := s.client.GetRecentPrioritizationFees(ctx, sol.PublicKeySlice{})
	if err != nil {
		s.logger.WithError(err).Warn("Failed to get priority fees, using default")
		return defaultPriorityFee
	}

	if len(priorityFeesResp) == 0 {
		s.logger.Warn("No priority fees returned, using default")
		return defaultPriorityFee
	}

	var maxPriorityFee uint64
	for _, fee := range priorityFeesResp {
		if fee.PrioritizationFee > maxPriorityFee {
			maxPriorityFee = fee.PrioritizationFee
		}
	}

	if maxPriorityFee == 0 {
		s.logger.Info("Max priority fee is 0, using default")
		return defaultPriorityFee
	}

	return maxPriorityFee
}
