package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Expense is one row in the gider defteri — a dated cost. It covers both goods
// purchases (manav, kasap) and operating costs (kira, personel, fatura),
// separated by Category. Lump-sum by design: no line items and no VAT breakdown
// in the MVP (see CLAUDE.md "ERP scope"). The richer itemized Purchase/StockItem
// model stays reserved for later real-inventory work.
//
// Amount is the full cost (the debt). Payments accrue against it over time, so
// a cost entered this month can be paid down across later months. Remaining
// balance = Amount - sum(Payments).
type Expense struct {
	ID           bson.ObjectID `bson:"_id,omitempty" json:"id"`
	RestaurantID bson.ObjectID `bson:"restaurantId" json:"restaurantId"`
	Category     string        `bson:"category" json:"category"` // "Sebze-Meyve", "Kira", ...
	Amount       Kurus         `bson:"amount" json:"amount"`     // full cost / debt, KDV-included
	Supplier     string        `bson:"supplier,omitempty" json:"supplier,omitempty"`
	Note         string        `bson:"note,omitempty" json:"note,omitempty"`
	SpentAt      time.Time     `bson:"spentAt" json:"spentAt"` // the calendar day the cost was incurred
	Payments     []Payment     `bson:"payments,omitempty" json:"payments"`
	CreatedBy    bson.ObjectID `bson:"createdBy" json:"createdBy"`
	CreatedAt    time.Time     `bson:"createdAt" json:"createdAt"`
}

// Payment is one settlement against an Expense's debt.
type Payment struct {
	ID     bson.ObjectID `bson:"_id" json:"id"`
	Amount Kurus         `bson:"amount" json:"amount"`
	PaidAt time.Time     `bson:"paidAt" json:"paidAt"` // the calendar day the payment was made
	Note   string        `bson:"note,omitempty" json:"note,omitempty"`
}
