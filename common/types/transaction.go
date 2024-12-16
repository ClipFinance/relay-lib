package types

import (
	"math/big"
)

// Transaction represents a blockchain transaction.
//
// Fields:
// - Hash: the hash of the transaction.
// - From: the address from which the transaction is sent.
// - To: the address to which the transaction is sent.
// - FromAmount: the amount sent from the sender's address.
// - ToAmount: the amount received at the recipient's address.
// - Token: the token involved in the transaction.
// - Nonce: the nonce of the transaction.
// - ChainID: the unique identifier for the chain where the transaction occurred.
// - QuoteID: the identifier for the quote associated with the transaction.
type Transaction struct {
	Hash       string
	From       string
	To         string
	FromAmount string
	ToAmount   string
	Token      string
	Nonce      uint64
	ChainID    uint64
	QuoteID    string
}

// TransactionIntent represents the intent to perform a blockchain transaction.
//
// Fields:
// - QuoteID: the identifier for the quote associated with the transaction intent.
// - FromChain: the unique identifier for the chain from which the transaction is sent.
// - FromToken: the token sent from the sender's address.
// - FromAmount: the amount sent from the sender's address.
// - ToChain: the unique identifier for the chain to which the transaction is sent.
// - ToToken: the token received at the recipient's address.
// - ToAmount: the amount received at the recipient's address.
// - UserAddress: the address of the user initiating the transaction.
// - RecipientAddress: the address of the recipient of the transaction.
type TransactionIntent struct {
	QuoteID          string
	FromChain        uint64
	FromToken        string
	FromAmount       *big.Int
	ToChain          uint64
	ToToken          string
	ToAmount         *big.Int
	UserAddress      string
	RecipientAddress string
}
