package solana

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ClipFinance/relay-lib/chains/solana/utils"
	"github.com/ClipFinance/relay-lib/common/types"
	sol "github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/pkg/errors"
)

// ValidateTransaction validates a transaction based on the quote and the event
func (s *solana) ValidateTransaction(ctx context.Context, quote *types.Quote, event types.ChainEvent) error {
	// Get transaction
	tx, err := s.getTransaction(ctx, event.TransactionHash)
	if err != nil {
		return errors.Wrap(err, "failed to get transaction")
	}

	// Validate using token balances
	if tx.Meta == nil || len(tx.Meta.PreTokenBalances) == 0 || len(tx.Meta.PostTokenBalances) == 0 {
		return errors.New("transaction meta data is missing or incomplete")
	}

	// Find source and destination accounts in token balances
	var sourcePreBalance, sourcePostBalance *rpc.TokenBalance
	var destPreBalance, destPostBalance *rpc.TokenBalance

	// Parse public keys
	userPubkey, err := sol.PublicKeyFromBase58(quote.Parameters.UserAddress)
	if err != nil {
		return fmt.Errorf("failed to parse user address: %v", err)
	}

	solverPubkey, err := sol.PublicKeyFromBase58(s.solverAddress)
	if err != nil {
		return fmt.Errorf("failed to parse solver address: %v", err)
	}

	tokenPubkey, err := sol.PublicKeyFromBase58(quote.Parameters.FromToken)
	if err != nil {
		return fmt.Errorf("failed to parse token address: %v", err)
	}

	for _, balance := range tx.Meta.PreTokenBalances {
		// Skip not quote tokens
		if balance.Mint.ToPointer().String() != tokenPubkey.ToPointer().String() {
			continue
		}

		if balance.Owner.ToPointer().String() == userPubkey.ToPointer().String() {
			sourcePreBalance = &balance
		}
		if balance.Owner.ToPointer().String() == solverPubkey.ToPointer().String() { // Check for Solver address instead of quote.Parameters.Receiver
			destPreBalance = &balance
		}
	}

	for _, balance := range tx.Meta.PostTokenBalances {
		// Skip not quote tokens
		if balance.Mint.ToPointer().String() != tokenPubkey.ToPointer().String() {
			continue
		}

		if balance.Owner.ToPointer().String() == userPubkey.ToPointer().String() {
			sourcePostBalance = &balance
		}
		if balance.Owner.ToPointer().String() == solverPubkey.ToPointer().String() { // Check for Solver address instead of quote.Parameters.Receiver
			destPostBalance = &balance
		}
	}

	if sourcePreBalance == nil || sourcePostBalance == nil {
		return errors.New("source account balances not found")
	}
	if destPreBalance == nil || destPostBalance == nil {
		return errors.New("solver account balances not found")
	}

	// Verify the transfer amount
	expectedAmount := new(big.Int)
	expectedAmount.SetString(quote.Parameters.Amount, 10)

	// Calculate actual transferred amount
	preAmount := new(big.Int)
	preAmount.SetString(destPreBalance.UiTokenAmount.Amount, 10)
	postAmount := new(big.Int)
	postAmount.SetString(destPostBalance.UiTokenAmount.Amount, 10)
	receivedAmount := new(big.Int).Sub(postAmount, preAmount)

	if receivedAmount.Cmp(expectedAmount) != 0 {
		return fmt.Errorf("amount mismatch: expected %s, got %s",
			expectedAmount.String(), receivedAmount.String())
	}

	// All checks passed
	return nil
}

func (s *solana) getTransaction(ctx context.Context, transactionHash string) (*utils.GetParsedTransactionResultV2, error) {
	txSig, err := sol.SignatureFromBase58(transactionHash)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse transaction signature")
	}

	tx, err := utils.GetParsedTransactionV2(ctx, s.client, txSig, &utils.GetParsedTransactionOptsV2{
		Commitment:                     rpc.CommitmentConfirmed,
		MaxSupportedTransactionVersion: 0,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch transaction: %w", err)
	}

	// Check if the transaction is valid
	if tx == nil || len(tx.Transaction.Signatures) == 0 {
		return tx, errors.New("invalid transaction data")
	}

	return tx, nil
}
