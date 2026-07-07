package tables

import (
	"errors"
	"net/http"

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

// MountReadOnly mounts the table list — readable by both admin and waiter.
func (h *Handler) MountReadOnly(r chi.Router) {
	r.Get("/tables", h.list)
}

// MountAdmin mounts table configuration — admin only.
func (h *Handler) MountAdmin(r chi.Router) {
	r.Post("/tables", h.add)
	r.Delete("/tables/{id}", h.delete)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.List(r.Context(), h.restaurantID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"tables": out})
}

func (h *Handler) add(w http.ResponseWriter, r *http.Request) {
	t, err := h.svc.AddNext(r.Context(), h.restaurantID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, t)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	idHex := chi.URLParam(r, "id")
	id, err := bson.ObjectIDFromHex(idHex)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.svc.Delete(r.Context(), h.restaurantID, id); err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "table not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
