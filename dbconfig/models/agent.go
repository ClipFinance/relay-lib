package models

import "time"

type Agent struct {
	ID        int64
	UID       string
	URL       string
	CreatedAt time.Time
	UpdatedAt time.Time
}
