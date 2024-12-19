package evm

import (
	"context"
	"github.com/ClipFinance/relay-lib/chains/evm/generated"
	"github.com/ClipFinance/relay-lib/chains/evm/utils"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"math/big"
	"strings"
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
func (e *evm) GetTokenBalance(ctx context.Context, address string, tokenAddress string) (*big.Int, error) {
	e.clientMutex.RLock()
	client := e.client
	e.clientMutex.RUnlock()

	if client == nil {
		return nil, errors.New("client not initialized")
	}

	// Check if requesting native token balance
	if tokenAddress == "" || tokenAddress == utils.ZeroAddress {
		balance, err := client.BalanceAt(ctx, common.HexToAddress(address), nil)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get native token balance")
		}
		return balance, nil
	}

	// For ERC20 tokens
	tokenAbi, err := abi.JSON(strings.NewReader(generated.ERC20ABI))
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse token ABI")
	}

	data, err := tokenAbi.Pack("balanceOf", common.HexToAddress(address))
	if err != nil {
		return nil, errors.Wrap(err, "failed to pack balanceOf data")
	}

	tokenAddr := common.HexToAddress(tokenAddress)
	result, err := client.CallContract(ctx, ethereum.CallMsg{
		To:   &tokenAddr,
		Data: data,
	}, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to call balanceOf")
	}

	if len(result) == 0 {
		return nil, errors.New("empty result from balanceOf call")
	}

	balance := new(big.Int)
	balance.SetBytes(result)

	return balance, nil
}
