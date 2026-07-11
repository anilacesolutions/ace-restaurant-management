package report

import (
	"net/http"
	"time"

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

// MountAdmin mounts the reports endpoints — admin only.
func (h *Handler) MountAdmin(r chi.Router) {
	r.Get("/reports/sales", h.sales)
}

func (h *Handler) sales(w http.ResponseWriter, r *http.Request) {
	from := h.parseDay(r.URL.Query().Get("from"))
	to := h.parseDay(r.URL.Query().Get("to"))

	rep, err := h.svc.Sales(r.Context(), h.restaurantID, from, to)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, rep)
}

// parseDay parses a "YYYY-MM-DD" query param in the restaurant's timezone.
// Empty or invalid yields a zero time (open bound).
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
