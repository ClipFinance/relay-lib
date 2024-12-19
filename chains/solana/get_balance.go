package solana

import (
	"context"
	"math/big"

	"github.com/ClipFinance/relay-lib/chains/solana/utils"
	sol "github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/pkg/errors"
)

// GetTokenBalance gets token balance for the given address.
// For native token balances, use tokenAddress as empty string or ZeroAddress
//
// Parameters:
// - ctx: the context for managing the request
// - address: the address to check balance for
// - tokenAddress: the token contract address
//
// Returns:
// - *big.Int: the token balance
// - error: an error if the balance check fails
func (s *solana) GetTokenBalance(ctx context.Context, address string, tokenAddress string) (*big.Int, error) {
	userPubKey, err := sol.PublicKeyFromBase58(address)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse userPubKey")
	}

	// Handle native SOL balance
	if tokenAddress == "" || tokenAddress == sol.SystemProgramID.String() {
		balance, err := s.getNativeBalance(ctx, userPubKey)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get native SOL balance")
		}
		return balance, nil
	}

	// Handle SPL token balance
	tokenPubKey, err := sol.PublicKeyFromBase58(tokenAddress)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse tokenAddress")
	}

	// Get ATA
	sourceATA, err := utils.GetAssociatedTokenAddress(tokenPubKey, userPubKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get associated token address")
	}

	// Get SPL token balance
	balance, err := s.getSPLTokenBalance(ctx, sourceATA)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get token balance")
	}

	return balance, nil
}

// getNativeBalance gets native SOL balance
func (s *solana) getNativeBalance(ctx context.Context, account sol.PublicKey) (*big.Int, error) {
	balance, err := s.client.GetBalance(ctx, account, rpc.CommitmentFinalized)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get native balance")
	}

	return big.NewInt(int64(balance.Value)), nil
}

// getSPLTokenBalance gets SPL token balance
func (s *solana) getSPLTokenBalance(ctx context.Context, account sol.PublicKey) (*big.Int, error) {
	balance, err := s.client.GetTokenAccountBalance(ctx, account, rpc.CommitmentFinalized)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get token balance")
	}

	amount, ok := big.NewInt(0).SetString(balance.Value.Amount, 10)
	if !ok {
		return nil, errors.New("failed to parse token balance")
	}

	return amount, nil
}

// checkSufficientBalance checks if account has sufficient balance
func (s *solana) checkSufficientBalance(ctx context.Context, account sol.PublicKey, amount uint64, isNative bool) error {
	var balance *big.Int
	var err error

	if isNative {
		balance, err = s.getNativeBalance(ctx, account)
	} else {
		balance, err = s.getSPLTokenBalance(ctx, account)
	}

	if err != nil {
		return errors.Wrap(err, "failed to get balance")
	}

	amountBigInt := big.NewInt(0).SetUint64(amount)
	if balance.Cmp(amountBigInt) < 0 {
		return errors.New("insufficient balance")
	}

	return nil
}
