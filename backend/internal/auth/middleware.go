package auth

import (
	"context"
	"errors"
	"net/http"

	"github.com/ace-solutions/restaurant-backend/internal/domain"
	"github.com/ace-solutions/restaurant-backend/internal/httpx"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type ctxKey int

const (
	ctxAdmin  ctxKey = 1
	ctxWaiter ctxKey = 2
)

// AdminFromCtx returns the logged-in admin user, panicking if the middleware
// was not installed — handlers behind RequireAdmin can always rely on it.
func AdminFromCtx(ctx context.Context) domain.User {
	return ctx.Value(ctxAdmin).(domain.User)
}

// WaiterFromCtx returns the logged-in waiter behind RequireWaiter.
func WaiterFromCtx(ctx context.Context) domain.Waiter {
	return ctx.Value(ctxWaiter).(domain.Waiter)
}

// SubjectFromCtx returns the id of whoever is authenticated (waiter or admin)
// and whether it is a waiter. Safe under RequireAuthenticated, where either
// kind may be present. ok is false if no auth middleware ran.
func SubjectFromCtx(ctx context.Context) (id bson.ObjectID, isWaiter, ok bool) {
	if w, isW := ctx.Value(ctxWaiter).(domain.Waiter); isW {
		return w.ID, true, true
	}
	if u, isU := ctx.Value(ctxAdmin).(domain.User); isU {
		return u.ID, false, true
	}
	return bson.NilObjectID, false, false
}

// RequireAdmin gates handlers behind a valid admin cookie. Returns 401 with a
// uniform envelope so the frontend can redirect to /login.
func (s *Service) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := ReadCookie(r, AdminCookieName)
		_, user, err := s.LookupAdmin(r.Context(), token)
		if err != nil {
			httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		ctx := context.WithValue(r.Context(), ctxAdmin, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireWaiter gates waiter-only routes.
func (s *Service) RequireWaiter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := ReadCookie(r, WaiterCookieName)
		_, waiter, err := s.LookupWaiter(r.Context(), token)
		if err != nil {
			status := http.StatusUnauthorized
			msg := "unauthorized"
			if errors.Is(err, ErrSessionExpired) {
				msg = "session expired"
			}
			httpx.WriteError(w, status, msg)
			return
		}
		ctx := context.WithValue(r.Context(), ctxWaiter, waiter)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAuthenticated accepts either an admin or a waiter session. Useful for
// shared reads like the table list — the garson picks a table, the admin
// manages it.
func (s *Service) RequireAuthenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if t := ReadCookie(r, AdminCookieName); t != "" {
			if _, user, err := s.LookupAdmin(r.Context(), t); err == nil {
				ctx := context.WithValue(r.Context(), ctxAdmin, user)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}
		if t := ReadCookie(r, WaiterCookieName); t != "" {
			if _, waiter, err := s.LookupWaiter(r.Context(), t); err == nil {
				ctx := context.WithValue(r.Context(), ctxWaiter, waiter)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
	})
}
