package database

import (
	"log"
	"yasirbin/internal/config"
)

// New initializes the database store based on configuration.
// If MongoDB is configured, it attempts to connect.
// If MongoDB connection fails or is not configured, it falls back to SQLite.
func New(cfg *config.Config) (Store, error) {
	if cfg.MongoURI != "" {
		log.Printf("Connecting to MongoDB (database: %s)...", cfg.MongoDB)
		store, err := NewMongo(cfg.MongoURI, cfg.MongoDB)
		if err == nil {
			log.Printf("✅ Connected to MongoDB successfully (database: %s)", cfg.MongoDB)
			return store, nil
		}
		log.Printf("⚠️ Failed to connect to MongoDB (%v). Falling back to SQLite...", err)
	} else {
		log.Printf("ℹ️ No MongoDB URI configured, using SQLite database: %s", cfg.DBPath)
	}

	store, err := NewSQLite(cfg.DBPath)
	if err != nil {
		return nil, err
	}
	log.Printf("✅ SQLite database initialized at %s", cfg.DBPath)
	return store, nil
}
