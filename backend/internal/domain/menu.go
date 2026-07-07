package domain

import "go.mongodb.org/mongo-driver/v2/bson"

type MenuCategory struct {
	ID           bson.ObjectID `bson:"_id,omitempty" json:"id"`
	RestaurantID bson.ObjectID `bson:"restaurantId" json:"restaurantId"`
	Name         string        `bson:"name" json:"name"` // "Meze", "Ana Yemek", "Icecek", "Alkol"
	SortOrder    int           `bson:"sortOrder" json:"sortOrder"`
	Active       bool          `bson:"active" json:"active"`
}

// MenuItem is a sellable item. Carries VAT info even though POS is Phase 2 —
// see top-level CLAUDE.md "KDV / VAT" for why this is non-negotiable.
type MenuItem struct {
	ID           bson.ObjectID `bson:"_id,omitempty" json:"id"`
	RestaurantID bson.ObjectID `bson:"restaurantId" json:"restaurantId"`
	CategoryID   bson.ObjectID `bson:"categoryId" json:"categoryId"`

	Name        string `bson:"name" json:"name"`
	Description string `bson:"description,omitempty" json:"description,omitempty"`
	Price       Kurus  `bson:"price" json:"price"` // KDV-included sale price in kurus

	// ImageURL points at the object in S3 (or a CDN in front of it). Empty when
	// no image was uploaded — the item is still fully sellable without one.
	ImageURL string `bson:"imageUrl,omitempty" json:"imageUrl,omitempty"`

	// KDV / ÖTV — required at creation; reports and POS depend on these.
	KDVOrani         int    `bson:"kdvOrani" json:"kdvOrani"`                 // 10, 20, ...
	OTVVar           bool   `bson:"otvVar" json:"otvVar"`                     // true for alcohol
	POSDepartmanKodu string `bson:"posDepartmanKodu" json:"posDepartmanKodu"` // "A".."H" — yazar kasa dept

	Available bool `bson:"available" json:"available"` // 86'd toggle
	SortOrder int  `bson:"sortOrder" json:"sortOrder"`

	// Prints on the kitchen ticket. Empty for things the kitchen doesn't make
	// (e.g. bottled drinks), so the bridge can skip the kitchen ticket.
	KitchenPrint bool `bson:"kitchenPrint" json:"kitchenPrint"`
}
