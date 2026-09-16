package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port       int
	DBPath     string
	MongoURI   string
	MongoDB    string
	BaseURL    string
	MaxSize    int64 // max paste size in bytes
	CleanupMin int   // cleanup interval in minutes
}

func Load() *Config {
	c := &Config{
		Port:       3000,
		DBPath:     "./data/yasirbin.db",
		MongoDB:    "yasirbin",
		BaseURL:    "",
		MaxSize:    1 * 1024 * 1024, // 1MB
		CleanupMin: 5,
	}

	if v := os.Getenv("PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.Port = n
		}
	}
	if v := os.Getenv("DB_PATH"); v != "" {
		c.DBPath = v
	}
	if v := os.Getenv("BASE_URL"); v != "" {
		c.BaseURL = v
	}
	if v := os.Getenv("MAX_SIZE"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			c.MaxSize = n
		}
	}

	// MongoDB configuration
	if v := os.Getenv("MONGODB_URI"); v != "" {
		c.MongoURI = v
	} else if v := os.Getenv("MONGO_URI"); v != "" {
		c.MongoURI = v
	} else if v := os.Getenv("DATABASE_URL"); strings.HasPrefix(v, "mongodb://") || strings.HasPrefix(v, "mongodb+srv://") {
		c.MongoURI = v
	}

	if v := os.Getenv("MONGODB_DATABASE"); v != "" {
		c.MongoDB = v
	} else if v := os.Getenv("MONGO_DB"); v != "" {
		c.MongoDB = v
	} else if v := os.Getenv("DB_NAME"); v != "" {
		c.MongoDB = v
	}

	return c
}
