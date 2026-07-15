// Package report builds sales analytics over closed orders — the ERP "Raporlar"
// screen. Everything is derived from orders with status=closed, bucketed by
// closedAt. Money stays in integer kuruş.
package report

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/ace-solutions/restaurant-backend/internal/domain"
	"github.com/ace-solutions/restaurant-backend/internal/mqttx"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// publisher is the slice of *mqttx.Client the service needs; nil disables print.
type publisher interface {
	Publish(topic string, qos byte, retained bool, payload []byte) error
}

type Service struct {
	db  *mongo.Database
	pub publisher // may be nil when realtime/print is disabled
}

func New(db *mongo.Database, pub publisher) *Service {
	// A typed-nil *mqttx.Client would be a non-nil interface; guard for that.
	if c, ok := pub.(*mqttx.Client); ok && c == nil {
		pub = nil
	}
	return &Service{db: db, pub: pub}
}

// ErrValidation wraps a user-facing message (maps to HTTP 400).
type ErrValidation struct{ Msg string }

func (e ErrValidation) Error() string { return e.Msg }

// ItemStat is one line in the best-sellers list.
type ItemStat struct {
	Name    string       `json:"name"`
	Qty     int          `json:"qty"`
	Revenue domain.Kurus `json:"revenue"`
}

// SalesReport aggregates one date range. Revenue is gross (KDV-included); Net is
// the matrah (tax-excluded); KDV maps rate -> tax portion.
//
// Financial overview: Expense is spending in the range (by spentAt), Profit is
// Revenue-Expense. OpenReceivable/OpenPayable are current snapshots (all-time
// outstanding, NOT range-scoped) — what's owed to us vs what we owe.
type SalesReport struct {
	Revenue    domain.Kurus            `json:"revenue"`
	Net        domain.Kurus            `json:"net"`
	OTV        domain.Kurus            `json:"otv"`
	OrderCount int                     `json:"orderCount"`
	Payment    map[string]domain.Kurus `json:"payment"` // "nakit","kart",... -> gross
	KDV        map[string]domain.Kurus `json:"kdv"`     // "10","20" -> tax portion
	TopItems   []ItemStat              `json:"topItems"`
	Guests     int                     `json:"guests"` // fiks menü kişi sayısı toplamı (IsFix satır qty)

	Expense        domain.Kurus `json:"expense"`        // giderler in range (by spentAt)
	Profit         domain.Kurus `json:"profit"`         // Revenue - Expense
	OpenReceivable domain.Kurus `json:"openReceivable"` // outstanding owed TO us (now)
	OpenPayable    domain.Kurus `json:"openPayable"`    // outstanding we owe (now)
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
			// Fiks menü parent line qty == number of people served.
			if it.IsFix {
				rep.Guests += it.Qty
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

	// Financial overview.
	exp, err := s.sumExpenses(ctx, restaurantID, from, to)
	if err != nil {
		return nil, err
	}
	rep.Expense = exp
	rep.Profit = rep.Revenue - exp

	rep.OpenReceivable, err = s.outstanding(ctx, "receivables", restaurantID)
	if err != nil {
		return nil, err
	}
	rep.OpenPayable, err = s.outstanding(ctx, "expenses", restaurantID)
	if err != nil {
		return nil, err
	}

	return rep, nil
}

// BucketPoint is one time bucket of the trend series (an hour, day, or month).
type BucketPoint struct {
	Start      time.Time    `json:"start"`
	Revenue    domain.Kurus `json:"revenue"`
	Expense    domain.Kurus `json:"expense"`
	Guests     int          `json:"guests"`
	OrderCount int          `json:"orderCount"`
}

// TimeSeriesReport is a bucketed trend over [from, to) plus range totals. Bucket
// is one of hour|day|month. Points covers every bucket in range (gaps = zero) so
// the chart is continuous.
type TimeSeriesReport struct {
	Bucket     string        `json:"bucket"`
	Points     []BucketPoint `json:"points"`
	Revenue    domain.Kurus  `json:"revenue"`
	Expense    domain.Kurus  `json:"expense"`
	Guests     int           `json:"guests"`
	OrderCount int           `json:"orderCount"`
}

func truncBucket(t time.Time, bucket string, loc *time.Location) time.Time {
	t = t.In(loc)
	switch bucket {
	case "month":
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, loc)
	case "hour":
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, loc)
	default: // day
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
	}
}

func nextBucket(t time.Time, bucket string) time.Time {
	switch bucket {
	case "month":
		return t.AddDate(0, 1, 0)
	case "hour":
		return t.Add(time.Hour)
	default:
		return t.AddDate(0, 0, 1)
	}
}

// TimeSeries buckets revenue+guests (from closed orders, by closedAt) and
// expense (by spentAt) into an hour/day/month trend over [from, to).
func (s *Service) TimeSeries(ctx context.Context, restaurantID bson.ObjectID, from, to time.Time, bucket string, loc *time.Location) (*TimeSeriesReport, error) {
	if bucket != "hour" && bucket != "day" && bucket != "month" {
		bucket = "day"
	}
	rep := &TimeSeriesReport{Bucket: bucket, Points: []BucketPoint{}}

	// Pre-create every bucket in range so gaps render as zero.
	idx := map[int64]*BucketPoint{}
	if !from.IsZero() && !to.IsZero() {
		for b := truncBucket(from, bucket, loc); b.Before(to); b = nextBucket(b, bucket) {
			p := &BucketPoint{Start: b}
			idx[b.Unix()] = p
			rep.Points = append(rep.Points, BucketPoint{Start: b})
		}
	}
	// bucketFor returns the mutable point for a timestamp, creating one if the
	// range was open (from/to zero) so nothing is dropped.
	get := func(t time.Time) *BucketPoint {
		key := truncBucket(t, bucket, loc)
		if p, ok := idx[key.Unix()]; ok {
			return p
		}
		p := &BucketPoint{Start: key}
		idx[key.Unix()] = p
		rep.Points = append(rep.Points, BucketPoint{Start: key})
		return p
	}

	// Closed orders → revenue, guests, count.
	ofilter := bson.M{"restaurantId": restaurantID, "status": domain.OrderClosed}
	orng := bson.M{}
	if !from.IsZero() {
		orng["$gte"] = from
	}
	if !to.IsZero() {
		orng["$lt"] = to
	}
	if len(orng) > 0 {
		ofilter["closedAt"] = orng
	}
	ocur, err := s.db.Collection("orders").Find(ctx, ofilter,
		options.Find().SetProjection(bson.M{"closedAt": 1, "grandTotal": 1, "items": 1}))
	if err != nil {
		return nil, fmt.Errorf("find closed orders: %w", err)
	}
	var orders []domain.Order
	if err := ocur.All(ctx, &orders); err != nil {
		return nil, fmt.Errorf("decode closed orders: %w", err)
	}
	for _, o := range orders {
		if o.ClosedAt == nil {
			continue
		}
		p := get(*o.ClosedAt)
		p.Revenue += o.GrandTotal
		p.OrderCount++
		rep.Revenue += o.GrandTotal
		rep.OrderCount++
		for _, it := range o.Items {
			if it.IsFix && it.Status != domain.ItemVoided && it.Status != domain.ItemRefunded {
				p.Guests += it.Qty
				rep.Guests += it.Qty
			}
		}
	}

	// Expenses → expense side, by spentAt.
	efilter := bson.M{"restaurantId": restaurantID}
	erng := bson.M{}
	if !from.IsZero() {
		erng["$gte"] = from
	}
	if !to.IsZero() {
		erng["$lt"] = to
	}
	if len(erng) > 0 {
		efilter["spentAt"] = erng
	}
	ecur, err := s.db.Collection("expenses").Find(ctx, efilter,
		options.Find().SetProjection(bson.M{"spentAt": 1, "amount": 1}))
	if err != nil {
		return nil, fmt.Errorf("find expenses: %w", err)
	}
	var exps []struct {
		SpentAt time.Time    `bson:"spentAt"`
		Amount  domain.Kurus `bson:"amount"`
	}
	if err := ecur.All(ctx, &exps); err != nil {
		return nil, fmt.Errorf("decode expenses: %w", err)
	}
	for _, e := range exps {
		get(e.SpentAt).Expense += e.Amount
		rep.Expense += e.Amount
	}

	// Sync the mutable point values back into the ordered slice.
	sort.Slice(rep.Points, func(i, j int) bool {
		return rep.Points[i].Start.Before(rep.Points[j].Start)
	})
	for i := range rep.Points {
		if p, ok := idx[rep.Points[i].Start.Unix()]; ok {
			rep.Points[i] = *p
		}
	}
	return rep, nil
}

// WaiterStat is one waiter's performance over a range: revenue on the tables
// they opened, how many orders (tables) they closed, and fiks kişi served.
type WaiterStat struct {
	WaiterID string       `json:"waiterId"`
	Name     string       `json:"name"`
	Revenue  domain.Kurus `json:"revenue"`
	Orders   int          `json:"orders"`
	Guests   int          `json:"guests"`
}

// WaiterStats aggregates closed orders in [from, to) by waiter, resolving names
// from the waiters collection. Sorted by revenue desc.
func (s *Service) WaiterStats(ctx context.Context, restaurantID bson.ObjectID, from, to time.Time) ([]WaiterStat, error) {
	filter := bson.M{"restaurantId": restaurantID, "status": domain.OrderClosed}
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
		options.Find().SetProjection(bson.M{"waiterId": 1, "grandTotal": 1, "items": 1}))
	if err != nil {
		return nil, fmt.Errorf("find closed orders: %w", err)
	}
	var orders []domain.Order
	if err := cur.All(ctx, &orders); err != nil {
		return nil, fmt.Errorf("decode closed orders: %w", err)
	}

	acc := map[bson.ObjectID]*WaiterStat{}
	for _, o := range orders {
		st, ok := acc[o.WaiterID]
		if !ok {
			st = &WaiterStat{WaiterID: o.WaiterID.Hex()}
			acc[o.WaiterID] = st
		}
		st.Revenue += o.GrandTotal
		st.Orders++
		for _, it := range o.Items {
			if it.IsFix && it.Status != domain.ItemVoided && it.Status != domain.ItemRefunded {
				st.Guests += it.Qty
			}
		}
	}
	if len(acc) == 0 {
		return []WaiterStat{}, nil
	}

	// Resolve names.
	ids := make([]bson.ObjectID, 0, len(acc))
	for id := range acc {
		ids = append(ids, id)
	}
	wcur, err := s.db.Collection("waiters").Find(ctx,
		bson.M{"restaurantId": restaurantID, "_id": bson.M{"$in": ids}},
		options.Find().SetProjection(bson.M{"name": 1}))
	if err != nil {
		return nil, fmt.Errorf("find waiters: %w", err)
	}
	var waiters []struct {
		ID   bson.ObjectID `bson:"_id"`
		Name string        `bson:"name"`
	}
	if err := wcur.All(ctx, &waiters); err != nil {
		return nil, fmt.Errorf("decode waiters: %w", err)
	}
	names := make(map[bson.ObjectID]string, len(waiters))
	for _, w := range waiters {
		names[w.ID] = w.Name
	}

	// Orders whose waiterId doesn't resolve to a current waiter (opened at the
	// register, or a since-deleted waiter) collapse into one "Kasa Açtı" row.
	out := make([]WaiterStat, 0, len(acc))
	kasa := WaiterStat{Name: "Kasa Açtı"}
	for id, st := range acc {
		if name, ok := names[id]; ok {
			st.Name = name
			out = append(out, *st)
		} else {
			kasa.Revenue += st.Revenue
			kasa.Orders += st.Orders
			kasa.Guests += st.Guests
		}
	}
	if kasa.Orders > 0 {
		out = append(out, kasa)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Revenue != out[j].Revenue {
			return out[i].Revenue > out[j].Revenue
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

// sumExpenses totals expense amounts whose spentAt is in [from, to).
func (s *Service) sumExpenses(ctx context.Context, restaurantID bson.ObjectID, from, to time.Time) (domain.Kurus, error) {
	filter := bson.M{"restaurantId": restaurantID}
	rng := bson.M{}
	if !from.IsZero() {
		rng["$gte"] = from
	}
	if !to.IsZero() {
		rng["$lt"] = to
	}
	if len(rng) > 0 {
		filter["spentAt"] = rng
	}
	cur, err := s.db.Collection("expenses").Find(ctx, filter,
		options.Find().SetProjection(bson.M{"amount": 1}))
	if err != nil {
		return 0, fmt.Errorf("find expenses: %w", err)
	}
	var rows []struct {
		Amount domain.Kurus `bson:"amount"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return 0, fmt.Errorf("decode expenses: %w", err)
	}
	var total domain.Kurus
	for _, r := range rows {
		total += r.Amount
	}
	return total, nil
}

// outstanding sums (amount - paid) across every doc in a collection that carries
// an amount + payments[] (expenses and receivables share this shape). It's a
// current snapshot — NOT date-scoped — so open balances reflect reality now.
func (s *Service) outstanding(ctx context.Context, coll string, restaurantID bson.ObjectID) (domain.Kurus, error) {
	cur, err := s.db.Collection(coll).Find(ctx,
		bson.M{"restaurantId": restaurantID},
		options.Find().SetProjection(bson.M{"amount": 1, "payments": 1}))
	if err != nil {
		return 0, fmt.Errorf("find %s: %w", coll, err)
	}
	var rows []struct {
		Amount   domain.Kurus `bson:"amount"`
		Payments []struct {
			Amount domain.Kurus `bson:"amount"`
		} `bson:"payments"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return 0, fmt.Errorf("decode %s: %w", coll, err)
	}
	var open domain.Kurus
	for _, r := range rows {
		paid := domain.Kurus(0)
		for _, p := range r.Payments {
			paid += p.Amount
		}
		open += r.Amount - paid
	}
	return open, nil
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

// --- end-of-day report print ----------------------------------------------

// reportPrint is the wire format published on report/print for the bridge.
// Keep in sync with the bridge reportPrintMsg. Money is integer kuruş.
type reportPrint struct {
	Title          string                  `json:"title"`
	RangeLabel     string                  `json:"rangeLabel"`
	Revenue        domain.Kurus            `json:"revenue"`
	OrderCount     int                     `json:"orderCount"`
	Payment        map[string]domain.Kurus `json:"payment"`
	KDV            map[string]domain.Kurus `json:"kdv"`
	OTV            domain.Kurus            `json:"otv"`
	Expense        domain.Kurus            `json:"expense"`
	Profit         domain.Kurus            `json:"profit"`
	OpenReceivable domain.Kurus            `json:"openReceivable"`
	OpenPayable    domain.Kurus            `json:"openPayable"`
	TopItems       []reportItem            `json:"topItems"`
}

type reportItem struct {
	Name string `json:"name"`
	Qty  int    `json:"qty"`
}

// PrintReport builds the report for the range and publishes it to the register
// bridge, which prints it on the report printer (58mm). title/rangeLabel are the
// human strings shown at the top of the printout.
func (s *Service) PrintReport(ctx context.Context, restaurantID bson.ObjectID, from, to time.Time, title, rangeLabel string) error {
	if s.pub == nil {
		return ErrValidation{"Yazıcı servisi kapalı (MQTT yok)"}
	}
	rep, err := s.Sales(ctx, restaurantID, from, to)
	if err != nil {
		return err
	}

	msg := reportPrint{
		Title:          title,
		RangeLabel:     rangeLabel,
		Revenue:        rep.Revenue,
		OrderCount:     rep.OrderCount,
		Payment:        rep.Payment,
		KDV:            rep.KDV,
		OTV:            rep.OTV,
		Expense:        rep.Expense,
		Profit:         rep.Profit,
		OpenReceivable: rep.OpenReceivable,
		OpenPayable:    rep.OpenPayable,
	}
	// Top 10 items on the printout (keep paper reasonable).
	for i, it := range rep.TopItems {
		if i >= 10 {
			break
		}
		msg.TopItems = append(msg.TopItems, reportItem{Name: it.Name, Qty: it.Qty})
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("report marshal: %w", err)
	}
	if err := s.pub.Publish(mqttx.ReportPrint(restaurantID.Hex()), mqttx.QoSAtLeastOnce, false, payload); err != nil {
		return fmt.Errorf("report publish: %w", err)
	}
	slog.Info("report printed", "range", rangeLabel)
	return nil
}
