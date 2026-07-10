package waiters

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ace-solutions/restaurant-backend/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ErrNotFound is returned when a waiter to update/delete does not exist.
var ErrNotFound = errors.New("waiter not found")

// ErrValidation wraps a user-facing validation message (maps to HTTP 400).
type ErrValidation struct{ Msg string }

func (e ErrValidation) Error() string { return e.Msg }

type Service struct {
	db *mongo.Database
}

func New(db *mongo.Database) *Service { return &Service{db: db} }

func (s *Service) coll() *mongo.Collection { return s.db.Collection("waiters") }

// ListActive returns active waiters sorted by name — the dataset the admin
// picks from when issuing a QR.
func (s *Service) ListActive(ctx context.Context, restaurantID bson.ObjectID) ([]domain.Waiter, error) {
	return s.find(ctx, bson.M{"restaurantId": restaurantID, "active": true})
}

// List returns every waiter (active + passive) sorted by name — for the ERP
// waiter management screen.
func (s *Service) List(ctx context.Context, restaurantID bson.ObjectID) ([]domain.Waiter, error) {
	return s.find(ctx, bson.M{"restaurantId": restaurantID})
}

func (s *Service) find(ctx context.Context, filter bson.M) ([]domain.Waiter, error) {
	cur, err := s.coll().Find(ctx, filter,
		options.Find().SetSort(bson.D{{Key: "name", Value: 1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("find waiters: %w", err)
	}
	out := []domain.Waiter{}
	if err := cur.All(ctx, &out); err != nil {
		return nil, fmt.Errorf("decode waiters: %w", err)
	}
	return out, nil
}

// WaiterInput is the create/update payload.
type WaiterInput struct {
	Name   string `json:"name"`
	Phone  string `json:"phone"`
	Active bool   `json:"active"`
}

func (in *WaiterInput) normalize() error {
	in.Name = strings.TrimSpace(in.Name)
	in.Phone = strings.TrimSpace(in.Phone)
	if in.Name == "" {
		return ErrValidation{"Garson adı zorunlu"}
	}
	return nil
}

// Create inserts a new waiter. New waiters are active by default.
func (s *Service) Create(ctx context.Context, restaurantID bson.ObjectID, in WaiterInput) (domain.Waiter, error) {
	if err := in.normalize(); err != nil {
		return domain.Waiter{}, err
	}
	w := domain.Waiter{
		ID:           bson.NewObjectID(),
		RestaurantID: restaurantID,
		Name:         in.Name,
		Phone:        in.Phone,
		Active:       true,
	}
	if _, err := s.coll().InsertOne(ctx, w); err != nil {
		return domain.Waiter{}, fmt.Errorf("insert waiter: %w", err)
	}
	return w, nil
}

// Update replaces a waiter's editable fields (name, phone, active/passive).
func (s *Service) Update(ctx context.Context, restaurantID, id bson.ObjectID, in WaiterInput) (domain.Waiter, error) {
	if err := in.normalize(); err != nil {
		return domain.Waiter{}, err
	}
	res := s.coll().FindOneAndUpdate(ctx,
		bson.M{"_id": id, "restaurantId": restaurantID},
		bson.M{"$set": bson.M{
			"name":   in.Name,
			"phone":  in.Phone,
			"active": in.Active,
		}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)
	var updated domain.Waiter
	if err := res.Decode(&updated); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return domain.Waiter{}, ErrNotFound
		}
		return domain.Waiter{}, fmt.Errorf("update waiter: %w", err)
	}
	return updated, nil
}

// Delete permanently removes a waiter. Past orders reference waiterId only, so
// deleting a waiter never rewrites order history (attribution just goes blank).
func (s *Service) Delete(ctx context.Context, restaurantID, id bson.ObjectID) error {
	res, err := s.coll().DeleteOne(ctx, bson.M{"_id": id, "restaurantId": restaurantID})
	if err != nil {
		return fmt.Errorf("delete waiter: %w", err)
	}
	if res.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}
