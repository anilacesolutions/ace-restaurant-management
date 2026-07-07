package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type OrderStatus string

const (
	OrderOpen      OrderStatus = "open"      // table opened, items being added
	OrderSent      OrderStatus = "sent"      // sent to kitchen
	OrderPreparing OrderStatus = "preparing" // kitchen accepted
	OrderReady     OrderStatus = "ready"     // ready to serve
	OrderServed    OrderStatus = "served"    // waiter delivered
	OrderClosed    OrderStatus = "closed"    // paid, table freed
	OrderCancelled OrderStatus = "cancelled"
)

type OrderItemStatus string

const (
	ItemNew      OrderItemStatus = "new"   // just added, not sent to kitchen yet
	ItemSent     OrderItemStatus = "sent"  // sent to kitchen
	ItemReady    OrderItemStatus = "ready" // kitchen finished
	ItemServed   OrderItemStatus = "served"
	ItemVoided   OrderItemStatus = "voided"   // before "sent" — silent removal
	ItemRefunded OrderItemStatus = "refunded" // after "sent" — needs reason
)

// Order is per table, per session — closes at payment.
type Order struct {
	ID           bson.ObjectID `bson:"_id,omitempty" json:"id"`
	RestaurantID bson.ObjectID `bson:"restaurantId" json:"restaurantId"`
	TableNumber  int           `bson:"tableNumber" json:"tableNumber"`
	WaiterID     bson.ObjectID `bson:"waiterId" json:"waiterId"`

	Status OrderStatus `bson:"status" json:"status"`
	Items  []OrderItem `bson:"items" json:"items"`

	// Totals are recomputed every time items change. Persisted to support
	// fast reads in lists and reports.
	Subtotal     Kurus            `bson:"subtotal" json:"subtotal"`
	KDVBreakdown map[string]Kurus `bson:"kdvBreakdown" json:"kdvBreakdown"` // key = "10", "20"
	OTV          Kurus            `bson:"otv" json:"otv"`
	GrandTotal   Kurus            `bson:"grandTotal" json:"grandTotal"`

	// PaymentMethod is set when the cashier closes the table ("nakit", "kart").
	PaymentMethod string `bson:"paymentMethod,omitempty" json:"paymentMethod,omitempty"`

	OpenedAt  time.Time  `bson:"openedAt" json:"openedAt"`
	ClosedAt  *time.Time `bson:"closedAt,omitempty" json:"closedAt,omitempty"`
	UpdatedAt time.Time  `bson:"updatedAt" json:"updatedAt"`
}

// OrderItem snapshots menu data at the time of order — prices and VAT rates
// must not change retroactively if the menu changes mid-shift.
type OrderItem struct {
	ID         bson.ObjectID `bson:"_id" json:"id"`
	MenuItemID bson.ObjectID `bson:"menuItemId" json:"menuItemId"`

	Name             string `bson:"name" json:"name"` // snapshot
	Qty              int    `bson:"qty" json:"qty"`
	UnitPrice        Kurus  `bson:"unitPrice" json:"unitPrice"`               // snapshot, KDV-included
	KDVOrani         int    `bson:"kdvOrani" json:"kdvOrani"`                 // snapshot
	OTVVar           bool   `bson:"otvVar" json:"otvVar"`                     // snapshot
	POSDepartmanKodu string `bson:"posDepartmanKodu" json:"posDepartmanKodu"` // snapshot

	Note   string          `bson:"note,omitempty" json:"note,omitempty"`
	Status OrderItemStatus `bson:"status" json:"status"`

	AddedAt  time.Time     `bson:"addedAt" json:"addedAt"`
	AddedBy  bson.ObjectID `bson:"addedBy" json:"addedBy"` // waiter id
	VoidedAt *time.Time    `bson:"voidedAt,omitempty" json:"voidedAt,omitempty"`
	VoidNote string        `bson:"voidNote,omitempty" json:"voidNote,omitempty"`
}
