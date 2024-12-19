package evm

import (
	"context"
	"github.com/ClipFinance/relay-lib/chains/evm/utils"
	"math/big"

	"github.com/ClipFinance/relay-lib/common/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
)

// ValidateTransaction validates a transaction based on the quote and the event.
//
// Parameters:
// - ctx: the context for managing the request.
// - quote: the quote containing transaction details.
// - event: the event containing chain event details.
//
// Returns:
// - error: an error if the transaction validation fails.
func (e *evm) ValidateTransaction(ctx context.Context, quote *types.Quote, event types.ChainEvent) error {
	e.clientMutex.RLock()
	client := e.client
	e.clientMutex.RUnlock()

	if client == nil {
		return errors.New("client not initialized")
	}

	txHash := common.HexToHash(event.TransactionHash)
	tx, _, err := client.TransactionByHash(ctx, txHash)
	if err != nil {
		return errors.Wrap(err, "failed to get transaction details")
	}

	e.solverAddressMutex.RLock()
	solverAddress := e.solverAddress
	e.solverAddressMutex.RUnlock()

	if err := e.validateTransaction(quote, tx, event, solverAddress.Hex()); err != nil {
		return errors.Wrap(err, "transaction validation failed")
	}

	return nil
}

// validateTransaction validates transaction details against quote parameters
func (e *evm) validateTransaction(quote *types.Quote, tx *ethtypes.Transaction, event types.ChainEvent, solverAddr string) error {
	// Validate transaction type (native token transfer or ERC20 token transfer)
	if quote.Parameters.FromToken == utils.ZeroAddress {
		// Validate native token transfer transaction
		return e.validateNativeTransfer(quote, tx)
	}

	// Validate ERC20 token transfer transaction
	return e.validateTokenTransfer(quote, event, solverAddr)
}

// validateNativeTransfer validates native token transfer transaction details
func (e *evm) validateNativeTransfer(quote *types.Quote, tx *ethtypes.Transaction) error {
	// Validate exact amount match
	amount := new(big.Int)
	amount.SetString(quote.Parameters.Amount, 10)
	if tx.Value().Cmp(amount) != 0 {
		return errors.New("amount mismatch")
	}

	// Get transaction sender address
	signer := ethtypes.LatestSignerForChainID(big.NewInt(int64(quote.Parameters.FromChain)))
	sender, err := ethtypes.Sender(signer, tx)
	if err != nil {
		return errors.Wrap(err, "failed to get transaction sender")
	}

	// Validate sender address match
	if sender.Hex() != quote.Parameters.UserAddress {
		return errors.New("sender address mismatch")
	}

	// Validate chain ID match
	if tx.ChainId().Int64() != int64(quote.Parameters.FromChain) {
		return errors.New("chain ID mismatch")
	}

	return nil
}

// validateTokenTransfer validates ERC20 token transfer
func (e *evm) validateTokenTransfer(quote *types.Quote, event types.ChainEvent, solverAddr string) error {
	tokenAddr := common.HexToAddress(quote.Parameters.FromToken)
	userAddr := common.HexToAddress(quote.Parameters.UserAddress)
	amount := new(big.Int)
	amount.SetString(quote.Parameters.Amount, 10)

	// Check if this is a transfer event from the correct token contract
	if event.FromTokenAddr != tokenAddr.Hex() {
		return errors.New("token address mismatch")
	}

	// Check sender address (user address)
	if event.FromAddress != userAddr.Hex() {
		return errors.New("sender address mismatch")
	}

	// Check receiver address (solver address)
	if event.ToAddress != solverAddr {
		return errors.New("receiver address mismatch")
	}

	// Check amount transferred
	transferAmount, success := new(big.Int).SetString(event.TransactionAmount, 10)
	if !success {
		return errors.New("failed to parse transfer amount")
	}

	if transferAmount.Cmp(amount) != 0 {
		return errors.New("transfer amount mismatch")
	}

	return nil
}
