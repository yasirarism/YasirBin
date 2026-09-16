package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"yasirbin/internal/config"
	"yasirbin/internal/database"
	"yasirbin/internal/handler"
)

func main() {
	cfg := config.Load()

	db, err := database.New(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Start cleanup goroutine
	go func() {
		ticker := time.NewTicker(time.Duration(cfg.CleanupMin) * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			n := db.CleanExpired()
			if n > 0 {
				log.Printf("Cleaned up %d expired documents", n)
			}
		}
	}()

	h := handler.New(db, cfg)

	mux := http.NewServeMux()

	// Static files
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("static/css"))))
	mux.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir("static/images"))))
	mux.Handle("/manifest.json", fs)

	// API routes
	mux.HandleFunc("/api/document", h.APIDocumentHandler)
	mux.HandleFunc("/api/theme", h.APITheme)

	// Page routes
	mux.HandleFunc("/about", h.About)
	mux.HandleFunc("/docs", h.Docs)
	mux.HandleFunc("/api-docs", h.Docs)
	mux.HandleFunc("/raw/", h.RawDocument)
	mux.HandleFunc("/", h.Home)

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("🚀 YasirBin starting on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
