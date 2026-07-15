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
	// A receivable now hangs off a cari (Party), same as an expense — so one
	// person's debts and what we owe them live under a single balance. PartyName
	// is snapshotted so deleting the cari never rewrites history. PersonName is
	// the legacy free-text field kept for pre-cari rows.
	PartyID    bson.ObjectID `bson:"partyId,omitempty" json:"partyId,omitempty"`
	PartyName  string        `bson:"partyName,omitempty" json:"partyName,omitempty"`
	PersonName string        `bson:"personName,omitempty" json:"personName,omitempty"` // legacy free text
	Amount     Kurus         `bson:"amount" json:"amount"`                             // total owed to us
	Note       string        `bson:"note,omitempty" json:"note,omitempty"`
	IssuedAt   time.Time     `bson:"issuedAt" json:"issuedAt"` // day the receivable arose
	Payments   []Payment     `bson:"payments,omitempty" json:"payments"`
	CreatedBy  bson.ObjectID `bson:"createdBy" json:"createdBy"`
	CreatedAt  time.Time     `bson:"createdAt" json:"createdAt"`
}
