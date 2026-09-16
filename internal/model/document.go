package model

import "time"

type Document struct {
	ID        int64
	Slug      string
	Content   string
	Password  string // bcrypt hash, empty if no password
	CreatedAt time.Time
	ExpiresAt *time.Time // nil = never
}
