package dbconfig

import "github.com/pkg/errors"

var (
	ErrChainNotFound   = errors.New("chain not found")
	ErrAgentNotFound   = errors.New("agent not found")
	ErrInvalidChainID  = errors.New("invalid chain id")
	ErrInvalidAgentID  = errors.New("invalid agent id")
	ErrDatabaseConnect = errors.New("failed to connect to database")
)
