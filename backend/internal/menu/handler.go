package menu

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ace-solutions/restaurant-backend/internal/httpx"
	"github.com/ace-solutions/restaurant-backend/internal/storage"
	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Handler struct {
	svc          *Service
	restaurantID bson.ObjectID
	s3           *storage.S3 // nil when image upload is disabled
}

func NewHandler(svc *Service, defaultRestaurantID string, s3 *storage.S3) (*Handler, error) {
	rid, err := bson.ObjectIDFromHex(defaultRestaurantID)
	if err != nil {
		return nil, err
	}
	return &Handler{svc: svc, restaurantID: rid, s3: s3}, nil
}

func (h *Handler) Mount(r chi.Router) {
	r.Get("/menu", h.getMenu)
}

// MountAdmin mounts menu management — admin only.
func (h *Handler) MountAdmin(r chi.Router) {
	r.Get("/menu/admin", h.listAll)
	r.Post("/menu/items", h.createItem)
	r.Patch("/menu/items/{id}", h.updateItem)
	r.Delete("/menu/items/{id}", h.deleteItem)
	r.Post("/menu/images/presign", h.presignImage)
}

func (h *Handler) getMenu(w http.ResponseWriter, r *http.Request) {
	cats, err := h.svc.GetMenu(r.Context(), h.restaurantID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"categories": cats})
}

func (h *Handler) listAll(w http.ResponseWriter, r *http.Request) {
	cats, err := h.svc.ListAll(r.Context(), h.restaurantID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"categories": cats})
}

func (h *Handler) createItem(w http.ResponseWriter, r *http.Request) {
	var in ItemInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "geçersiz istek gövdesi")
		return
	}
	item, err := h.svc.CreateItem(r.Context(), h.restaurantID, in)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, item)
}

func (h *Handler) updateItem(w http.ResponseWriter, r *http.Request) {
	id, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "geçersiz id")
		return
	}
	var in ItemInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "geçersiz istek gövdesi")
		return
	}
	item, err := h.svc.UpdateItem(r.Context(), h.restaurantID, id, in)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) deleteItem(w http.ResponseWriter, r *http.Request) {
	id, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "geçersiz id")
		return
	}
	if err := h.svc.DeleteItem(r.Context(), h.restaurantID, id); err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// presignImageRequest asks for a short-lived upload URL for one image.
type presignImageRequest struct {
	ContentType string `json:"contentType"`
	Ext         string `json:"ext"` // "jpg", "png", "webp" — used only for the object key
}

func (h *Handler) presignImage(w http.ResponseWriter, r *http.Request) {
	if h.s3 == nil {
		httpx.WriteError(w, http.StatusNotImplemented, "görsel yükleme yapılandırılmamış (S3_BUCKET yok)")
		return
	}
	var req presignImageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "geçersiz istek gövdesi")
		return
	}
	if !strings.HasPrefix(req.ContentType, "image/") {
		httpx.WriteError(w, http.StatusBadRequest, "sadece görsel yüklenebilir")
		return
	}
	ext := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(req.Ext), "."))
	if ext == "" {
		ext = "jpg"
	}

	key := fmt.Sprintf("menu/%s/%s.%s", h.restaurantID.Hex(), bson.NewObjectID().Hex(), ext)
	uploadURL, err := h.s3.PresignPut(r.Context(), key, req.ContentType, 5*time.Minute)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{
		"uploadUrl": uploadURL,
		"publicUrl": h.s3.PublicURL(key),
	})
}

// writeServiceError maps service errors to HTTP status codes.
func (h *Handler) writeServiceError(w http.ResponseWriter, err error) {
	var verr ErrValidation
	switch {
	case errors.As(err, &verr):
		httpx.WriteError(w, http.StatusBadRequest, verr.Msg)
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "ürün bulunamadı")
	default:
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
	}
}
