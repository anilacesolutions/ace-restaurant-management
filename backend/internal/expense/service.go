// Package expense implements the gider defteri — a simple dated ledger of
// outgoing payments (goods purchases and operating costs alike).
package expense

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

// ErrNotFound is returned when a delete targets a missing expense.
var ErrNotFound = errors.New("expense not found")

// ErrValidation wraps a user-facing validation message (maps to HTTP 400).
type ErrValidation struct{ Msg string }

func (e ErrValidation) Error() string { return e.Msg }

const dateLayout = "2006-01-02"

// PartyResolver fetches a party so its display name can be snapshotted onto an
// expense (the party package satisfies this).
type PartyResolver interface {
	Get(ctx context.Context, restaurantID, id bson.ObjectID) (domain.Party, error)
}

type Service struct {
	db      *mongo.Database
	loc     *time.Location // Istanbul — SpentAt is a calendar day in local time
	parties PartyResolver
}

func New(db *mongo.Database, loc *time.Location, parties PartyResolver) *Service {
	return &Service{db: db, loc: loc, parties: parties}
}

func (s *Service) coll() *mongo.Collection { return s.db.Collection("expenses") }

// Input is the create payload. SpentAt is a "YYYY-MM-DD" calendar day. PartyID
// (optional) attaches the expense to a cari/kişi.
type Input struct {
	Category string       `json:"category"`
	Amount   domain.Kurus `json:"amount"`
	Supplier string       `json:"supplier"`
	PartyID  string       `json:"partyId"`
	Note     string       `json:"note"`
	SpentAt  string       `json:"spentAt"`
}

// Create validates the input and inserts one expense row.
func (s *Service) Create(ctx context.Context, restaurantID, createdBy bson.ObjectID, in Input, now time.Time) (*domain.Expense, error) {
	// Category (Sebze-Meyve, Kira...) was merged into the Cari/Kalemler list —
	// it's now optional (old rows keep theirs), the party/kalem is mandatory.
	category := strings.TrimSpace(in.Category)
	if in.Amount <= 0 {
		return nil, ErrValidation{"Tutar 0'dan büyük olmalı"}
	}
	spentAt, err := time.ParseInLocation(dateLayout, strings.TrimSpace(in.SpentAt), s.loc)
	if err != nil {
		return nil, ErrValidation{"Geçerli bir tarih girilmeli"}
	}

	// Cari / kalem is required — resolve it and snapshot its name onto the row.
	raw := strings.TrimSpace(in.PartyID)
	if raw == "" {
		return nil, ErrValidation{"Cari / kalem seçilmeli"}
	}
	pid, err := bson.ObjectIDFromHex(raw)
	if err != nil {
		return nil, ErrValidation{"Geçersiz cari / kalem"}
	}
	p, err := s.parties.Get(ctx, restaurantID, pid)
	if err != nil {
		return nil, ErrValidation{"Cari / kalem bulunamadı"}
	}
	partyID := p.ID
	partyName := p.Name

	exp := domain.Expense{
		ID:           bson.NewObjectID(),
		RestaurantID: restaurantID,
		Category:     category,
		Amount:       in.Amount,
		Supplier:     strings.TrimSpace(in.Supplier),
		PartyID:      partyID,
		PartyName:    partyName,
		Note:         strings.TrimSpace(in.Note),
		SpentAt:      spentAt,
		CreatedBy:    createdBy,
		CreatedAt:    now,
	}
	if _, err := s.coll().InsertOne(ctx, exp); err != nil {
		return nil, fmt.Errorf("insert expense: %w", err)
	}
	return &exp, nil
}

// List returns expenses in [from, to) newest-first. A zero from/to is open.
func (s *Service) List(ctx context.Context, restaurantID bson.ObjectID, from, to time.Time) ([]domain.Expense, error) {
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

	cur, err := s.coll().Find(ctx, filter,
		options.Find().SetSort(bson.D{{Key: "spentAt", Value: -1}, {Key: "createdAt", Value: -1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("find expenses: %w", err)
	}
	out := []domain.Expense{}
	if err := cur.All(ctx, &out); err != nil {
		return nil, fmt.Errorf("decode expenses: %w", err)
	}
	return out, nil
}

// Delete removes one expense. Bookkeeping is correctable — see CLAUDE.md.
func (s *Service) Delete(ctx context.Context, restaurantID, id bson.ObjectID) error {
	res, err := s.coll().DeleteOne(ctx, bson.M{"_id": id, "restaurantId": restaurantID})
	if err != nil {
		return fmt.Errorf("delete expense: %w", err)
	}
	if res.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}

// PaymentInput is the add-payment payload. PaidAt is a "YYYY-MM-DD" calendar day.
type PaymentInput struct {
	Amount domain.Kurus `json:"amount"`
	PaidAt string       `json:"paidAt"`
	Note   string       `json:"note"`
}

// AddPayment appends a settlement to an expense and returns the updated doc.
func (s *Service) AddPayment(ctx context.Context, restaurantID, expenseID bson.ObjectID, in PaymentInput, now time.Time) (*domain.Expense, error) {
	if in.Amount <= 0 {
		return nil, ErrValidation{"Ödeme tutarı 0'dan büyük olmalı"}
	}
	pay := domain.Payment{
		ID:     bson.NewObjectID(),
		Amount: in.Amount,
		PaidAt: now,
		Note:   strings.TrimSpace(in.Note),
	}
	if raw := strings.TrimSpace(in.PaidAt); raw != "" {
		t, err := time.ParseInLocation(dateLayout, raw, s.loc)
		if err != nil {
			return nil, ErrValidation{"Geçerli bir ödeme tarihi girilmeli"}
		}
		pay.PaidAt = t
	}

	res := s.coll().FindOneAndUpdate(ctx,
		bson.M{"_id": expenseID, "restaurantId": restaurantID},
		bson.M{"$push": bson.M{"payments": pay}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)
	return decodeExpense(res)
}

// DeletePayment pulls one settlement from an expense and returns the updated doc.
func (s *Service) DeletePayment(ctx context.Context, restaurantID, expenseID, paymentID bson.ObjectID) (*domain.Expense, error) {
	res := s.coll().FindOneAndUpdate(ctx,
		bson.M{"_id": expenseID, "restaurantId": restaurantID},
		bson.M{"$pull": bson.M{"payments": bson.M{"_id": paymentID}}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)
	return decodeExpense(res)
}

func decodeExpense(res *mongo.SingleResult) (*domain.Expense, error) {
	var exp domain.Expense
	if err := res.Decode(&exp); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("decode expense: %w", err)
	}
	return &exp, nil
}
