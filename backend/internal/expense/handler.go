package expense

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/ace-solutions/restaurant-backend/internal/auth"
	"github.com/ace-solutions/restaurant-backend/internal/httpx"
	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const dateParam = "2006-01-02"

type Handler struct {
	svc          *Service
	restaurantID bson.ObjectID
	loc          *time.Location
}

func NewHandler(svc *Service, defaultRestaurantID string, loc *time.Location) (*Handler, error) {
	rid, err := bson.ObjectIDFromHex(defaultRestaurantID)
	if err != nil {
		return nil, err
	}
	return &Handler{svc: svc, restaurantID: rid, loc: loc}, nil
}

// MountAdmin mounts the gider defteri — admin only.
func (h *Handler) MountAdmin(r chi.Router) {
	r.Get("/expenses", h.list)
	r.Post("/expenses", h.create)
	r.Delete("/expenses/{id}", h.delete)
	r.Post("/expenses/{id}/payments", h.addPayment)
	r.Delete("/expenses/{id}/payments/{paymentId}", h.deletePayment)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	from := h.parseDay(r.URL.Query().Get("from"))
	to := h.parseDay(r.URL.Query().Get("to"))

	out, err := h.svc.List(r.Context(), h.restaurantID, from, to)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"expenses": out})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var in Input
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "geçersiz istek gövdesi")
		return
	}
	user := auth.AdminFromCtx(r.Context())
	exp, err := h.svc.Create(r.Context(), h.restaurantID, user.ID, in, time.Now())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, exp)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "geçersiz id")
		return
	}
	if err := h.svc.Delete(r.Context(), h.restaurantID, id); err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "gider bulunamadı")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) addPayment(w http.ResponseWriter, r *http.Request) {
	id, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "geçersiz id")
		return
	}
	var in PaymentInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "geçersiz istek gövdesi")
		return
	}
	exp, err := h.svc.AddPayment(r.Context(), h.restaurantID, id, in, time.Now())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, exp)
}

func (h *Handler) deletePayment(w http.ResponseWriter, r *http.Request) {
	id, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "geçersiz id")
		return
	}
	pid, err := bson.ObjectIDFromHex(chi.URLParam(r, "paymentId"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "geçersiz ödeme id")
		return
	}
	exp, err := h.svc.DeletePayment(r.Context(), h.restaurantID, id, pid)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, exp)
}

// parseDay parses a "YYYY-MM-DD" query param in the restaurant's timezone.
// Empty or invalid yields a zero time (treated as an open bound).
func (h *Handler) parseDay(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.ParseInLocation(dateParam, s, h.loc)
	if err != nil {
		return time.Time{}
	}
	return t
}

func (h *Handler) writeServiceError(w http.ResponseWriter, err error) {
	var verr ErrValidation
	switch {
	case errors.As(err, &verr):
		httpx.WriteError(w, http.StatusBadRequest, verr.Msg)
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "gider bulunamadı")
	default:
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
	}
}
