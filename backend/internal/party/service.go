// Package party manages the "cari / kişiler" master list — people and firms the
// restaurant owes money to (suppliers, landlord, a waiter's advance). Expenses
// reference a party; Summary aggregates each party's debt vs paid.
package party

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ace-solutions/restaurant-backend/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var ErrNotFound = errors.New("party not found")

// ErrValidation wraps a user-facing message (maps to HTTP 400).
type ErrValidation struct{ Msg string }

func (e ErrValidation) Error() string { return e.Msg }

type Service struct {
	db *mongo.Database
}

func New(db *mongo.Database) *Service { return &Service{db: db} }

func (s *Service) coll() *mongo.Collection { return s.db.Collection("parties") }

// Get returns one party by id (used to snapshot the name onto an expense).
func (s *Service) Get(ctx context.Context, restaurantID, id bson.ObjectID) (domain.Party, error) {
	var p domain.Party
	err := s.coll().FindOne(ctx, bson.M{"_id": id, "restaurantId": restaurantID}).Decode(&p)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return domain.Party{}, ErrNotFound
	}
	if err != nil {
		return domain.Party{}, fmt.Errorf("find party: %w", err)
	}
	return p, nil
}

// PartyWithSummary is a party plus its cari-hesap balance (kuruş). A cari is a
// two-sided ledger now: Borc is what WE still owe them (unpaid expenses), Alacak
// is what THEY still owe us (uncollected receivables). Net = Alacak - Borc:
// positive means they owe us, negative means we owe them.
type PartyWithSummary struct {
	domain.Party  `bson:",inline"`
	Borc          domain.Kurus `json:"borc"`   // we still owe them
	Alacak        domain.Kurus `json:"alacak"` // they still owe us
	Net           domain.Kurus `json:"net"`    // alacak - borc
	MovementCount int          `json:"movementCount"`
}

// List returns all active parties with their cari-hesap balance, sorted by name.
func (s *Service) List(ctx context.Context, restaurantID bson.ObjectID) ([]PartyWithSummary, error) {
	cur, err := s.coll().Find(ctx,
		bson.M{"restaurantId": restaurantID, "active": true},
		options.Find().SetSort(bson.D{{Key: "name", Value: 1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("find parties: %w", err)
	}
	var parties []domain.Party
	if err := cur.All(ctx, &parties); err != nil {
		return nil, fmt.Errorf("decode parties: %w", err)
	}

	sums, err := s.summaries(ctx, restaurantID)
	if err != nil {
		return nil, err
	}

	out := make([]PartyWithSummary, 0, len(parties))
	for _, p := range parties {
		row := PartyWithSummary{Party: p}
		if sm, ok := sums[p.ID]; ok {
			row.Borc = sm.borc
			row.Alacak = sm.alacak
			row.Net = sm.alacak - sm.borc
			row.MovementCount = sm.count
		}
		out = append(out, row)
	}
	return out, nil
}

type partySum struct {
	borc   domain.Kurus // unpaid expenses (we owe them)
	alacak domain.Kurus // uncollected receivables (they owe us)
	count  int          // total movements (expenses + receivables)
}

// remainingByParty sums (amount - paid) per partyId over a collection whose docs
// carry `amount` and a `payments` array. Used for both expenses (→ borç) and
// receivables (→ alacak); a cari balance is not date-scoped, so all time.
func (s *Service) remainingByParty(ctx context.Context, coll string, restaurantID bson.ObjectID) (map[bson.ObjectID]struct {
	rem   domain.Kurus
	count int
}, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"restaurantId": restaurantID,
			"partyId":      bson.M{"$exists": true, "$ne": nil},
		}}},
		{{Key: "$group", Value: bson.M{
			"_id":   "$partyId",
			"rem":   bson.M{"$sum": bson.M{"$subtract": bson.A{"$amount", bson.M{"$sum": "$payments.amount"}}}},
			"count": bson.M{"$sum": 1},
		}}},
	}
	cur, err := s.db.Collection(coll).Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("aggregate %s sums: %w", coll, err)
	}
	var rows []struct {
		ID    bson.ObjectID `bson:"_id"`
		Rem   domain.Kurus  `bson:"rem"`
		Count int           `bson:"count"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, fmt.Errorf("decode %s sums: %w", coll, err)
	}
	m := make(map[bson.ObjectID]struct {
		rem   domain.Kurus
		count int
	}, len(rows))
	for _, r := range rows {
		m[r.ID] = struct {
			rem   domain.Kurus
			count int
		}{rem: r.Rem, count: r.Count}
	}
	return m, nil
}

// summaries combines the expense (borç) and receivable (alacak) sides into one
// net balance per cari.
func (s *Service) summaries(ctx context.Context, restaurantID bson.ObjectID) (map[bson.ObjectID]partySum, error) {
	exp, err := s.remainingByParty(ctx, "expenses", restaurantID)
	if err != nil {
		return nil, err
	}
	rec, err := s.remainingByParty(ctx, "receivables", restaurantID)
	if err != nil {
		return nil, err
	}
	m := make(map[bson.ObjectID]partySum, len(exp)+len(rec))
	for id, e := range exp {
		sm := m[id]
		sm.borc += e.rem
		sm.count += e.count
		m[id] = sm
	}
	for id, r := range rec {
		sm := m[id]
		sm.alacak += r.rem
		sm.count += r.count
		m[id] = sm
	}
	return m, nil
}

// Ledger is a single cari's full statement: the party, its movements (expenses
// = money we owe, receivables = money owed to us), and the net balance.
type Ledger struct {
	Party       domain.Party        `json:"party"`
	Expenses    []domain.Expense    `json:"expenses"`
	Receivables []domain.Receivable `json:"receivables"`
	Borc        domain.Kurus        `json:"borc"`
	Alacak      domain.Kurus        `json:"alacak"`
	Net         domain.Kurus        `json:"net"`
}

// GetLedger returns one cari with all its expenses and receivables, newest-first.
func (s *Service) GetLedger(ctx context.Context, restaurantID, id bson.ObjectID) (*Ledger, error) {
	p, err := s.Get(ctx, restaurantID, id)
	if err != nil {
		return nil, err
	}
	l := &Ledger{Party: p, Expenses: []domain.Expense{}, Receivables: []domain.Receivable{}}

	expCur, err := s.db.Collection("expenses").Find(ctx,
		bson.M{"restaurantId": restaurantID, "partyId": id},
		options.Find().SetSort(bson.D{{Key: "spentAt", Value: -1}, {Key: "createdAt", Value: -1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("find cari expenses: %w", err)
	}
	if err := expCur.All(ctx, &l.Expenses); err != nil {
		return nil, fmt.Errorf("decode cari expenses: %w", err)
	}
	recCur, err := s.db.Collection("receivables").Find(ctx,
		bson.M{"restaurantId": restaurantID, "partyId": id},
		options.Find().SetSort(bson.D{{Key: "issuedAt", Value: -1}, {Key: "createdAt", Value: -1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("find cari receivables: %w", err)
	}
	if err := recCur.All(ctx, &l.Receivables); err != nil {
		return nil, fmt.Errorf("decode cari receivables: %w", err)
	}

	for _, e := range l.Expenses {
		paid := domain.Kurus(0)
		for _, pay := range e.Payments {
			paid += pay.Amount
		}
		l.Borc += e.Amount - paid
	}
	for _, r := range l.Receivables {
		coll := domain.Kurus(0)
		for _, pay := range r.Payments {
			coll += pay.Amount
		}
		l.Alacak += r.Amount - coll
	}
	l.Net = l.Alacak - l.Borc
	return l, nil
}

// Input is the create payload.
type Input struct {
	Name string `json:"name"`
	Note string `json:"note"`
}

// Create inserts a new party (active by default).
func (s *Service) Create(ctx context.Context, restaurantID bson.ObjectID, in Input, now time.Time) (domain.Party, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return domain.Party{}, ErrValidation{"İsim zorunlu"}
	}
	p := domain.Party{
		ID:           bson.NewObjectID(),
		RestaurantID: restaurantID,
		Name:         name,
		Note:         strings.TrimSpace(in.Note),
		Active:       true,
		CreatedAt:    now,
	}
	if _, err := s.coll().InsertOne(ctx, p); err != nil {
		return domain.Party{}, fmt.Errorf("insert party: %w", err)
	}
	return p, nil
}

// EnsureByName returns an active cari whose name matches (case-insensitive,
// trimmed), creating one if none exists. Used to backfill legacy free-text
// receivables onto a cari.
func (s *Service) EnsureByName(ctx context.Context, restaurantID bson.ObjectID, name string, now time.Time) (domain.Party, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.Party{}, ErrValidation{"İsim zorunlu"}
	}
	var p domain.Party
	err := s.coll().FindOne(ctx, bson.M{
		"restaurantId": restaurantID,
		"name":         bson.M{"$regex": "^" + regexp.QuoteMeta(name) + "$", "$options": "i"},
	}).Decode(&p)
	if err == nil {
		return p, nil
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return domain.Party{}, fmt.Errorf("find party by name: %w", err)
	}
	return s.Create(ctx, restaurantID, Input{Name: name}, now)
}

// Delete removes a party. Past expenses keep their snapshotted PartyName, so
// deleting a party never rewrites the ledger (it just drops from the picker and
// the summary list).
func (s *Service) Delete(ctx context.Context, restaurantID, id bson.ObjectID) error {
	res, err := s.coll().DeleteOne(ctx, bson.M{"_id": id, "restaurantId": restaurantID})
	if err != nil {
		return fmt.Errorf("delete party: %w", err)
	}
	if res.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}
