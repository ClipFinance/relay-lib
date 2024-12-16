package types

import (
	"math/big"
	"time"
)

// RelayData represents the data needed to relay a cross-chain transaction.
type RelayData struct {
	QuoteId       string
	Depositor     string
	Recipient     string
	DestChainId   *big.Int
	InputToken    string
	InputAmount   *big.Int
	OriginChainId *big.Int
	OutputToken   string
	OutputAmount  *big.Int
}

// Intent represents a cross-chain transaction intent with its current state.
type Intent struct {
	ID               int64
	QuoteID          string
	FromChain        uint64
	FromToken        string
	FromAmount       *big.Int
	ToChain          uint64
	ToToken          string
	ToAmount         *big.Int
	UserAddress      string
	RecipientAddress string
	FromTx           string
	FromNonce        uint64
	ToTx             *string
	Status           IntentStatus
	SubStatus        *string
	RequestedAt      time.Time
	FromTxMinedAt    time.Time
	ToTxSetAt        *time.Time
	ToTxMinedAt      *time.Time
	Refund           *bool
	RefundTx         *string
	RefundTxSetAt    *time.Time
	RefundTxMinedAt  *time.Time
	BlockHash        string
	Quorum           int
}

// ConvertIntentToRelayData converts Intent to RelayData.
func (i *Intent) ConvertIntentToRelayData() *RelayData {
	return &RelayData{
		QuoteId:       i.QuoteID,
		Depositor:     i.UserAddress,
		Recipient:     i.RecipientAddress,
		DestChainId:   new(big.Int).SetUint64(i.ToChain),
		InputToken:    i.FromToken,
		InputAmount:   i.FromAmount,
		OriginChainId: new(big.Int).SetUint64(i.FromChain),
		OutputToken:   i.ToToken,
		OutputAmount:  i.ToAmount,
	}
}
