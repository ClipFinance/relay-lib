package models

import (
	"time"
)

type Chain struct {
	ID              int64
	ChainID         uint64
	Name            string
	Type            string
	ReceiverAddress string
	Active          bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
