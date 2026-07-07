// Package order manages per-table orders: opening a table, adding items,
// recomputing totals, and fanning changes out over MQTT to the cashier and
// kitchen. The backend is the single authoritative writer to Mongo; the waiter
// app talks REST and the backend publishes the realtime updates.
package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/ace-solutions/restaurant-backend/internal/domain"
	"github.com/ace-solutions/restaurant-backend/internal/mqttx"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ErrValidation wraps a user-facing message (maps to HTTP 400).
type ErrValidation struct{ Msg string }

func (e ErrValidation) Error() string { return e.Msg }

// activeStatuses are the order states that count as "table occupied" — a table
// has at most one order in one of these at a time.
var activeStatuses = bson.A{
	domain.OrderOpen, domain.OrderSent, domain.OrderPreparing,
	domain.OrderReady, domain.OrderServed,
}

// publisher is the slice of *mqttx.Client the service needs; nil disables it.
type publisher interface {
	Publish(topic string, qos byte, retained bool, payload []byte) error
}

type Service struct {
	db  *mongo.Database
	pub publisher // may be nil when realtime is disabled
}

func New(db *mongo.Database, pub publisher) *Service {
	// A typed-nil *mqttx.Client would be a non-nil interface; guard for that.
	if c, ok := pub.(*mqttx.Client); ok && c == nil {
		pub = nil
	}
	return &Service{db: db, pub: pub}
}

func (s *Service) coll() *mongo.Collection { return s.db.Collection("orders") }

// ActiveOrder returns the table's current open order, or nil if the table is
// free.
func (s *Service) ActiveOrder(ctx context.Context, restaurantID bson.ObjectID, tableNumber int) (*domain.Order, error) {
	var o domain.Order
	err := s.coll().FindOne(ctx, bson.M{
		"restaurantId": restaurantID,
		"tableNumber":  tableNumber,
		"status":       bson.M{"$in": activeStatuses},
	}).Decode(&o)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find active order: %w", err)
	}
	return &o, nil
}

// ListActive returns every occupied table's order, newest first — the cashier's
// live floor view.
func (s *Service) ListActive(ctx context.Context, restaurantID bson.ObjectID) ([]domain.Order, error) {
	cur, err := s.coll().Find(ctx,
		bson.M{"restaurantId": restaurantID, "status": bson.M{"$in": activeStatuses}},
		options.Find().SetSort(bson.D{{Key: "openedAt", Value: 1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("find active orders: %w", err)
	}
	out := []domain.Order{}
	if err := cur.All(ctx, &out); err != nil {
		return nil, fmt.Errorf("decode active orders: %w", err)
	}
	return out, nil
}

// AddItemInput is one line the waiter fires to the kitchen.
type AddItemInput struct {
	MenuItemID string `json:"menuItemId"`
	Qty        int    `json:"qty"`
	Note       string `json:"note"`
}

// AddItems opens the table's order if needed, appends the given items (snapshot
// of menu data, marked "sent"), recomputes totals, persists, and publishes the
// kitchen ticket + cashier update. This is the waiter's "Mutfağa Gönder".
func (s *Service) AddItems(ctx context.Context, restaurantID, subjectID bson.ObjectID, tableNumber int, ins []AddItemInput, now time.Time) (*domain.Order, error) {
	if len(ins) == 0 {
		return nil, ErrValidation{"En az bir ürün ekleyin"}
	}

	// Snapshot menu items so later menu edits don't rewrite history.
	ids := make([]bson.ObjectID, 0, len(ins))
	for _, in := range ins {
		id, err := bson.ObjectIDFromHex(in.MenuItemID)
		if err != nil {
			return nil, ErrValidation{"Geçersiz ürün"}
		}
		ids = append(ids, id)
	}
	menu, err := s.menuByID(ctx, restaurantID, ids)
	if err != nil {
		return nil, err
	}

	newItems := make([]domain.OrderItem, 0, len(ins))
	for _, in := range ins {
		if in.Qty <= 0 {
			return nil, ErrValidation{"Adet 0'dan büyük olmalı"}
		}
		mi, ok := menu[in.MenuItemID]
		if !ok {
			return nil, ErrValidation{"Ürün bulunamadı: " + in.MenuItemID}
		}
		newItems = append(newItems, domain.OrderItem{
			ID:               bson.NewObjectID(),
			MenuItemID:       mi.ID,
			Name:             mi.Name,
			Qty:              in.Qty,
			UnitPrice:        mi.Price,
			KDVOrani:         mi.KDVOrani,
			OTVVar:           mi.OTVVar,
			POSDepartmanKodu: mi.POSDepartmanKodu,
			Note:             in.Note,
			Status:           domain.ItemSent,
			AddedAt:          now,
			AddedBy:          subjectID,
		})
	}

	existing, err := s.ActiveOrder(ctx, restaurantID, tableNumber)
	if err != nil {
		return nil, err
	}

	var order *domain.Order
	if existing == nil {
		order = &domain.Order{
			ID:           bson.NewObjectID(),
			RestaurantID: restaurantID,
			TableNumber:  tableNumber,
			WaiterID:     subjectID,
			Status:       domain.OrderOpen,
			Items:        newItems,
			OpenedAt:     now,
			UpdatedAt:    now,
		}
		applyTotals(order)
		if _, err := s.coll().InsertOne(ctx, order); err != nil {
			return nil, fmt.Errorf("insert order: %w", err)
		}
	} else {
		existing.Items = append(existing.Items, newItems...)
		existing.UpdatedAt = now
		applyTotals(existing)
		_, err := s.coll().UpdateOne(ctx,
			bson.M{"_id": existing.ID},
			bson.M{"$set": bson.M{
				"items":        existing.Items,
				"subtotal":     existing.Subtotal,
				"kdvBreakdown": existing.KDVBreakdown,
				"otv":          existing.OTV,
				"grandTotal":   existing.GrandTotal,
				"updatedAt":    existing.UpdatedAt,
			}},
		)
		if err != nil {
			return nil, fmt.Errorf("update order: %w", err)
		}
		order = existing
	}

	s.publishKitchen(restaurantID, order, newItems, menu, now)
	s.publishTableUpdate(restaurantID, order)
	return order, nil
}

func (s *Service) menuByID(ctx context.Context, restaurantID bson.ObjectID, ids []bson.ObjectID) (map[string]domain.MenuItem, error) {
	cur, err := s.db.Collection("menuItems").Find(ctx, bson.M{
		"restaurantId": restaurantID,
		"_id":          bson.M{"$in": ids},
	})
	if err != nil {
		return nil, fmt.Errorf("find menu items: %w", err)
	}
	var items []domain.MenuItem
	if err := cur.All(ctx, &items); err != nil {
		return nil, fmt.Errorf("decode menu items: %w", err)
	}
	m := make(map[string]domain.MenuItem, len(items))
	for _, it := range items {
		m[it.ID.Hex()] = it
	}
	return m, nil
}

// applyTotals recomputes an order's money fields from its live items. Prices are
// KDV-included, so grandTotal = gross paid, subtotal = net (matrah), and
// kdvBreakdown holds the tax portion per rate. ÖTV needs a per-item amount we do
// not carry yet (menu has only otvVar) — left at 0, see CLAUDE.md.
func applyTotals(o *domain.Order) {
	kdv := map[string]domain.Kurus{}
	var gross, totalKDV domain.Kurus
	for _, it := range o.Items {
		if it.Status == domain.ItemVoided || it.Status == domain.ItemRefunded {
			continue
		}
		lineGross := it.UnitPrice * domain.Kurus(it.Qty)
		gross += lineGross
		lineKDV := lineGross * domain.Kurus(it.KDVOrani) / domain.Kurus(100+it.KDVOrani)
		kdv[strconv.Itoa(it.KDVOrani)] += lineKDV
		totalKDV += lineKDV
	}
	o.KDVBreakdown = kdv
	o.GrandTotal = gross
	o.Subtotal = gross - totalKDV
	o.OTV = 0
}

// --- realtime fan-out (best-effort; a DB write already succeeded) ----------

func (s *Service) publishKitchen(restaurantID bson.ObjectID, o *domain.Order, sent []domain.OrderItem, menu map[string]domain.MenuItem, now time.Time) {
	if s.pub == nil {
		return
	}
	// Only items the kitchen actually makes go on the ticket (bottled drinks
	// carry kitchenPrint=false).
	type ticketLine struct {
		Name string `json:"name"`
		Qty  int    `json:"qty"`
		Note string `json:"note,omitempty"`
	}
	lines := make([]ticketLine, 0, len(sent))
	for _, it := range sent {
		if mi, ok := menu[it.MenuItemID.Hex()]; ok && !mi.KitchenPrint {
			continue
		}
		lines = append(lines, ticketLine{Name: it.Name, Qty: it.Qty, Note: it.Note})
	}
	if len(lines) == 0 {
		return
	}
	payload, err := json.Marshal(map[string]any{
		"orderId":     o.ID.Hex(),
		"tableNumber": o.TableNumber,
		"waiterId":    o.WaiterID.Hex(),
		"items":       lines,
		"at":          now,
	})
	if err != nil {
		slog.Error("kitchen payload marshal", "err", err)
		return
	}
	topic := mqttx.KitchenPrint(restaurantID.Hex())
	if err := s.pub.Publish(topic, mqttx.QoSExactlyOnce, false, payload); err != nil {
		slog.Error("kitchen publish failed", "err", err, "table", o.TableNumber)
	}
}

func (s *Service) publishTableUpdate(restaurantID bson.ObjectID, o *domain.Order) {
	if s.pub == nil {
		return
	}
	payload, err := json.Marshal(o)
	if err != nil {
		slog.Error("order payload marshal", "err", err)
		return
	}
	topic := mqttx.TableOrders(restaurantID.Hex(), o.TableNumber)
	if err := s.pub.Publish(topic, mqttx.QoSAtLeastOnce, false, payload); err != nil {
		slog.Error("table order publish failed", "err", err, "table", o.TableNumber)
	}
}
