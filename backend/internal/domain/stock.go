package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// StockUnit is the unit of measure for a stock item.
type StockUnit string

const (
	UnitKg    StockUnit = "kg"
	UnitGr    StockUnit = "gr"
	UnitLt    StockUnit = "lt"
	UnitMl    StockUnit = "ml"
	UnitAdet  StockUnit = "adet"
	UnitKoli  StockUnit = "koli"
)

// StockItem is a raw or sellable inventory item ("marul", "kola 33cl").
// MVP keeps stock flat — no recipe expansion. A dish "Karısık Salata" does
// not auto-deduct lettuce. Recipe linking comes later.
type StockItem struct {
	ID           bson.ObjectID `bson:"_id,omitempty" json:"id"`
	RestaurantID bson.ObjectID `bson:"restaurantId" json:"restaurantId"`
	Name         string        `bson:"name" json:"name"`
	Unit         StockUnit     `bson:"unit" json:"unit"`
	CurrentQty   float64       `bson:"currentQty" json:"currentQty"` // can be fractional
	Active       bool          `bson:"active" json:"active"`
}

// Purchase is a stok girişi — "today bought 10kg marul for X TL".
type Purchase struct {
	ID           bson.ObjectID  `bson:"_id,omitempty" json:"id"`
	RestaurantID bson.ObjectID  `bson:"restaurantId" json:"restaurantId"`
	Supplier     string         `bson:"supplier,omitempty" json:"supplier,omitempty"`
	Items        []PurchaseItem `bson:"items" json:"items"`
	Total        Kurus          `bson:"total" json:"total"` // sum of (qty * unitCost)
	Note         string         `bson:"note,omitempty" json:"note,omitempty"`
	PurchasedAt  time.Time      `bson:"purchasedAt" json:"purchasedAt"`
	CreatedBy    bson.ObjectID  `bson:"createdBy" json:"createdBy"`
}

type PurchaseItem struct {
	StockItemID bson.ObjectID `bson:"stockItemId" json:"stockItemId"`
	Qty         float64       `bson:"qty" json:"qty"`           // in the item's Unit
	UnitCost    Kurus         `bson:"unitCost" json:"unitCost"` // KDV-included; cost reporting is a later concern
}

// StockMovement is an append-only ledger of stock changes.
// Qty > 0 = in, Qty < 0 = out. Reason ties it to the source document.
type StockMovement struct {
	ID           bson.ObjectID `bson:"_id,omitempty" json:"id"`
	RestaurantID bson.ObjectID `bson:"restaurantId" json:"restaurantId"`
	StockItemID  bson.ObjectID `bson:"stockItemId" json:"stockItemId"`
	Qty          float64       `bson:"qty" json:"qty"`
	Reason       string        `bson:"reason" json:"reason"` // "purchase", "sale", "waste", "adjust"
	RefID        bson.ObjectID `bson:"refId,omitempty" json:"refId,omitempty"`
	At           time.Time     `bson:"at" json:"at"`
}
