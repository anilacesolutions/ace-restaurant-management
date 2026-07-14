// Package receivable implements the alacaklar defteri — money owed TO the
// restaurant (veresiye / open customer tabs), tracked under yönetim and fully
// independent of tables/orders. Mirror of the expense package: an amount owed
// with collections (tahsilat) accruing against it.
package receivable

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

var ErrNotFound = errors.New("receivable not found")

// ErrValidation wraps a user-facing message (maps to HTTP 400).
type ErrValidation struct{ Msg string }

func (e ErrValidation) Error() string { return e.Msg }

const dateLayout = "2006-01-02"

type Service struct {
	db  *mongo.Database
	loc *time.Location // Istanbul — IssuedAt is a calendar day in local time
}

func New(db *mongo.Database, loc *time.Location) *Service {
	return &Service{db: db, loc: loc}
}

func (s *Service) coll() *mongo.Collection { return s.db.Collection("receivables") }

// Input is the create payload. IssuedAt is a "YYYY-MM-DD" calendar day.
type Input struct {
	PersonName string       `json:"personName"`
	Amount     domain.Kurus `json:"amount"`
	Note       string       `json:"note"`
	IssuedAt   string       `json:"issuedAt"`
}

// Create validates the input and inserts one receivable row.
func (s *Service) Create(ctx context.Context, restaurantID, createdBy bson.ObjectID, in Input, now time.Time) (*domain.Receivable, error) {
	name := strings.TrimSpace(in.PersonName)
	if name == "" {
		return nil, ErrValidation{"Kim borçlu, isim girilmeli"}
	}
	if in.Amount <= 0 {
		return nil, ErrValidation{"Tutar 0'dan büyük olmalı"}
	}
	issuedAt, err := time.ParseInLocation(dateLayout, strings.TrimSpace(in.IssuedAt), s.loc)
	if err != nil {
		return nil, ErrValidation{"Geçerli bir tarih girilmeli"}
	}

	r := domain.Receivable{
		ID:           bson.NewObjectID(),
		RestaurantID: restaurantID,
		PersonName:   name,
		Amount:       in.Amount,
		Note:         strings.TrimSpace(in.Note),
		IssuedAt:     issuedAt,
		CreatedBy:    createdBy,
		CreatedAt:    now,
	}
	if _, err := s.coll().InsertOne(ctx, r); err != nil {
		return nil, fmt.Errorf("insert receivable: %w", err)
	}
	return &r, nil
}

// List returns receivables in [from, to) newest-first. A zero from/to is open.
func (s *Service) List(ctx context.Context, restaurantID bson.ObjectID, from, to time.Time) ([]domain.Receivable, error) {
	filter := bson.M{"restaurantId": restaurantID}
	rng := bson.M{}
	if !from.IsZero() {
		rng["$gte"] = from
	}
	if !to.IsZero() {
		rng["$lt"] = to
	}
	if len(rng) > 0 {
		filter["issuedAt"] = rng
	}

	cur, err := s.coll().Find(ctx, filter,
		options.Find().SetSort(bson.D{{Key: "issuedAt", Value: -1}, {Key: "createdAt", Value: -1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("find receivables: %w", err)
	}
	out := []domain.Receivable{}
	if err := cur.All(ctx, &out); err != nil {
		return nil, fmt.Errorf("decode receivables: %w", err)
	}
	return out, nil
}

// Delete removes one receivable.
func (s *Service) Delete(ctx context.Context, restaurantID, id bson.ObjectID) error {
	res, err := s.coll().DeleteOne(ctx, bson.M{"_id": id, "restaurantId": restaurantID})
	if err != nil {
		return fmt.Errorf("delete receivable: %w", err)
	}
	if res.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}

// PaymentInput is the add-collection payload. PaidAt is a "YYYY-MM-DD" day.
type PaymentInput struct {
	Amount domain.Kurus `json:"amount"`
	PaidAt string       `json:"paidAt"`
	Note   string       `json:"note"`
}

// AddPayment appends a collection (tahsilat) to a receivable.
func (s *Service) AddPayment(ctx context.Context, restaurantID, id bson.ObjectID, in PaymentInput, now time.Time) (*domain.Receivable, error) {
	if in.Amount <= 0 {
		return nil, ErrValidation{"Tahsilat tutarı 0'dan büyük olmalı"}
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
			return nil, ErrValidation{"Geçerli bir tahsilat tarihi girilmeli"}
		}
		pay.PaidAt = t
	}

	res := s.coll().FindOneAndUpdate(ctx,
		bson.M{"_id": id, "restaurantId": restaurantID},
		bson.M{"$push": bson.M{"payments": pay}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)
	return decode(res)
}

// DeletePayment pulls one collection from a receivable.
func (s *Service) DeletePayment(ctx context.Context, restaurantID, id, paymentID bson.ObjectID) (*domain.Receivable, error) {
	res := s.coll().FindOneAndUpdate(ctx,
		bson.M{"_id": id, "restaurantId": restaurantID},
		bson.M{"$pull": bson.M{"payments": bson.M{"_id": paymentID}}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)
	return decode(res)
}

func decode(res *mongo.SingleResult) (*domain.Receivable, error) {
	var r domain.Receivable
	if err := res.Decode(&r); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("decode receivable: %w", err)
	}
	return &r, nil
}
