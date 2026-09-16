package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"yasirbin/internal/config"
	"yasirbin/internal/database"
	"yasirbin/internal/middleware"
	"yasirbin/internal/model"
	"yasirbin/internal/util"
)

type Handler struct {
	db   *database.DB
	cfg  *config.Config
	tmpl *template.Template
}

func New(db *database.DB, cfg *config.Config) *Handler {
	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
	}
	tmpl := template.Must(template.New("").Funcs(funcMap).ParseGlob("templates/*.html"))
	return &Handler{db: db, cfg: cfg, tmpl: tmpl}
}

// --- API ---

type CreateRequest struct {
	Content  string `json:"content"`
	Password string `json:"password"`
	Expires  string `json:"expires"`
}

type APIResponse struct {
	OK      bool        `json:"ok"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (h *Handler) APICreateDocument(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.jsonError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req CreateRequest

	ct := r.Header.Get("Content-Type")
	if strings.Contains(ct, "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.jsonError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
	} else {
		// form-data or url-encoded
		if err := r.ParseForm(); err != nil {
			h.jsonError(w, http.StatusBadRequest, "invalid form data")
			return
		}
		req.Content = r.FormValue("content")
		req.Password = r.FormValue("password")
		req.Expires = r.FormValue("expires")
	}

	if strings.TrimSpace(req.Content) == "" {
		h.jsonError(w, http.StatusBadRequest, "content is required")
		return
	}

	if int64(len(req.Content)) > h.cfg.MaxSize {
		h.jsonError(w, http.StatusBadRequest, "content too large")
		return
	}

	// Generate unique slug
	var slug string
	for i := 0; i < 10; i++ {
		slug = util.GenerateSlug(10)
		if !h.db.SlugExists(slug) {
			break
		}
	}

	now := time.Now().UTC()
	doc := &model.Document{
		Slug:      slug,
		Content:   req.Content,
		CreatedAt: now,
	}

	// Handle password
	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			h.jsonError(w, http.StatusInternalServerError, "failed to hash password")
			return
		}
		doc.Password = string(hash)
	}

	// Handle expiration
	switch req.Expires {
	case "1d":
		t := now.Add(24 * time.Hour)
		doc.ExpiresAt = &t
	case "7d":
		t := now.Add(7 * 24 * time.Hour)
		doc.ExpiresAt = &t
	case "never":
		doc.ExpiresAt = nil
	default: // "30d" or empty
		t := now.Add(30 * 24 * time.Hour)
		doc.ExpiresAt = &t
	}

	if err := h.db.Create(doc); err != nil {
		h.jsonError(w, http.StatusInternalServerError, "failed to create document")
		return
	}

	baseURL := h.cfg.BaseURL
	if baseURL == "" {
		scheme := "http"
		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		baseURL = scheme + "://" + r.Host
	}

	resp := APIResponse{
		OK:      true,
		Message: "succesfully created document",
		Data: map[string]interface{}{
			"url":       baseURL + "/" + slug,
			"length":    len(req.Content),
			"date":      now.Format(time.RFC3339Nano),
			"expire_at": func() interface{} {
				if doc.ExpiresAt != nil {
					return doc.ExpiresAt.Format(time.RFC3339Nano)
				}
				return nil
			}(),
		},
	}

	// Check if request accepts HTML (from HTMX form submission)
	accept := r.Header.Get("Accept")
	hxRequest := r.Header.Get("HX-Request")
	if hxRequest == "true" || strings.Contains(accept, "text/html") {
		// Redirect to the document page
		w.Header().Set("HX-Redirect", "/"+slug)
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) APIGetDocument(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		h.jsonError(w, http.StatusBadRequest, "key parameter is required")
		return
	}

	doc, err := h.db.GetBySlug(key)
	if err != nil {
		if err == sql.ErrNoRows {
			h.jsonError(w, http.StatusNotFound, "document not found")
		} else {
			h.jsonError(w, http.StatusInternalServerError, "database error")
		}
		return
	}

	// Check expiry
	if doc.ExpiresAt != nil && doc.ExpiresAt.Before(time.Now().UTC()) {
		h.jsonError(w, http.StatusNotFound, "document has expired")
		return
	}

	// Password protected responses don't include content
	if doc.Password != "" {
		password := r.URL.Query().Get("password")
		if password == "" {
			h.jsonError(w, http.StatusForbidden, "password required")
			return
		}
		if bcrypt.CompareHashAndPassword([]byte(doc.Password), []byte(password)) != nil {
			h.jsonError(w, http.StatusForbidden, "incorrect password")
			return
		}
	}

	resp := APIResponse{
		OK:      true,
		Message: "document found",
		Data: map[string]interface{}{
			"slug":    doc.Slug,
			"content": doc.Content,
			"date":    doc.CreatedAt.Format(time.RFC3339Nano),
			"expire_at": func() interface{} {
				if doc.ExpiresAt != nil {
					return doc.ExpiresAt.Format(time.RFC3339Nano)
				}
				return nil
			}(),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) APIDocumentHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.APICreateDocument(w, r)
	case http.MethodGet:
		h.APIGetDocument(w, r)
	default:
		h.jsonError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) APITheme(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	current := middleware.GetTheme(r)
	newTheme := "dark"
	if current == "dark" {
		newTheme = "light"
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "theme",
		Value:    newTheme,
		Path:     "/",
		MaxAge:   365 * 24 * 60 * 60,
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	})

	w.WriteHeader(http.StatusOK)
}

// --- Pages ---

type PageData struct {
	Theme     string
	ThemeClass string
	Doc       *model.Document
	Lines     []CodeLine
	Slug      string
	Error     string
	BaseURL   string
}

type CodeLine struct {
	Number  int
	Content string
}

func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	// Only handle exact "/" path
	if r.URL.Path != "/" {
		// Try as document slug
		h.ViewDocument(w, r)
		return
	}

	theme := middleware.GetTheme(r)
	data := PageData{
		Theme:      theme,
		ThemeClass: theme,
	}
	h.render(w, "home.html", data)
}

func (h *Handler) About(w http.ResponseWriter, r *http.Request) {
	theme := middleware.GetTheme(r)
	data := PageData{
		Theme:      theme,
		ThemeClass: theme,
	}
	h.render(w, "about.html", data)
}

func (h *Handler) ViewDocument(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimPrefix(r.URL.Path, "/")
	if slug == "" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	theme := middleware.GetTheme(r)

	doc, err := h.db.GetBySlug(slug)
	if err != nil {
		if err == sql.ErrNoRows {
			data := PageData{Theme: theme, ThemeClass: theme, Error: "Document not found"}
			w.WriteHeader(http.StatusNotFound)
			h.render(w, "error.html", data)
		} else {
			data := PageData{Theme: theme, ThemeClass: theme, Error: "Internal server error"}
			w.WriteHeader(http.StatusInternalServerError)
			h.render(w, "error.html", data)
		}
		return
	}

	// Check expiry
	if doc.ExpiresAt != nil && doc.ExpiresAt.Before(time.Now().UTC()) {
		data := PageData{Theme: theme, ThemeClass: theme, Error: "Document has expired"}
		w.WriteHeader(http.StatusNotFound)
		h.render(w, "error.html", data)
		return
	}

	// Password protected
	if doc.Password != "" {
		if r.Method == http.MethodPost {
			r.ParseForm()
			password := r.FormValue("password")
			if bcrypt.CompareHashAndPassword([]byte(doc.Password), []byte(password)) != nil {
				data := PageData{
					Theme:      theme,
					ThemeClass: theme,
					Slug:       slug,
					Error:      "Incorrect password",
				}
				h.render(w, "password.html", data)
				return
			}
			// Password correct, fall through to show document
		} else {
			data := PageData{
				Theme:      theme,
				ThemeClass: theme,
				Slug:       slug,
			}
			h.render(w, "password.html", data)
			return
		}
	}

	// Build code lines
	contentLines := strings.Split(doc.Content, "\n")
	lines := make([]CodeLine, len(contentLines))
	for i, line := range contentLines {
		lines[i] = CodeLine{Number: i + 1, Content: line}
	}

	data := PageData{
		Theme:      theme,
		ThemeClass: theme,
		Doc:        doc,
		Lines:      lines,
		Slug:       slug,
	}
	h.render(w, "document.html", data)
}

func (h *Handler) RawDocument(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimPrefix(r.URL.Path, "/raw/")
	if slug == "" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	doc, err := h.db.GetBySlug(slug)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Document not found", http.StatusNotFound)
		} else {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Check expiry
	if doc.ExpiresAt != nil && doc.ExpiresAt.Before(time.Now().UTC()) {
		http.Error(w, "Document has expired", http.StatusNotFound)
		return
	}

	// Password check via query param
	if doc.Password != "" {
		password := r.URL.Query().Get("password")
		if password == "" {
			http.Error(w, "Password required", http.StatusForbidden)
			return
		}
		if bcrypt.CompareHashAndPassword([]byte(doc.Password), []byte(password)) != nil {
			http.Error(w, "Incorrect password", http.StatusForbidden)
			return
		}
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprint(w, doc.Content)
}

// --- Helpers ---

func (h *Handler) render(w http.ResponseWriter, name string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("template error (%s): %v", name, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *Handler) jsonError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(APIResponse{OK: false, Message: message})
}
