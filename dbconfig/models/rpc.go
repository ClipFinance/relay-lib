package models

import "time"

type RPC struct {
	ID        int64
	ChainID   uint64
	URL       string
	Provider  string
	AgentID   int64
	Active    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
