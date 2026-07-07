package httpx

import (
	"encoding/json"
	"net/http"
	"regexp"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// devLANOrigin matches dev origins on this machine and the local LAN — e.g.
// localhost:3000, 127.0.0.1:3000, 192.168.x.y:3000 (a phone on the same WiFi).
var devLANOrigin = regexp.MustCompile(`^https?://(localhost|127\.0\.0\.1|10\.\d+\.\d+\.\d+|192\.168\.\d+\.\d+|172\.(1[6-9]|2\d|3[01])\.\d+\.\d+)(:\d+)?$`)

// NewRouter returns a chi router with the standard middleware stack and CORS.
// In dev (when no allowedOrigins are configured) we accept any local/LAN
// origin so a phone scanning a QR over WiFi can talk to the API. In prod the
// config provides an explicit allow-list.
func NewRouter(allowedOrigins []string) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	opts := cors.Options{
		AllowedMethods:   []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}
	if len(allowedOrigins) > 0 {
		opts.AllowedOrigins = allowedOrigins
		// Also accept LAN origins in addition to the explicit list — useful
		// when CORS_ALLOWED_ORIGINS pins a prod host but dev hits the same
		// binary from a LAN device.
		opts.AllowOriginFunc = func(_ *http.Request, origin string) bool {
			return devLANOrigin.MatchString(origin)
		}
	} else {
		opts.AllowOriginFunc = func(_ *http.Request, origin string) bool {
			return devLANOrigin.MatchString(origin)
		}
	}
	r.Use(cors.Handler(opts))

	return r
}

// WriteJSON writes v as JSON with the given status. Use everywhere — never
// build JSON responses by hand.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// WriteError writes a uniform error envelope.
func WriteError(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, map[string]string{"error": msg})
}
