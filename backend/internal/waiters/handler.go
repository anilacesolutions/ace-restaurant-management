package waiters

import (
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

func (h *Handler) Mount(r chi.Router) {
	r.Get("/waiters", h.list)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.ListActive(r.Context(), h.restaurantID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"waiters": out})
}
