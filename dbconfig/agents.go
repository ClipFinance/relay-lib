package dbconfig

import (
	"context"
	"database/sql"
	"github.com/ClipFinance/relay-lib/dbconfig/models"
)

// GetAgentByUID returns an agent by its UID from the database or an error if not found.
func (r *DBConfig) GetAgentByUID(ctx context.Context, uid string) (*models.Agent, error) {
	if uid == "" {
		return nil, ErrInvalidAgentID
	}

	db, err := sql.Open("postgres", r.dbConnStr)
	if err != nil {
		return nil, ErrDatabaseConnect
	}
	defer db.Close()

	var agent models.Agent
	var url sql.NullString

	err = db.QueryRowContext(ctx, `
       SELECT 
           id,
           uid,
           url,
           created_at,
           updated_at
       FROM agents
       WHERE uid = $1
    `, uid).Scan(
		&agent.ID,
		&agent.UID,
		&url,
		&agent.CreatedAt,
		&agent.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrAgentNotFound
	}

	if err != nil {
		return nil, ErrDatabaseConnect
	}

	if url.Valid {
		agent.URL = url.String
	}

	return &agent, nil
}
