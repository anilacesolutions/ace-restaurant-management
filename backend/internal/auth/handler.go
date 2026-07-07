package auth

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ace-solutions/restaurant-backend/internal/httpx"
	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Handler struct {
	svc          *Service
	cookieSecure bool   // Secure flag for cookies — false in dev (http), true in prod
	cookieDomain string // parent domain in prod (".gunguzelbahce.online"), empty in dev
}

func NewHandler(svc *Service, cookieSecure bool, cookieDomain string) *Handler {
	return &Handler{svc: svc, cookieSecure: cookieSecure, cookieDomain: cookieDomain}
}

// MountPublic mounts routes that do not require authentication.
func (h *Handler) MountPublic(r chi.Router) {
	r.Route("/auth", func(r chi.Router) {
		r.Post("/login", h.login)
		r.Post("/logout", h.logout)
		r.Get("/me", h.me)
		r.Post("/qr/exchange", h.qrExchange)
	})
}

// MountAdmin mounts routes behind RequireAdmin. Uses direct paths instead of
// a nested r.Route("/auth") so the public group can also own /auth.
func (h *Handler) MountAdmin(r chi.Router) {
	r.Post("/auth/qr/issue", h.qrIssue)
}

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.Username == "" || req.Password == "" {
		httpx.WriteError(w, http.StatusBadRequest, "username and password required")
		return
	}
	sess, user, err := h.svc.LoginAdmin(r.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, ErrBadCredentials) {
			httpx.WriteError(w, http.StatusUnauthorized, "kullanici adi veya sifre hatali")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	SetSessionCookie(w, AdminCookieName, sess.Token, sess.ExpiresAt, h.cookieSecure, h.cookieDomain)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"user": user,
		"kind": "admin",
	})
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	// Try to invalidate whichever cookie is present.
	if t := ReadCookie(r, AdminCookieName); t != "" {
		_ = h.svc.Logout(r.Context(), t)
		ClearSessionCookie(w, AdminCookieName, h.cookieSecure, h.cookieDomain)
	}
	if t := ReadCookie(r, WaiterCookieName); t != "" {
		_ = h.svc.Logout(r.Context(), t)
		ClearSessionCookie(w, WaiterCookieName, h.cookieSecure, h.cookieDomain)
	}
	w.WriteHeader(http.StatusNoContent)
}

// me returns the currently authenticated principal (admin or waiter) or 401.
// Frontend calls this on mount to know where to send the user.
func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	if t := ReadCookie(r, AdminCookieName); t != "" {
		_, user, err := h.svc.LookupAdmin(r.Context(), t)
		if err == nil {
			httpx.WriteJSON(w, http.StatusOK, map[string]any{"user": user, "kind": "admin"})
			return
		}
	}
	if t := ReadCookie(r, WaiterCookieName); t != "" {
		_, waiter, err := h.svc.LookupWaiter(r.Context(), t)
		if err == nil {
			httpx.WriteJSON(w, http.StatusOK, map[string]any{"waiter": waiter, "kind": "waiter"})
			return
		}
	}
	httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
}

type qrIssueReq struct {
	WaiterID string `json:"waiterId"`
}

type qrIssueResp struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expiresAt"`
}

func (h *Handler) qrIssue(w http.ResponseWriter, r *http.Request) {
	var req qrIssueReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid body")
		return
	}
	waiterID, err := bson.ObjectIDFromHex(req.WaiterID)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid waiterId")
		return
	}
	admin := AdminFromCtx(r.Context())
	tok, exp, err := h.svc.IssueQR(r.Context(), waiterID, admin.ID)
	if err != nil {
		if errors.Is(err, ErrWaiterNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "waiter not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, qrIssueResp{
		Token:     tok,
		ExpiresAt: exp.Format("2006-01-02T15:04:05Z07:00"),
	})
}

type qrExchangeReq struct {
	Token string `json:"token"`
}

func (h *Handler) qrExchange(w http.ResponseWriter, r *http.Request) {
	var req qrExchangeReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.Token == "" {
		httpx.WriteError(w, http.StatusBadRequest, "token required")
		return
	}
	sess, waiter, err := h.svc.ExchangeQR(r.Context(), req.Token)
	if err != nil {
		if errors.Is(err, ErrTokenInvalid) {
			httpx.WriteError(w, http.StatusUnauthorized, "qr gecersiz veya kullanildi")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	SetSessionCookie(w, WaiterCookieName, sess.Token, sess.ExpiresAt, h.cookieSecure, h.cookieDomain)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"waiter":    waiter,
		"kind":      "waiter",
		"expiresAt": sess.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}
