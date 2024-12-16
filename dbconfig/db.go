package dbconfig

import (
	_ "github.com/lib/pq"
)

type DBConfig struct {
	dbConnStr string
}

// NewDBConfig creates a new DBConfig instance with the provided connection string.
func NewDBConfig(connStr string) (*DBConfig, error) {
	return &DBConfig{
		dbConnStr: connStr,
	}, nil
}
