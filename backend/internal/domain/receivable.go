package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Receivable is money owed TO the restaurant — the mirror of an Expense. Someone
// (a customer who left a tab, a veresiye) owes the restaurant; it's tracked here
// under yönetim, completely independent of tables/orders. Amount is the total
// owed; Payments are collections (tahsilat) accruing against it over time.
// Remaining = Amount - sum(Payments).
type Receivable struct {
	ID           bson.ObjectID `bson:"_id,omitempty" json:"id"`
	RestaurantID bson.ObjectID `bson:"restaurantId" json:"restaurantId"`
	PersonName   string        `bson:"personName" json:"personName"` // who owes us (free text)
	Amount       Kurus         `bson:"amount" json:"amount"`         // total owed to us
	Note         string        `bson:"note,omitempty" json:"note,omitempty"`
	IssuedAt     time.Time     `bson:"issuedAt" json:"issuedAt"` // day the receivable arose
	Payments     []Payment     `bson:"payments,omitempty" json:"payments"`
	CreatedBy    bson.ObjectID `bson:"createdBy" json:"createdBy"`
	CreatedAt    time.Time     `bson:"createdAt" json:"createdAt"`
}
