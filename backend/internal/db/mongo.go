package db

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Mongo struct {
	Client *mongo.Client
	DB     *mongo.Database
}

func Connect(ctx context.Context, uri, dbName string) (*Mongo, error) {
	// Atlas cold connects (SRV lookup + TLS handshake to 3 shard nodes) can
	// take 20s+ on the first attempt; mongosh-like CLIs hide this.
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	opts := options.Client().
		ApplyURI(uri).
		SetServerSelectionTimeout(30 * time.Second)

	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, fmt.Errorf("mongo connect: %w", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("mongo ping: %w", err)
	}
	return &Mongo{Client: client, DB: client.Database(dbName)}, nil
}

func (m *Mongo) Close(ctx context.Context) error {
	return m.Client.Disconnect(ctx)
}
