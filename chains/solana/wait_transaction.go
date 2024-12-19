package solana

import (
	"context"
	"time"

	"github.com/ClipFinance/relay-lib/chains/solana/utils"
	"github.com/ClipFinance/relay-lib/common/types"
	sol "github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// WaitTransactionConfirmation waits for transaction confirmation using the provided commitment level
const (
	MaxValidSlots   = 160 // 150 + safety margin of 10 slots
	PoolingInterval = 2 * time.Second
)

// WaitTransactionConfirmation waits for transaction confirmation using the provided commitment level
func (s *solana) WaitTransactionConfirmation(ctx context.Context, tx *types.Transaction) (types.TransactionStatus, error) {
	txMetadata, ok := tx.Metadata.(utils.SolanaMetadata)
	if !ok {
		return types.TxFailed, errors.New("failed to get solana metadata from transaction metadata")
	}

	s.logger.WithFields(logrus.Fields{
		"signature":   tx.Hash,
		"initialSlot": txMetadata.BlockhashSlot,
		"blockhash":   txMetadata.Blockhash,
	}).Info("Starting transaction confirmation monitoring")

	sig, err := sol.SignatureFromBase58(tx.Hash)
	if err != nil {
		return types.TxFailed, errors.Wrap(err, "failed to parse signature")
	}

	//  Set commitment level (maybe we should make it configurable for small amounts)
	commitment := rpc.ConfirmationStatusFinalized
	// Create ticker for polling transaction status
	ticker := time.NewTicker(PoolingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return types.TxFailed, ctx.Err()
		case <-ticker.C:
			// Get transaction status
			statuses, err := s.client.GetSignatureStatuses(ctx, true, sig)
			if err != nil {
				s.logger.WithError(err).Warn("Failed to get signature statuses")
				continue
			}

			status := statuses.Value[0]
			if status != nil {
				// Check if transaction is failed
				if status.Err != nil {
					s.logger.WithFields(logrus.Fields{
						"slot":               status.Slot,
						"confirmations":      status.Confirmations,
						"confirmationStatus": status.ConfirmationStatus,
						"error":              status.Err,
					}).Debug("Transaction is failed")
					return types.TxFailed, nil
				}

				s.logger.WithFields(logrus.Fields{
					"slot":               status.Slot,
					"confirmations":      status.Confirmations,
					"confirmationStatus": status.ConfirmationStatus,
					"error":              status.Err,
				}).Debug("Transaction status update")

				// Check if transaction is confirmed
				if status.ConfirmationStatus == rpc.ConfirmationStatusType(commitment) {
					return types.TxDone, nil
				}

				// Proceed to next iteration if transaction is not confirmed yet
				continue
			}

			// If status is nil, check blockhash validity
			IsValidBlockhashResult, err := s.client.IsBlockhashValid(
				ctx,
				txMetadata.Blockhash,
				rpc.CommitmentFinalized,
			)
			if err != nil {
				s.logger.WithError(err).Warn("Failed to check blockhash validity")
				continue
			}

			currentSlot := IsValidBlockhashResult.RPCContext.Context.Slot
			slotDifference := currentSlot - txMetadata.BlockhashSlot
			isValid := IsValidBlockhashResult.Value

			s.logger.WithFields(logrus.Fields{
				"currentSlot":    currentSlot,
				"slotDifference": slotDifference,
				"isValid":        isValid,
				"signature":      sig.String(),
			}).Debug("Checking blockhash validity")

			// Check if we exceeded max valid slots
			if slotDifference > MaxValidSlots {
				// Check if blockhash is still valid
				if isValid {
					s.logger.WithFields(logrus.Fields{
						"slotDifference": slotDifference,
						"maxValidSlots":  MaxValidSlots,
						"signature":      sig.String(),
					}).Warn("Transaction expired: exceeded max valid slots, blockhash is still valid")
					// Proceed to check if transaction blockhash is still valid
					continue
				}

				s.logger.WithFields(logrus.Fields{
					"slotDifference": slotDifference,
					"maxValidSlots":  MaxValidSlots,
					"signature":      sig.String(),
				}).Info("Transaction expired: exceeded max valid slots")

				// Return false to trigger transaction resubmission
				return types.TxNeedsRetry, nil
			}
		}
	}
}
