package utils

import (
	"context"
	"encoding/binary"
	"fmt"
	"strings"

	sol "github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/pkg/errors"
)

// ExtractQuoteIdFromLogs checks if logs contain target log and extracts quote ID from it.
func ExtractQuoteIdFromLogs(logs []string) string {
	// Looking specifically for Memo program logs
	for _, log := range logs {
		// Check if log is from Memo program and contains hex string
		if strings.Contains(log, "Memo (len 66)") && strings.Contains(log, "0x") {
			// Extract the hex string between quotes
			start := strings.Index(log, "\"0x")
			if start == -1 {
				continue
			}
			end := strings.LastIndex(log, "\"")
			if end == -1 || end <= start {
				continue
			}

			quoteId := log[start+1 : end]
			// Validate hex string length (32 bytes = 64 chars + "0x")
			if len(quoteId) == 66 && strings.HasPrefix(quoteId, "0x") {
				return quoteId
			}
		}
	}
	return ""
}

// GetAssociatedTokenAddress returns the token account address for a given token and owner.
// This is a deterministic address that follows Solana's Associated Token Account Program conventions.
func GetAssociatedTokenAddress(tokenMint, owner sol.PublicKey) (sol.PublicKey, error) {

	seeds := [][]byte{
		owner.Bytes(),              // The owner's public key
		sol.TokenProgramID.Bytes(), // The Token Program ID
		tokenMint.Bytes(),          // The mint address of the token
	}

	// Find PDA (Program Derived Address) for the Associated Token Account
	addr, _, err := sol.FindProgramAddress(
		seeds,
		sol.SPLAssociatedTokenAccountProgramID,
	)

	return addr, err
}

// CreateAssociatedTokenAccountInstruction creates the instruction for ATA creation
func CreateAssociatedTokenAccountInstruction(
	payer sol.PublicKey,
	associatedToken sol.PublicKey,
	owner sol.PublicKey,
	mint sol.PublicKey,
	ataProgram sol.PublicKey,
	tokenProgram sol.PublicKey,
) sol.Instruction {
	return sol.NewInstruction(
		ataProgram,
		sol.AccountMetaSlice{
			{PublicKey: payer, IsSigner: true, IsWritable: true},
			{PublicKey: associatedToken, IsSigner: false, IsWritable: true},
			{PublicKey: owner, IsSigner: false, IsWritable: false},
			{PublicKey: mint, IsSigner: false, IsWritable: false},
			{PublicKey: sol.SystemProgramID, IsSigner: false, IsWritable: false},
			{PublicKey: tokenProgram, IsSigner: false, IsWritable: false},
			{PublicKey: sol.SysVarRentPubkey, IsSigner: false, IsWritable: false},
		},
		[]byte{},
	)
}

// createMemoInstruction creates a memo instruction with the given message
func CreateMemoInstruction(message string) sol.Instruction {
	return sol.NewInstruction(
		sol.MemoProgramID,
		sol.AccountMetaSlice{},
		[]byte(message),
	)
}

// CreateTransferInstruction creates a transfer instruction for the given source, destination, owner, and amount
func CreateTransferInstruction(
	source sol.PublicKey, // Source ATA account
	destination sol.PublicKey, // Destination ATA account
	owner sol.PublicKey, // Owner of the source ATA account
	amount uint64, // Amount to transfer
) sol.Instruction {
	// Create instruction data
	data := make([]byte, 9) // 1 byte for instruction code + 8 bytes for amount
	data[0] = 3             // Transfer instruction code
	binary.LittleEndian.PutUint64(data[1:], amount)

	return sol.NewInstruction(
		sol.TokenProgramID,
		sol.AccountMetaSlice{
			{PublicKey: source, IsSigner: false, IsWritable: true},
			{PublicKey: destination, IsSigner: false, IsWritable: true},
			{PublicKey: owner, IsSigner: true, IsWritable: false},
		},
		data,
	)
}

// SimulateTransaction simulates transaction to calculate required compute units
func SimulateTransaction(ctx context.Context, client *rpc.Client, signer sol.PrivateKey, instructions []sol.Instruction, latestBlockHash sol.Hash) (uint64, error) {
	tx, err := sol.NewTransaction(
		instructions,
		latestBlockHash,
		sol.TransactionPayer(signer.PublicKey()),
	)
	if err != nil {
		return 0, errors.Wrap(err, "failed to create transaction")
	}

	// Sign the transaction
	_, err = tx.Sign(func(key sol.PublicKey) *sol.PrivateKey {
		if signer.PublicKey().Equals(key) {
			return &signer
		}
		return nil
	})
	if err != nil {
		return 0, errors.Wrap(err, "failed to sign transaction")
	}

	sim, err := client.SimulateTransaction(ctx, tx)
	if err != nil {
		return 0, errors.Wrap(err, "failed to simulate transaction")
	}

	if sim.Value.Err != nil {
		return 0, fmt.Errorf("simulation failed: %v", sim.Value.Err)
	}

	return *sim.Value.UnitsConsumed, nil
}
