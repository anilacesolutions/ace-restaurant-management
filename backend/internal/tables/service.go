package tables

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ace-solutions/restaurant-backend/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var ErrNotFound = errors.New("table not found")

type Service struct {
	db *mongo.Database
}

func New(db *mongo.Database) *Service { return &Service{db: db} }

func (s *Service) coll() *mongo.Collection { return s.db.Collection("tables") }

// List returns all tables for a restaurant, ordered by number.
func (s *Service) List(ctx context.Context, restaurantID bson.ObjectID) ([]domain.Table, error) {
	cur, err := s.coll().Find(ctx,
		bson.M{"restaurantId": restaurantID},
		options.Find().SetSort(bson.D{{Key: "number", Value: 1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("find tables: %w", err)
	}
	var out []domain.Table
	if err := cur.All(ctx, &out); err != nil {
		return nil, fmt.Errorf("decode tables: %w", err)
	}
	return out, nil
}

// AddNext picks the smallest free table number (1, 2, 3, ...) and inserts it.
// Returns the created table.
func (s *Service) AddNext(ctx context.Context, restaurantID bson.ObjectID) (domain.Table, error) {
	// Pull just the numbers — cheap even at hundreds of tables.
	cur, err := s.coll().Find(ctx,
		bson.M{"restaurantId": restaurantID},
		options.Find().SetProjection(bson.M{"number": 1, "_id": 0}).
			SetSort(bson.D{{Key: "number", Value: 1}}),
	)
	if err != nil {
		return domain.Table{}, fmt.Errorf("scan existing: %w", err)
	}
	var rows []struct {
		Number int `bson:"number"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return domain.Table{}, fmt.Errorf("decode existing: %w", err)
	}

	next := 1
	for _, r := range rows {
		if r.Number == next {
			next++
		} else if r.Number > next {
			break // gap found
		}
	}

	t := domain.Table{
		ID:           bson.NewObjectID(),
		RestaurantID: restaurantID,
		Number:       next,
		Seats:        4,
		Active:       true,
	}
	if _, err := s.coll().InsertOne(ctx, t); err != nil {
		return domain.Table{}, fmt.Errorf("insert table: %w", err)
	}
	return t, nil
}

// Delete removes a table by id. Per project decision, no check for open orders —
// table management is a setup-time concern (see top-level CLAUDE.md).
func (s *Service) Delete(ctx context.Context, restaurantID, id bson.ObjectID) error {
	res, err := s.coll().DeleteOne(ctx, bson.M{"_id": id, "restaurantId": restaurantID})
	if err != nil {
		return fmt.Errorf("delete table: %w", err)
	}
	if res.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}

// touch is a placeholder for future "last activity" tracking on a table.
// Kept here to satisfy import organization tools if added later.
var _ = time.Now
