package dbconfig

import (
	"context"
	"database/sql"
	"github.com/ClipFinance/relay-lib/dbconfig/models"
)

// GetAgentByUID returns an agent by its UID from the database or an error if not found.
//
// Parameters:
// - ctx: the context for managing the request.
// - uid: the unique identifier for the agent.
//
// Returns:
// - *models.Agent: a pointer to the Agent model.
// - error: an error if the database operation fails or the agent is not found.
func (dc *DBConfig) GetAgentByUID(ctx context.Context, uid string) (*models.Agent, error) {
	if uid == "" {
		return nil, ErrInvalidAgentID
	}

	db, err := sql.Open("postgres", dc.dbConnStr)
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
