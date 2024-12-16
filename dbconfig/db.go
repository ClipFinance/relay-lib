package dbconfig

import (
	_ "github.com/lib/pq"
)

type DBConfig struct {
	dbConnStr string
}

// NewDBConfig creates a new DBConfig instance with the provided connection string.
//
// Parameters:
// - connStr: the database connection string.
//
// Returns:
// - *DBConfig: a pointer to the newly created DBConfig instance.
// - error: an error if the creation of the DBConfig instance fails.
func NewDBConfig(connStr string) (*DBConfig, error) {
	return &DBConfig{
		dbConnStr: connStr,
	}, nil
}
