package types

// TransactionStatus represents the status of a transaction.
type TransactionStatus int

const (
	// TxFailed represents a failed transaction.
	TxFailed TransactionStatus = iota
	// TxDone represents a successfully confirmed transaction.
	TxDone
	// TxNeedsRetry represents a transaction that needs to be retried.
	TxNeedsRetry
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
// - Metadata: additional metadata associated with the transaction.
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
	Metadata   interface{}
}

// Parameters represents transaction parameters.
//
// Fields:
// - FromChain: the chain ID from which the transaction is sent.
// - ToChain: the chain ID to which the transaction is sent.
// - FromToken: the token address from which the transaction is sent.
// - ToToken: the token address to which the transaction is sent.
// - Amount: the amount of tokens to be transferred.
// - UserAddress: the address of the user initiating the transaction.
// - Receiver: the address of the receiver of the transaction.
type Parameters struct {
	FromChain   int    `json:"fromChain"`
	ToChain     int    `json:"toChain"`
	FromToken   string `json:"fromToken"`
	ToToken     string `json:"toToken"`
	Amount      string `json:"amount"`
	UserAddress string `json:"userAddress"`
	Receiver    string `json:"receiver"`
}

// Quote represents quote information.
//
// Fields:
// - QuoteID: the unique identifier for the quote.
// - ToAmount: the amount to be received in the quote.
// - ToAmountMin: the minimum amount to be received in the quote.
// - ExecutionDuration: the duration for the quote execution.
// - GasEstimate: the estimated gas for the quote.
// - RequestedAt: the timestamp when the quote was requested.
// - Deadline: the deadline for the quote.
// - Parameters: the transaction parameters associated with the quote.
type Quote struct {
	QuoteID           string     `json:"quoteId"`
	ToAmount          string     `json:"toAmount"`
	ToAmountMin       string     `json:"toAmountMin"`
	ExecutionDuration int        `json:"executionDuration"`
	GasEstimate       int        `json:"gasEstimate"`
	RequestedAt       string     `json:"requestedAt"`
	Deadline          string     `json:"deadline"`
	Parameters        Parameters `json:"parameters"`
}
