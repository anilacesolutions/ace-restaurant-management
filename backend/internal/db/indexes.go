package db

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// EnsureIndexes creates the indexes the app expects. Idempotent.
// Add new index sets here as collections appear.
func (m *Mongo) EnsureIndexes(ctx context.Context) error {
	type spec struct {
		coll  string
		model mongo.IndexModel
	}

	unique := true
	specs := []spec{
		{"tables", mongo.IndexModel{
			Keys: bson.D{{Key: "restaurantId", Value: 1}, {Key: "number", Value: 1}},
			Options: &options.IndexOptionsBuilder{Opts: []func(*options.IndexOptions) error{
				func(o *options.IndexOptions) error { o.Unique = &unique; return nil },
			}},
		}},
		{"menuItems", mongo.IndexModel{
			Keys: bson.D{{Key: "restaurantId", Value: 1}, {Key: "categoryId", Value: 1}, {Key: "sortOrder", Value: 1}},
		}},
		{"orders", mongo.IndexModel{
			Keys: bson.D{{Key: "restaurantId", Value: 1}, {Key: "status", Value: 1}, {Key: "openedAt", Value: -1}},
		}},
		{"orders.byTable", mongo.IndexModel{
			Keys: bson.D{{Key: "restaurantId", Value: 1}, {Key: "tableNumber", Value: 1}, {Key: "status", Value: 1}},
		}},
		{"qrLoginTokens", mongo.IndexModel{
			Keys:    bson.D{{Key: "token", Value: 1}},
			Options: options.Index().SetUnique(true),
		}},
		{"qrLoginTokens.ttl", mongo.IndexModel{
			Keys:    bson.D{{Key: "expiresAt", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(0),
		}},
		{"users", mongo.IndexModel{
			Keys:    bson.D{{Key: "restaurantId", Value: 1}, {Key: "username", Value: 1}},
			Options: options.Index().SetUnique(true),
		}},
		{"waiters", mongo.IndexModel{
			Keys: bson.D{{Key: "restaurantId", Value: 1}, {Key: "active", Value: 1}},
		}},
		{"sessions", mongo.IndexModel{
			Keys:    bson.D{{Key: "token", Value: 1}},
			Options: options.Index().SetUnique(true),
		}},
		{"sessions.ttl", mongo.IndexModel{
			Keys:    bson.D{{Key: "expiresAt", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(0),
		}},
		{"sessions.bySubject", mongo.IndexModel{
			Keys: bson.D{{Key: "restaurantId", Value: 1}, {Key: "kind", Value: 1}, {Key: "subjectId", Value: 1}},
		}},
		{"stockMovements", mongo.IndexModel{
			Keys: bson.D{{Key: "restaurantId", Value: 1}, {Key: "stockItemId", Value: 1}, {Key: "at", Value: -1}},
		}},
		{"purchases", mongo.IndexModel{
			Keys: bson.D{{Key: "restaurantId", Value: 1}, {Key: "purchasedAt", Value: -1}},
		}},
		{"expenses", mongo.IndexModel{
			Keys: bson.D{{Key: "restaurantId", Value: 1}, {Key: "spentAt", Value: -1}},
		}},
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	for _, s := range specs {
		coll := splitColl(s.coll)
		if _, err := m.DB.Collection(coll).Indexes().CreateOne(ctx, s.model); err != nil {
			return fmt.Errorf("index on %s: %w", coll, err)
		}
	}
	return nil
}

// splitColl lets us write "orders.byTable" in the spec list to disambiguate
// multiple indexes on the same collection.
func splitColl(s string) string {
	for i, r := range s {
		if r == '.' {
			return s[:i]
		}
	}
	return s
}
