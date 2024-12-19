package utils

import (
	sol "github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

type GetParsedTransactionOptsV2 struct {
	Commitment                     rpc.CommitmentType `json:"commitment,omitempty"`
	MaxSupportedTransactionVersion uint64             `json:"maxSupportedTransactionVersion,omitempty"`
}

type InnerInstruction struct {
	Index        uint8                     `json:"index"`
	Instructions []ParsedInstructionInfoV2 `json:"instructions"`
}

type ParsedInstructionInfoV2 struct {
	ProgramIDIndex   uint16            `json:"programIdIndex"`
	PublicKey        sol.PublicKey     `json:"PublicKey"`
	Accounts         []uint16          `json:"accounts"`
	Data             string            `json:"data"`
	InnerInstruction *InnerInstruction `json:"innerInstructions"`
}

type SolanaMetadata struct {
	Blockhash     sol.Hash
	BlockhashSlot uint64
}
