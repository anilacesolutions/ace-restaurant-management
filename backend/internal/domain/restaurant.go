package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Restaurant represents a single restaurant tenant. Even though the MVP
// is single-restaurant, every other document carries RestaurantID so
// multi-tenancy is a config flip later.
type Restaurant struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Name      string        `bson:"name" json:"name"`
	Timezone  string        `bson:"timezone" json:"timezone"` // IANA, e.g. "Europe/Istanbul"
	CreatedAt time.Time     `bson:"createdAt" json:"createdAt"`
}

// Table is a physical table in a restaurant.
type Table struct {
	ID           bson.ObjectID `bson:"_id,omitempty" json:"id"`
	RestaurantID bson.ObjectID `bson:"restaurantId" json:"restaurantId"`
	Number       int           `bson:"number" json:"number"`         // human-facing table number
	Label        string        `bson:"label,omitempty" json:"label"` // optional ("Teras 3")
	Seats        int           `bson:"seats,omitempty" json:"seats"`
	Active       bool          `bson:"active" json:"active"`
}
