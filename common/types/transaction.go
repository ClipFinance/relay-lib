package types

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

// Parameters represents transaction parameters
type Parameters struct {
	FromChain   int    `json:"fromChain"`
	ToChain     int    `json:"toChain"`
	FromToken   string `json:"fromToken"`
	ToToken     string `json:"toToken"`
	Amount      string `json:"amount"`
	UserAddress string `json:"userAddress"`
	Receiver    string `json:"receiver"`
}

// Quote represents quote information
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
