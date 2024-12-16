package types

// Intent represents a cross-chain transaction intent with its current state
type Intent struct {
	QuoteID          string  // Unique identifier of the quote
	FromChain        int     // Source chain ID
	FromToken        string  // Source token address
	FromAmount       string  // Amount to be sent from source chain
	ToChain          int     // Destination chain ID
	ToToken          string  // Destination token address
	ToAmount         string  // Amount to be received on destination chain
	UserAddress      string  // User's wallet address
	RecipientAddress string  // Recipient's wallet address
	FromTx           string  // Transaction hash on source chain
	FromNonce        uint64  // Nonce of the transaction on source chain
	Status           string  // Current status of the intent
	SubStatus        *string // Additional status details
	QuoteRequestedAt string  // Timestamp when quote was requested
	FromTxMinedAt    string  // Timestamp when source transaction was mined
	ToTxSentAt       *string // Timestamp when destination transaction was sent
	ToTxMinedAt      *string // Timestamp when destination transaction was mined
	Refund           bool    // Whether a refund was required
	RefundTx         *string // Refund transaction hash if any
	RefundTxSetAt    *string // Timestamp when refund was initiated
	RefundTxMinedAt  *string // Timestamp when refund was completed
	BlockHash        string  // Block hash where the intent was created
	Quorum           int     // Number of confirmations received
}
