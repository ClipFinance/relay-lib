package dbconfig

import (
	"context"
	"database/sql"
	"github.com/pkg/errors"
	"math/big"
)

// GetNativeTokenAddress returns the native token address for the given chain ID.
func (dc *DBConfig) GetNativeTokenAddress(ctx context.Context, chainID uint64) (string, error) {
	db, err := sql.Open("postgres", dc.dbConnStr)
	if err != nil {
		return "", errors.Wrap(err, "failed to connect to database")
	}
	defer db.Close()

	var address string
	err = db.QueryRowContext(ctx, `
        SELECT address 
        FROM chain_tokens 
        WHERE chain_id = $1 AND native = true
    `, chainID).Scan(&address)
	if err != nil {
		return "", errors.Wrap(err, "failed to get native token address")
	}

	return address, nil
}

// UpdateBalance updates token balance in database for the given chain ID and token address.
func (dc *DBConfig) UpdateBalance(ctx context.Context, chainID uint64, tokenAddress string, balance *big.Int) error {
	db, err := sql.Open("postgres", dc.dbConnStr)
	if err != nil {
		return errors.Wrap(err, "failed to connect to database")
	}
	defer db.Close()

	// Get token decimals from DB
	var decimals int
	err = db.QueryRowContext(ctx, `
       SELECT decimals 
       FROM chain_tokens 
       WHERE chain_id = $1 AND address = $2
   `, chainID, tokenAddress).Scan(&decimals)
	if err != nil {
		return errors.Wrap(err, "failed to get token decimals")
	}

	// Calculate formatted balance
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	balanceFloat := new(big.Float).SetInt(balance)
	divisorFloat := new(big.Float).SetInt(divisor)
	formattedBalance := new(big.Float)
	formattedBalance.Quo(balanceFloat, divisorFloat)

	// Update balance in DB
	query := `
       UPDATE chain_tokens 
       SET 
           balance = $1,
           balance_formatted = $2,
           updated_at = NOW()
       WHERE chain_id = $3 AND address = $4
   `

	result, err := db.ExecContext(ctx, query,
		balance.String(),
		formattedBalance.String(),
		chainID,
		tokenAddress,
	)
	if err != nil {
		return errors.Wrap(err, "failed to update token balance")
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}

	if affected == 0 {
		return errors.New("token not found in database")
	}

	return nil
}
