package dbconfig

import (
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

type DBConfig struct {
	logger    *logrus.Logger
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
func NewDBConfig(connStr string, logger *logrus.Logger) *DBConfig {
	return &DBConfig{
		logger:    logger,
		dbConnStr: connStr,
	}
}
