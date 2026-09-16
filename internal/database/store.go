package database

import (
	"errors"
	"yasirbin/internal/model"
)

var ErrNotFound = errors.New("document not found")

type Store interface {
	Create(doc *model.Document) error
	GetBySlug(slug string) (*model.Document, error)
	SlugExists(slug string) bool
	CleanExpired() int64
	Close() error
	Driver() string
}
