package order

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/ace-solutions/restaurant-backend/internal/auth"
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

// Mount mounts order routes readable/writable by any authenticated session —
// the waiter fires items, the cashier reads the tab.
func (h *Handler) Mount(r chi.Router) {
	r.Get("/orders/active", h.listActive)
	r.Get("/orders/table/{number}", h.activeForTable)
	r.Post("/orders/table/{number}/items", h.addItems)
	r.Delete("/orders/table/{number}/items/{itemId}", h.voidItem)
}

// MountAdmin mounts cashier-only order actions (settle/close a table).
func (h *Handler) MountAdmin(r chi.Router) {
	r.Post("/orders/table/{number}/close", h.closeTable)
}

type closeReq struct {
	PaymentMethod string `json:"paymentMethod"`
}

func (h *Handler) closeTable(w http.ResponseWriter, r *http.Request) {
	tableNo, ok := parseTable(w, r)
	if !ok {
		return
	}
	var req closeReq
	// Body is optional; ignore decode errors on an empty body.
	_ = json.NewDecoder(r.Body).Decode(&req)

	o, err := h.svc.CloseTable(r.Context(), h.restaurantID, tableNo, req.PaymentMethod, time.Now())
	if err != nil {
		var verr ErrValidation
		if errors.As(err, &verr) {
			httpx.WriteError(w, http.StatusBadRequest, verr.Msg)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"order": o})
}

func (h *Handler) listActive(w http.ResponseWriter, r *http.Request) {
	orders, err := h.svc.ListActive(r.Context(), h.restaurantID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"orders": orders})
}

func (h *Handler) activeForTable(w http.ResponseWriter, r *http.Request) {
	tableNo, ok := parseTable(w, r)
	if !ok {
		return
	}
	o, err := h.svc.ActiveOrder(r.Context(), h.restaurantID, tableNo)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"order": o})
}

type addItemsReq struct {
	Items []AddItemInput `json:"items"`
}

func (h *Handler) addItems(w http.ResponseWriter, r *http.Request) {
	tableNo, ok := parseTable(w, r)
	if !ok {
		return
	}
	var req addItemsReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "geçersiz istek gövdesi")
		return
	}
	subjectID, _, ok := auth.SubjectFromCtx(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	o, err := h.svc.AddItems(r.Context(), h.restaurantID, subjectID, tableNo, req.Items, time.Now())
	if err != nil {
		var verr ErrValidation
		if errors.As(err, &verr) {
			httpx.WriteError(w, http.StatusBadRequest, verr.Msg)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"order": o})
}

func (h *Handler) voidItem(w http.ResponseWriter, r *http.Request) {
	tableNo, ok := parseTable(w, r)
	if !ok {
		return
	}
	itemID, err := bson.ObjectIDFromHex(chi.URLParam(r, "itemId"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "geçersiz ürün id")
		return
	}
	subjectID, _, ok := auth.SubjectFromCtx(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	o, err := h.svc.VoidItem(r.Context(), h.restaurantID, subjectID, tableNo, itemID, time.Now())
	if err != nil {
		var verr ErrValidation
		switch {
		case errors.As(err, &verr):
			httpx.WriteError(w, http.StatusBadRequest, verr.Msg)
		case errors.Is(err, ErrItemNotFound):
			httpx.WriteError(w, http.StatusNotFound, "sipariş satırı bulunamadı")
		default:
			httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"order": o})
}

func parseTable(w http.ResponseWriter, r *http.Request) (int, bool) {
	n, err := strconv.Atoi(chi.URLParam(r, "number"))
	if err != nil || n < 1 {
		httpx.WriteError(w, http.StatusBadRequest, "geçersiz masa numarası")
		return 0, false
	}
	return n, true
}
