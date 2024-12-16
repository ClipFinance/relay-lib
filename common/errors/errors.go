package errors

import "github.com/pkg/errors"

var (
	ErrChainNotFound      = errors.New("chain not found")
	ErrAgentNotFound      = errors.New("agent not found")
	ErrInvalidChainID     = errors.New("invalid chain id")
	ErrInvalidAgentID     = errors.New("invalid agent id")
	ErrDatabaseConnect    = errors.New("failed to connect to database")
	ErrInvalidConfig      = errors.New("invalid chain configuration")
	ErrChainExists        = errors.New("chain already exists in registry")
	ErrFactoryNotProvided = errors.New("chain factory not provided")
	ErrInvalidChainType   = errors.New("invalid chain type")
	ErrNotImplemented     = errors.New("functionality not implemented")
)
