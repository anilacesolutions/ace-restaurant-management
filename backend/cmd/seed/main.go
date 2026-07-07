// Seed script for development. Creates a demo restaurant with tables,
// categories, and a realistic Turkish menu including correct KDV/ÖTV.
//
// Usage:
//
//	go run ./cmd/seed              # idempotent: skips if demo restaurant exists
//	go run ./cmd/seed -reset       # wipes demo data first, then re-seeds
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/ace-solutions/restaurant-backend/internal/auth"
	"github.com/ace-solutions/restaurant-backend/internal/config"
	"github.com/ace-solutions/restaurant-backend/internal/db"
	"github.com/ace-solutions/restaurant-backend/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	demoRestaurantName = "Demo Restoran"

	// Default admin credentials for the demo restaurant. The password is
	// committed intentionally — it is the initial credential the operator
	// uses on the cashier terminal and will be changeable from the admin
	// panel later. Anyone with shell access could read .env anyway.
	defaultAdminUsername = "admin"
	defaultAdminPassword = "gunguzelbahce123!!"
)

func main() {
	reset := flag.Bool("reset", false, "Wipe demo restaurant data and re-seed")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "err", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	mongo, err := db.Connect(ctx, cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		slog.Error("mongo connect failed", "err", err)
		os.Exit(1)
	}
	defer func() {
		sCtx, c := context.WithTimeout(context.Background(), 5*time.Second)
		defer c()
		_ = mongo.Close(sCtx)
	}()

	if err := mongo.EnsureIndexes(ctx); err != nil {
		slog.Error("ensure indexes failed", "err", err)
		os.Exit(1)
	}

	restaurants := mongo.DB.Collection("restaurants")
	tables := mongo.DB.Collection("tables")
	categories := mongo.DB.Collection("menuCategories")
	items := mongo.DB.Collection("menuItems")

	// Stable ID across -reset runs so DEFAULT_RESTAURANT_ID in .env never
	// goes stale. Only the children (tables, menu, orders) are wiped.
	var existing domain.Restaurant
	err = restaurants.FindOne(ctx, bson.M{"name": demoRestaurantName}).Decode(&existing)
	var restaurantID bson.ObjectID
	switch {
	case err == nil:
		restaurantID = existing.ID
		if !*reset {
			slog.Info("demo restaurant exists — pass -reset to wipe children and re-seed",
				"restaurantId", restaurantID.Hex())
			fmt.Printf("\nRESTAURANT_ID=%s\n", restaurantID.Hex())
			return
		}
		slog.Info("wiping demo restaurant children", "restaurantId", restaurantID.Hex())
		for _, coll := range []string{"tables", "menuCategories", "menuItems", "orders", "users", "waiters", "sessions", "qrLoginTokens"} {
			if _, err := mongo.DB.Collection(coll).DeleteMany(ctx, bson.M{"restaurantId": restaurantID}); err != nil {
				slog.Error("wipe failed", "coll", coll, "err", err)
				os.Exit(1)
			}
		}
	default:
		restaurantID = bson.NewObjectID()
		r := domain.Restaurant{
			ID:        restaurantID,
			Name:      demoRestaurantName,
			Timezone:  "Europe/Istanbul",
			CreatedAt: time.Now(),
		}
		if _, err := restaurants.InsertOne(ctx, r); err != nil {
			slog.Error("insert restaurant failed", "err", err)
			os.Exit(1)
		}
	}

	// Tables 1..25
	var tableDocs []any
	for i := 1; i <= 25; i++ {
		tableDocs = append(tableDocs, domain.Table{
			ID:           bson.NewObjectID(),
			RestaurantID: restaurantID,
			Number:       i,
			Seats:        4,
			Active:       true,
		})
	}
	if _, err := tables.InsertMany(ctx, tableDocs); err != nil {
		slog.Error("insert tables failed", "err", err)
		os.Exit(1)
	}

	// Categories
	type catSpec struct {
		name      string
		sortOrder int
	}
	catSpecs := []catSpec{
		{"Meze", 1},
		{"Salata", 2},
		{"Ana Yemek", 3},
		{"Tatli", 4},
		{"Icecek", 5},
		{"Alkol", 6},
	}
	catIDs := make(map[string]bson.ObjectID, len(catSpecs))
	var catDocs []any
	for _, c := range catSpecs {
		id := bson.NewObjectID()
		catIDs[c.name] = id
		catDocs = append(catDocs, domain.MenuCategory{
			ID:           id,
			RestaurantID: restaurantID,
			Name:         c.name,
			SortOrder:    c.sortOrder,
			Active:       true,
		})
	}
	if _, err := categories.InsertMany(ctx, catDocs); err != nil {
		slog.Error("insert categories failed", "err", err)
		os.Exit(1)
	}

	// Menu items
	type itemSpec struct {
		category string
		name     string
		price    domain.Kurus // KDV-included
	}
	// KDV 10 yemek/icecek, kitchen prints food (not drinks)
	specs := []itemSpec{
		{"Meze", "Humus", 8500},
		{"Meze", "Cacik", 6000},
		{"Meze", "Babagannus", 9000},
		{"Meze", "Sigara Boregi", 9500},
		{"Meze", "Patlican Salatasi", 8000},

		{"Salata", "Coban Salata", 12000},
		{"Salata", "Mevsim Salata", 13500},
		{"Salata", "Roka Salata", 14500},

		{"Ana Yemek", "Adana Kebap", 35000},
		{"Ana Yemek", "Urfa Kebap", 35000},
		{"Ana Yemek", "Karisik Izgara", 48000},
		{"Ana Yemek", "Tavuk Sis", 28000},
		{"Ana Yemek", "Kuzu Pirzola", 55000},
		{"Ana Yemek", "Iskender", 42000},

		{"Tatli", "Kunefe", 18000},
		{"Tatli", "Sutlac", 9500},
		{"Tatli", "Baklava", 14000},

		{"Icecek", "Ayran", 4500},
		{"Icecek", "Kola", 6000},
		{"Icecek", "Limonata", 7500},
		{"Icecek", "Su", 2000},
		{"Icecek", "Caydanlik (kucuk)", 12000},
	}
	var itemDocs []any
	for i, s := range specs {
		kitchen := s.category != "Icecek"
		dept := "A" // food → A
		if s.category == "Icecek" {
			dept = "B"
		}
		itemDocs = append(itemDocs, domain.MenuItem{
			ID:               bson.NewObjectID(),
			RestaurantID:     restaurantID,
			CategoryID:       catIDs[s.category],
			Name:             s.name,
			Price:            s.price,
			KDVOrani:         10,
			OTVVar:           false,
			POSDepartmanKodu: dept,
			Available:        true,
			SortOrder:        i,
			KitchenPrint:     kitchen,
		})
	}
	// Alcohol — KDV 20, OTV included
	alcohol := []itemSpec{
		{"Alkol", "Efes Bira (50cl)", 12000},
		{"Alkol", "Kavaklidere Kadeh Sarap", 18000},
		{"Alkol", "Yeni Raki Kadeh", 15000},
	}
	for i, s := range alcohol {
		itemDocs = append(itemDocs, domain.MenuItem{
			ID:               bson.NewObjectID(),
			RestaurantID:     restaurantID,
			CategoryID:       catIDs[s.category],
			Name:             s.name,
			Price:            s.price,
			KDVOrani:         20,
			OTVVar:           true,
			POSDepartmanKodu: "C",
			Available:        true,
			SortOrder:        100 + i,
			KitchenPrint:     false,
		})
	}
	if _, err := items.InsertMany(ctx, itemDocs); err != nil {
		slog.Error("insert items failed", "err", err)
		os.Exit(1)
	}

	// Admin user
	users := mongo.DB.Collection("users")
	hash, err := auth.HashPassword(defaultAdminPassword)
	if err != nil {
		slog.Error("hash admin password failed", "err", err)
		os.Exit(1)
	}
	adminUser := domain.User{
		ID:           bson.NewObjectID(),
		RestaurantID: restaurantID,
		Username:     defaultAdminUsername,
		PasswordHash: hash,
		Role:         domain.RoleAdmin,
		CreatedAt:    time.Now(),
	}
	if _, err := users.InsertOne(ctx, adminUser); err != nil {
		slog.Error("insert admin failed", "err", err)
		os.Exit(1)
	}

	// Sample waiters — admin can add/remove from the panel later.
	waiters := mongo.DB.Collection("waiters")
	waiterDocs := []any{
		domain.Waiter{ID: bson.NewObjectID(), RestaurantID: restaurantID, Name: "Ali Yilmaz", Active: true},
		domain.Waiter{ID: bson.NewObjectID(), RestaurantID: restaurantID, Name: "Mehmet Demir", Active: true},
		domain.Waiter{ID: bson.NewObjectID(), RestaurantID: restaurantID, Name: "Ayse Kaya", Active: true},
		domain.Waiter{ID: bson.NewObjectID(), RestaurantID: restaurantID, Name: "Fatma Ozturk", Active: true},
	}
	if _, err := waiters.InsertMany(ctx, waiterDocs); err != nil {
		slog.Error("insert waiters failed", "err", err)
		os.Exit(1)
	}

	slog.Info("seed complete",
		"restaurantId", restaurantID.Hex(),
		"tables", len(tableDocs),
		"categories", len(catDocs),
		"items", len(itemDocs),
		"users", 1,
		"waiters", len(waiterDocs))
	fmt.Printf("\nRESTAURANT_ID=%s\n", restaurantID.Hex())
	fmt.Printf("ADMIN_USERNAME=%s\n", defaultAdminUsername)
	fmt.Printf("ADMIN_PASSWORD=%s\n", defaultAdminPassword)
}
