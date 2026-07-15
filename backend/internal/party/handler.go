package party

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/ace-solutions/restaurant-backend/internal/httpx"
	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Handler struct {
	svc          *Service
	restaurantID bson.ObjectID
}

func NewHandler(svc *Service, defaultRestaurantID string) (*Handler, error) {
	rid, err := bson.ObjectIDFromHex(defaultRestaurantID)
	if err != nil {
		return nil, err
	}
	return &Handler{svc: svc, restaurantID: rid}, nil
}

// MountAdmin mounts the cari / kişiler list — admin only.
func (h *Handler) MountAdmin(r chi.Router) {
	r.Get("/parties", h.list)
	r.Get("/parties/{id}", h.ledger)
	r.Post("/parties", h.create)
	r.Delete("/parties/{id}", h.delete)
}

func (h *Handler) ledger(w http.ResponseWriter, r *http.Request) {
	id, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "geçersiz id")
		return
	}
	l, err := h.svc.GetLedger(r.Context(), h.restaurantID, id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, l)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.List(r.Context(), h.restaurantID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"parties": out})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var in Input
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "geçersiz istek gövdesi")
		return
	}
	p, err := h.svc.Create(r.Context(), h.restaurantID, in, time.Now())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, p)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "geçersiz id")
		return
	}
	if err := h.svc.Delete(r.Context(), h.restaurantID, id); err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) writeServiceError(w http.ResponseWriter, err error) {
	var verr ErrValidation
	switch {
	case errors.As(err, &verr):
		httpx.WriteError(w, http.StatusBadRequest, verr.Msg)
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "kişi bulunamadı")
	default:
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
	}
}
