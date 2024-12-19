package solana

import (
	"context"

	"github.com/ClipFinance/relay-lib/chains/solana/utils"
	"github.com/ClipFinance/relay-lib/common/types"
	sol "github.com/gagliardetto/solana-go"
	computebudget "github.com/gagliardetto/solana-go/programs/compute-budget"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// SendAsset sends an asset to a recipient address on the chain.
func (s *solana) SendAsset(ctx context.Context, intent *types.Intent) (*types.Transaction, error) {
	userPubKey, err := sol.PublicKeyFromBase58(intent.RecipientAddress)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse userPubKey")
	}
	signerPubKey := s.signer.PublicKey()

	// Get recent blockhash
	latestBlockhashResult, err := s.client.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get latest blockhash")
	}
	latestBlockhash := latestBlockhashResult.Value.Blockhash

	// Create instructions
	instructions, err := s.createTransferInstructions(ctx, intent, latestBlockhash)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create instructions")
	}

	// Estimate transaction cost
	cost, err := s.estimateTransactionCost(ctx, instructions, signerPubKey, latestBlockhash)
	if err != nil {
		return nil, errors.Wrap(err, "failed to estimate gas")
	}
	s.logger.WithFields(logrus.Fields{
		"quoteID":   intent.QuoteID,
		"cost":      cost,
		"costInSol": utils.LamportsToSol(cost),
	}).Info("Transaction cost estimated")

	// Send transaction
	sig, err := s.sendTransaction(ctx, instructions, latestBlockhash)
	if err != nil {
		return nil, errors.Wrap(err, "failed to send transaction")
	}

	// Return transaction with defined data
	return &types.Transaction{
		Hash:     sig.String(),
		From:     signerPubKey.String(),
		To:       userPubKey.String(),
		ToAmount: "0", // TODO: decide if we need value for token transfers
		Token:    intent.ToToken,
		ChainID:  s.config.ChainID,
		QuoteID:  intent.QuoteID,
		// Nonce:   0, // TODO: decide if we need nonce
		Metadata: utils.SolanaMetadata{
			Blockhash:     latestBlockhash,
			BlockhashSlot: latestBlockhashResult.Context.Slot,
		},
	}, nil
}

func (s *solana) createTransferInstructions(ctx context.Context, intent *types.Intent, latestBlockHash sol.Hash) ([]sol.Instruction, error) {
	// Check if the intent is for a native asset.
	toTokenPubKey, err := sol.PublicKeyFromBase58(intent.ToToken)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse toToken")
	}

	var instructions []sol.Instruction

	if toTokenPubKey == sol.SystemProgramID {
		instructions, err = s.createFillRelayNativelyInstructions(ctx, intent)
		if err != nil {
			return nil, errors.Wrap(err, "failed to createFillRelayNativelyInstructions")
		}
	} else {
		instructions, err = s.createFillRelayWithTokenInstructions(ctx, intent, latestBlockHash)
		if err != nil {
			return nil, errors.Wrap(err, "failed to createFillRelayWithTokenInstructions")
		}
	}

	return instructions, nil

}

func (s *solana) createFillRelayNativelyInstructions(ctx context.Context, intent *types.Intent) ([]sol.Instruction, error) {
	err := errors.New("not implemented")
	return nil, errors.Wrap(err, "createFillRelayNativelyInstructions not implemented")
}

func (s *solana) createFillRelayWithTokenInstructions(ctx context.Context, intent *types.Intent, latestBlockHash sol.Hash) ([]sol.Instruction, error) {
	amount := intent.ToAmount.Uint64()
	quoteID := intent.QuoteID
	toTokenPubKey, err := sol.PublicKeyFromBase58(intent.ToToken)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse toToken")
	}
	userPubKey, err := sol.PublicKeyFromBase58(intent.RecipientAddress)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse userPubKey")
	}
	signerPubKey := s.signer.PublicKey()

	// Initialize base instructions
	basicInstructions := make([]sol.Instruction, 0)

	// Check ATA and create if needed
	userCreateATAInstruction, err := s.checkAndCreateATAInstructionIfNotExist(
		ctx,
		signerPubKey,
		toTokenPubKey,
		userPubKey,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check and create ATA instruction")
	}

	// Append instruction if it exists
	if userCreateATAInstruction != nil {
		basicInstructions = append(basicInstructions, userCreateATAInstruction)
	}

	// Get ATAs
	sourceATA, err := utils.GetAssociatedTokenAddress(toTokenPubKey, signerPubKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get associated token address for signer")
	}
	destATA, err := utils.GetAssociatedTokenAddress(toTokenPubKey, userPubKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get associated token address for user")
	}

	// Check source balance
	err = s.checkSufficientBalance(ctx, sourceATA, amount, false)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check balance")
	}

	// Create transfer instruction
	transferInstruction := utils.CreateTransferInstruction(
		sourceATA,
		destATA,
		signerPubKey,
		amount,
	)
	basicInstructions = append(basicInstructions, transferInstruction)

	// Add memo instruction
	memoInstruction := utils.CreateMemoInstruction(quoteID)
	basicInstructions = append(basicInstructions, memoInstruction)

	if err != nil {
		return nil, errors.Wrap(err, "failed to create basic instructions")
	}

	// Simulate transaction to get compute units
	computeUnits, err := utils.SimulateTransaction(ctx, s.client, s.signer, basicInstructions, latestBlockHash)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to simulate transaction, using default compute units")
		computeUnits = defaultComputeUnits
	}

	// Add buffer to compute units
	computeUnits = (computeUnits * computeUnitBuffer) / 100
	s.logger.WithField("computeUnits", computeUnits).Debug("Computed units with buffer")

	// Get priority fee
	priorityFee := s.getPriorityFee(ctx)
	s.logger.WithFields(logrus.Fields{
		"priorityFee": priorityFee,
		"totalFee":    priorityFee * computeUnits,
	}).Debug("Priority fee details")

	// Create final instructions with compute budget
	finalInstructions := make([]sol.Instruction, 0)

	// Add compute unit limit instruction
	setComputeUnitLimitIx, err := computebudget.NewSetComputeUnitLimitInstruction(uint32(computeUnits)).ValidateAndBuild()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create compute unit limit instruction")
	}
	finalInstructions = append(finalInstructions, setComputeUnitLimitIx)

	// Add priority fee instruction
	setPriorityFeeIx, err := computebudget.NewSetComputeUnitPriceInstruction(priorityFee).ValidateAndBuild()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create priority fee instruction")
	}
	finalInstructions = append(finalInstructions, setPriorityFeeIx)

	// Add all basic instructions
	finalInstructions = append(finalInstructions, basicInstructions...)

	return finalInstructions, nil
}

// CheckAndCreateATAInstructionIfNotExist returns the instruction to create an associated token account if it doesn't exist
func (s *solana) checkAndCreateATAInstructionIfNotExist(
	ctx context.Context,
	payer sol.PublicKey,
	mint sol.PublicKey,
	owner sol.PublicKey,
) (sol.Instruction, error) {
	addr, err := utils.GetAssociatedTokenAddress(mint, owner)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get associated token address")
	}

	// Check if account exists
	acc, err := s.client.GetAccountInfo(ctx, addr)
	if err != nil && err.Error() != "not found" { // skip not found error
		return nil, errors.Wrap(err, "failed to get account info")
	}

	if acc == nil {
		// Return associated token account if it doesn't exist
		instruction := utils.CreateAssociatedTokenAccountInstruction(
			payer,
			addr,
			owner,
			mint,
			sol.SPLAssociatedTokenAccountProgramID,
			sol.TokenProgramID,
		)

		return instruction, nil
	}

	return nil, nil
}

// sendTransaction sends a transaction with multiple instructions
func (s *solana) sendTransaction(
	ctx context.Context,
	instructions []sol.Instruction,
	recentBlockHash sol.Hash,
) (sol.Signature, error) {
	tx, err := sol.NewTransaction(
		instructions,
		recentBlockHash,
		sol.TransactionPayer(s.signer.PublicKey()),
	)
	if err != nil {
		return sol.Signature{}, errors.Wrap(err, "failed to create transaction")
	}

	// Sign the transaction
	_, err = tx.Sign(func(key sol.PublicKey) *sol.PrivateKey {
		if s.signer.PublicKey().Equals(key) {
			return &s.signer
		}

		return nil
	})
	if err != nil {
		return sol.Signature{}, errors.Wrap(err, "failed to sign transaction")
	}

	// Send transaction
	sig, err := s.client.SendTransactionWithOpts(ctx, tx, rpc.TransactionOpts{
		SkipPreflight:       false,
		PreflightCommitment: rpc.CommitmentProcessed,
	})
	if err != nil {
		return sol.Signature{}, errors.Wrap(err, "failed to send transaction")
	}

	return sig, nil
}
