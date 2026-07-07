package auth

import (
	"net/http"
	"time"
)

const (
	AdminCookieName  = "rs_admin"
	WaiterCookieName = "rs_waiter"
)

// SetSessionCookie writes an HTTP-only session cookie. Secure is enabled in
// non-dev environments; SameSite=Lax allows the QR-scan redirect flow. domain
// is empty in dev (host-only cookie); in prod it is set to the parent domain
// (e.g. ".gunguzelbahce.online") so the cookie is shared across the app and API
// subdomains — otherwise SSR on the app host can't see the API-set cookie.
func SetSessionCookie(w http.ResponseWriter, name, token string, expiresAt time.Time, secure bool, domain string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    token,
		Path:     "/",
		Domain:   domain,
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearSessionCookie expires the cookie immediately. domain must match the one
// used when the cookie was set.
func ClearSessionCookie(w http.ResponseWriter, name string, secure bool, domain string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		Domain:   domain,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// ReadCookie returns the cookie value or "" if absent.
func ReadCookie(r *http.Request, name string) string {
	c, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return c.Value
}
