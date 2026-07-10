package waiters

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
	restaurantID bson.ObjectID
}

func NewHandler(svc *Service, defaultRestaurantID string) (*Handler, error) {
	rid, err := bson.ObjectIDFromHex(defaultRestaurantID)
	if err != nil {
		return nil, err
	}
	return &Handler{svc: svc, restaurantID: rid}, nil
}

// MountAdmin mounts waiter listing + management — admin only.
func (h *Handler) MountAdmin(r chi.Router) {
	r.Get("/waiters", h.listActive)  // active only — QR issuance picker
	r.Get("/waiters/all", h.listAll) // active + passive — management screen
	r.Post("/waiters", h.create)
	r.Patch("/waiters/{id}", h.update)
	r.Delete("/waiters/{id}", h.delete)
}

func (h *Handler) listActive(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.ListActive(r.Context(), h.restaurantID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"waiters": out})
}

func (h *Handler) listAll(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.List(r.Context(), h.restaurantID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"waiters": out})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var in WaiterInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "geçersiz istek gövdesi")
		return
	}
	waiter, err := h.svc.Create(r.Context(), h.restaurantID, in)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, waiter)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "geçersiz id")
		return
	}
	var in WaiterInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "geçersiz istek gövdesi")
		return
	}
	waiter, err := h.svc.Update(r.Context(), h.restaurantID, id, in)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, waiter)
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

// writeServiceError maps service errors to HTTP status codes.
func (h *Handler) writeServiceError(w http.ResponseWriter, err error) {
	var verr ErrValidation
	switch {
	case errors.As(err, &verr):
		httpx.WriteError(w, http.StatusBadRequest, verr.Msg)
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "garson bulunamadı")
	default:
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
	}
}
