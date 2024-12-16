package models

import (
	"github.com/ClipFinance/relay-lib/common/types"
	"time"
)

type Chain struct {
	ID              int64
	ChainID         uint64
	Name            string
	Type            types.ChainType
	ReceiverAddress string
	Active          bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
