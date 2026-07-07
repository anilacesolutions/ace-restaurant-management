package waiters

import (
	"context"
	"fmt"

	"github.com/ace-solutions/restaurant-backend/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Service struct {
	db *mongo.Database
}

func New(db *mongo.Database) *Service { return &Service{db: db} }

// ListActive returns active waiters sorted by name — the dataset the admin
// picks from when issuing a QR.
func (s *Service) ListActive(ctx context.Context, restaurantID bson.ObjectID) ([]domain.Waiter, error) {
	cur, err := s.db.Collection("waiters").Find(ctx,
		bson.M{"restaurantId": restaurantID, "active": true},
		options.Find().SetSort(bson.D{{Key: "name", Value: 1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("find waiters: %w", err)
	}
	var out []domain.Waiter
	if err := cur.All(ctx, &out); err != nil {
		return nil, fmt.Errorf("decode waiters: %w", err)
	}
	return out, nil
}
