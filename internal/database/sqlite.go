package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"yasirbin/internal/model"
)

type SQLiteStore struct {
	conn *sql.DB
}

func NewSQLite(dbPath string) (*SQLiteStore, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	conn, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}

	store := &SQLiteStore{conn: conn}
	if err := store.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("sqlite migrate: %w", err)
	}
	return store, nil
}

func (s *SQLiteStore) Driver() string {
	return "sqlite"
}

func (s *SQLiteStore) migrate() error {
	_, err := s.conn.Exec(`
		CREATE TABLE IF NOT EXISTS documents (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			slug TEXT UNIQUE NOT NULL,
			content TEXT NOT NULL,
			password TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME
		);
		CREATE INDEX IF NOT EXISTS idx_slug ON documents(slug);
		CREATE INDEX IF NOT EXISTS idx_expires ON documents(expires_at);
	`)
	return err
}

func (s *SQLiteStore) Create(doc *model.Document) error {
	_, err := s.conn.Exec(
		`INSERT INTO documents (slug, content, password, created_at, expires_at)
		 VALUES (?, ?, ?, ?, ?)`,
		doc.Slug, doc.Content, doc.Password, doc.CreatedAt, doc.ExpiresAt,
	)
	return err
}

func (s *SQLiteStore) GetBySlug(slug string) (*model.Document, error) {
	doc := &model.Document{}
	var expiresAt sql.NullTime
	err := s.conn.QueryRow(
		`SELECT id, slug, content, password, created_at, expires_at
		 FROM documents WHERE slug = ?`, slug,
	).Scan(&doc.ID, &doc.Slug, &doc.Content, &doc.Password, &doc.CreatedAt, &expiresAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if expiresAt.Valid {
		doc.ExpiresAt = &expiresAt.Time
	}
	return doc, nil
}

func (s *SQLiteStore) SlugExists(slug string) bool {
	var count int
	s.conn.QueryRow("SELECT COUNT(*) FROM documents WHERE slug = ?", slug).Scan(&count)
	return count > 0
}

func (s *SQLiteStore) CleanExpired() int64 {
	result, err := s.conn.Exec(
		"DELETE FROM documents WHERE expires_at IS NOT NULL AND expires_at < ?",
		time.Now().UTC(),
	)
	if err != nil {
		log.Printf("[sqlite] cleanup error: %v", err)
		return 0
	}
	n, _ := result.RowsAffected()
	return n
}

func (s *SQLiteStore) Close() error {
	return s.conn.Close()
}
