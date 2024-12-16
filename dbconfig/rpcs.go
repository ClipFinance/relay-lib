package dbconfig

import (
	"context"
	"database/sql"
	"github.com/ClipFinance/relay-lib/dbconfig/models"
)

// GetRPCsByChainID returns all RPCs for a given chain ID from the database, optionally filtering by active status.
//
// Parameters:
// - ctx: the context for managing the request.
// - chainID: the unique identifier for the chain.
// - activeOnly: a boolean flag to filter only active RPCs.
//
// Returns:
// - []models.RPC: a slice of RPC models.
// - error: an error if the database operation fails.
func (r *DBConfig) GetRPCsByChainID(ctx context.Context, chainID uint64, activeOnly bool) ([]models.RPC, error) {
	if chainID == 0 {
		return nil, ErrInvalidChainID
	}

	db, err := sql.Open("postgres", r.dbConnStr)
	if err != nil {
		return nil, ErrDatabaseConnect
	}
	defer db.Close()

	query := `
  		SELECT 
  			id,
			chain_id,
			url,
			provider,
			agent_id,
			active,
			created_at,
			updated_at
		FROM rpcs
		WHERE chain_id = $1
   `

	args := []interface{}{chainID}
	argCount := 1

	if activeOnly {
		query += " AND active = $2"
		args = append(args, true)
		argCount++
	}

	query += " ORDER BY created_at DESC"

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, ErrDatabaseConnect
	}
	defer rows.Close()

	var rpcs []models.RPC
	for rows.Next() {
		var rpc models.RPC
		var agentID sql.NullInt64
		var provider sql.NullString

		err := rows.Scan(
			&rpc.ID,
			&rpc.ChainID,
			&rpc.URL,
			&provider,
			&agentID,
			&rpc.Active,
			&rpc.CreatedAt,
			&rpc.UpdatedAt,
		)
		if err != nil {
			return nil, ErrDatabaseConnect
		}

		if provider.Valid {
			rpc.Provider = provider.String
		}
		if agentID.Valid {
			rpc.AgentID = agentID.Int64
		}

		rpcs = append(rpcs, rpc)
	}

	if err = rows.Err(); err != nil {
		return nil, ErrDatabaseConnect
	}

	return rpcs, nil
}

// GetAgentRPCs returns all RPCs for a given agent ID from the database, optionally filtering by active status.
//
// Parameters:
// - ctx: the context for managing the request.
// - agentID: the unique identifier for the agent.
// - activeOnly: a boolean flag to filter only active RPCs.
//
// Returns:
// - []models.RPC: a slice of RPC models.
// - error: an error if the database operation fails.
func (r *DBConfig) GetAgentRPCs(ctx context.Context, agentID int64, activeOnly bool) ([]models.RPC, error) {
	if agentID == 0 {
		return nil, ErrInvalidAgentID
	}

	db, err := sql.Open("postgres", r.dbConnStr)
	if err != nil {
		return nil, ErrDatabaseConnect
	}
	defer db.Close()

	query := `
       SELECT 
           r.id,
           r.chain_id,
           r.url,
           r.provider,
           r.agent_id,
           r.active,
           r.created_at,
           r.updated_at
       FROM rpcs r
       JOIN chains c ON c.chain_id = r.chain_id
       WHERE r.agent_id = $1
    `

	args := []interface{}{agentID}
	argCount := 1

	if activeOnly {
		query += " AND r.active = $2 AND c.active = $2"
		args = append(args, true)
		argCount++
	}

	query += " ORDER BY r.chain_id ASC, r.created_at DESC"

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, ErrDatabaseConnect
	}
	defer rows.Close()

	var rpcs []models.RPC
	for rows.Next() {
		var rpc models.RPC
		var provider sql.NullString

		err := rows.Scan(
			&rpc.ID,
			&rpc.ChainID,
			&rpc.URL,
			&provider,
			&rpc.AgentID,
			&rpc.Active,
			&rpc.CreatedAt,
			&rpc.UpdatedAt,
		)
		if err != nil {
			return nil, ErrDatabaseConnect
		}

		if provider.Valid {
			rpc.Provider = provider.String
		}

		rpcs = append(rpcs, rpc)
	}

	if err = rows.Err(); err != nil {
		return nil, ErrDatabaseConnect
	}

	return rpcs, nil
}
