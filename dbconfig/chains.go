package dbconfig

import (
	"context"
	"database/sql"
	"github.com/ClipFinance/relay-lib/common/errors"
	"github.com/ClipFinance/relay-lib/common/types"
	"github.com/ClipFinance/relay-lib/dbconfig/models"
	"strings"
)

// GetChains returns all chains from the database, optionally filtering by active status.
func (r *DBConfig) GetChains(ctx context.Context, activeOnly bool) ([]models.Chain, error) {
	db, err := sql.Open("postgres", r.dbConnStr)
	if err != nil {
		return nil, errors.ErrDatabaseConnect
	}
	defer db.Close()

	query := `
      SELECT 
          id,
          chain_id,
          name,
          chain_type,
          receiver_address,
          active,
          created_at,
          updated_at
      FROM chains
  `

	var args []interface{}
	if activeOnly {
		query += " WHERE active = $1"
		args = append(args, true)
	}

	query += " ORDER BY chain_id ASC"

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.ErrDatabaseConnect
	}
	defer rows.Close()

	var chains []models.Chain
	for rows.Next() {
		var chain models.Chain
		var receiverAddress sql.NullString
		var chainType sql.NullString

		err := rows.Scan(
			&chain.ID,
			&chain.ChainID,
			&chain.Name,
			&chainType,
			&receiverAddress,
			&chain.Active,
			&chain.CreatedAt,
			&chain.UpdatedAt,
		)
		if err != nil {
			return nil, errors.ErrDatabaseConnect
		}

		if receiverAddress.Valid {
			chain.ReceiverAddress = receiverAddress.String
		}
		if chainType.Valid {
			chain.Type = types.ChainType(strings.ToUpper(chainType.String))
		}

		chains = append(chains, chain)
	}

	if err = rows.Err(); err != nil {
		return nil, errors.ErrDatabaseConnect
	}

	return chains, nil
}

func (r *DBConfig) GetChainByID(ctx context.Context, chainID uint64) (*models.Chain, error) {
	if chainID == 0 {
		return nil, errors.ErrInvalidChainID
	}

	db, err := sql.Open("postgres", r.dbConnStr)
	if err != nil {
		return nil, errors.ErrDatabaseConnect
	}
	defer db.Close()

	var chain models.Chain
	var receiverAddress sql.NullString
	var chainType sql.NullString

	err = db.QueryRowContext(ctx, `
       SELECT 
           id,
           chain_id,
           name,
           chain_type,
           receiver_address,
           active,
           created_at,
           updated_at
       FROM chains
       WHERE chain_id = $1
    `, chainID).Scan(
		&chain.ID,
		&chain.ChainID,
		&chain.Name,
		&chainType,
		&receiverAddress,
		&chain.Active,
		&chain.CreatedAt,
		&chain.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.ErrChainNotFound
	}

	if err != nil {
		return nil, errors.ErrDatabaseConnect
	}

	if receiverAddress.Valid {
		chain.ReceiverAddress = receiverAddress.String
	}
	if chainType.Valid {
		chain.Type = types.ChainType(strings.ToUpper(chainType.String))
	}

	return &chain, nil
}
