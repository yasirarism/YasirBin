package middleware

import (
	"net/http"
)

// ThemeCookie reads the "theme" cookie and sets html class accordingly.
// Returns "dark" or "light".
func GetTheme(r *http.Request) string {
	cookie, err := r.Cookie("theme")
	if err != nil || cookie.Value != "light" {
		return "dark"
	}
	return "light"
}
