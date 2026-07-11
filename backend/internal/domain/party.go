package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Party is a person or firm the restaurant tracks money against in the gider
// defteri — a supplier (Ahmet Bey), a landlord, or a waiter taking an advance
// (avans). Expenses reference a Party so the owner can see, per person, how much
// is owed vs paid. Managed as a simple add/delete master list (no edit for MVP).
type Party struct {
	ID           bson.ObjectID `bson:"_id,omitempty" json:"id"`
	RestaurantID bson.ObjectID `bson:"restaurantId" json:"restaurantId"`
	Name         string        `bson:"name" json:"name"`
	Note         string        `bson:"note,omitempty" json:"note,omitempty"`
	Active       bool          `bson:"active" json:"active"`
	CreatedAt    time.Time     `bson:"createdAt" json:"createdAt"`
}
