// Package report builds sales analytics over closed orders — the ERP "Raporlar"
// screen. Everything is derived from orders with status=closed, bucketed by
// closedAt. Money stays in integer kuruş.
package report

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/ace-solutions/restaurant-backend/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Service struct {
	db *mongo.Database
}

func New(db *mongo.Database) *Service { return &Service{db: db} }

// ItemStat is one line in the best-sellers list.
type ItemStat struct {
	Name    string       `json:"name"`
	Qty     int          `json:"qty"`
	Revenue domain.Kurus `json:"revenue"`
}

// SalesReport aggregates one date range. Revenue is gross (KDV-included); Net is
// the matrah (tax-excluded); KDV maps rate -> tax portion.
type SalesReport struct {
	Revenue    domain.Kurus            `json:"revenue"`
	Net        domain.Kurus            `json:"net"`
	OTV        domain.Kurus            `json:"otv"`
	OrderCount int                     `json:"orderCount"`
	Payment    map[string]domain.Kurus `json:"payment"` // "nakit","kart",... -> gross
	KDV        map[string]domain.Kurus `json:"kdv"`     // "10","20" -> tax portion
	TopItems   []ItemStat              `json:"topItems"`
}

// Sales aggregates closed orders whose closedAt is in [from, to).
func (s *Service) Sales(ctx context.Context, restaurantID bson.ObjectID, from, to time.Time) (*SalesReport, error) {
	filter := bson.M{
		"restaurantId": restaurantID,
		"status":       domain.OrderClosed,
	}
	rng := bson.M{}
	if !from.IsZero() {
		rng["$gte"] = from
	}
	if !to.IsZero() {
		rng["$lt"] = to
	}
	if len(rng) > 0 {
		filter["closedAt"] = rng
	}

	cur, err := s.db.Collection("orders").Find(ctx, filter,
		options.Find().SetProjection(bson.M{
			"grandTotal":    1,
			"subtotal":      1,
			"otv":           1,
			"kdvBreakdown":  1,
			"paymentMethod": 1,
			"items":         1,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("find closed orders: %w", err)
	}
	var orders []domain.Order
	if err := cur.All(ctx, &orders); err != nil {
		return nil, fmt.Errorf("decode closed orders: %w", err)
	}

	rep := &SalesReport{
		Payment: map[string]domain.Kurus{},
		KDV:     map[string]domain.Kurus{},
	}
	items := map[string]*ItemStat{}

	for _, o := range orders {
		rep.Revenue += o.GrandTotal
		rep.Net += o.Subtotal
		rep.OTV += o.OTV
		rep.OrderCount++

		method := o.PaymentMethod
		if method == "" {
			method = "belirtilmemiş"
		}
		rep.Payment[method] += o.GrandTotal

		for rate, amt := range o.KDVBreakdown {
			rep.KDV[rate] += amt
		}

		for _, it := range o.Items {
			if it.Status == domain.ItemVoided || it.Status == domain.ItemRefunded {
				continue
			}
			st, ok := items[it.Name]
			if !ok {
				st = &ItemStat{Name: it.Name}
				items[it.Name] = st
			}
			st.Qty += it.Qty
			st.Revenue += it.UnitPrice * domain.Kurus(it.Qty)
		}
	}

	rep.TopItems = topItems(items, 20)
	return rep, nil
}

// topItems returns the n most-sold items by quantity (revenue breaks ties).
func topItems(m map[string]*ItemStat, n int) []ItemStat {
	out := make([]ItemStat, 0, len(m))
	for _, v := range m {
		out = append(out, *v)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Qty != out[j].Qty {
			return out[i].Qty > out[j].Qty
		}
		if out[i].Revenue != out[j].Revenue {
			return out[i].Revenue > out[j].Revenue
		}
		return out[i].Name < out[j].Name
	})
	if len(out) > n {
		out = out[:n]
	}
	return out
}
