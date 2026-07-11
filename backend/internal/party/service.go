// Package party manages the "cari / kişiler" master list — people and firms the
// restaurant owes money to (suppliers, landlord, a waiter's advance). Expenses
// reference a party; Summary aggregates each party's debt vs paid.
package party

import (
	"context"
	"errors"
	"fmt"
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

// PartyWithSummary is a party plus its money totals (kuruş). Debt is the sum of
// expense amounts, Paid the sum of settlements, Remaining = Debt - Paid.
type PartyWithSummary struct {
	domain.Party `bson:",inline"`
	Debt         domain.Kurus `json:"debt"`
	Paid         domain.Kurus `json:"paid"`
	Remaining    domain.Kurus `json:"remaining"`
	ExpenseCount int          `json:"expenseCount"`
}

// List returns all active parties with their debt/paid summary, sorted by name.
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
			row.Debt = sm.debt
			row.Paid = sm.paid
			row.Remaining = sm.debt - sm.paid
			row.ExpenseCount = sm.count
		}
		out = append(out, row)
	}
	return out, nil
}

type partySum struct {
	debt  domain.Kurus
	paid  domain.Kurus
	count int
}

// summaries aggregates expenses by partyId across ALL time (a cari balance is
// not date-scoped). paid sums every payment on each expense.
func (s *Service) summaries(ctx context.Context, restaurantID bson.ObjectID) (map[bson.ObjectID]partySum, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"restaurantId": restaurantID,
			"partyId":      bson.M{"$exists": true, "$ne": nil},
		}}},
		{{Key: "$group", Value: bson.M{
			"_id":   "$partyId",
			"debt":  bson.M{"$sum": "$amount"},
			"paid":  bson.M{"$sum": bson.M{"$sum": "$payments.amount"}},
			"count": bson.M{"$sum": 1},
		}}},
	}
	cur, err := s.db.Collection("expenses").Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("aggregate party sums: %w", err)
	}
	var rows []struct {
		ID    bson.ObjectID `bson:"_id"`
		Debt  domain.Kurus  `bson:"debt"`
		Paid  domain.Kurus  `bson:"paid"`
		Count int           `bson:"count"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, fmt.Errorf("decode party sums: %w", err)
	}
	m := make(map[bson.ObjectID]partySum, len(rows))
	for _, r := range rows {
		m[r.ID] = partySum{debt: r.Debt, paid: r.Paid, count: r.Count}
	}
	return m, nil
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
