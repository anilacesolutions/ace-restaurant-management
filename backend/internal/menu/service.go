package menu

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/ace-solutions/restaurant-backend/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ErrNotFound is returned when an item to update does not exist.
var ErrNotFound = errors.New("menu item not found")

// ErrValidation wraps a user-facing validation message (maps to HTTP 400).
type ErrValidation struct{ Msg string }

func (e ErrValidation) Error() string { return e.Msg }

type Service struct {
	db *mongo.Database
}

func New(db *mongo.Database) *Service { return &Service{db: db} }

// CategoryWithItems is the API shape: a category followed by its items.
type CategoryWithItems struct {
	domain.MenuCategory `bson:",inline"`
	Items               []domain.MenuItem `json:"items"`
}

// GetMenu returns active categories with their available items, in sortOrder.
func (s *Service) GetMenu(ctx context.Context, restaurantID bson.ObjectID) ([]CategoryWithItems, error) {
	catCur, err := s.db.Collection("menuCategories").Find(ctx,
		bson.M{"restaurantId": restaurantID, "active": true},
		options.Find().SetSort(bson.D{{Key: "sortOrder", Value: 1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("find categories: %w", err)
	}
	var cats []domain.MenuCategory
	if err := catCur.All(ctx, &cats); err != nil {
		return nil, fmt.Errorf("decode categories: %w", err)
	}

	itemCur, err := s.db.Collection("menuItems").Find(ctx,
		bson.M{"restaurantId": restaurantID, "available": true},
		options.Find().SetSort(bson.D{{Key: "sortOrder", Value: 1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("find items: %w", err)
	}
	var items []domain.MenuItem
	if err := itemCur.All(ctx, &items); err != nil {
		return nil, fmt.Errorf("decode items: %w", err)
	}

	itemsByCat := make(map[bson.ObjectID][]domain.MenuItem, len(cats))
	for _, it := range items {
		itemsByCat[it.CategoryID] = append(itemsByCat[it.CategoryID], it)
	}

	return assemble(cats, items), nil
}

// ListAll returns every category and every item — including inactive
// categories and unavailable ("86'd") items — for the ERP management screen.
func (s *Service) ListAll(ctx context.Context, restaurantID bson.ObjectID) ([]CategoryWithItems, error) {
	catCur, err := s.db.Collection("menuCategories").Find(ctx,
		bson.M{"restaurantId": restaurantID},
		options.Find().SetSort(bson.D{{Key: "sortOrder", Value: 1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("find categories: %w", err)
	}
	var cats []domain.MenuCategory
	if err := catCur.All(ctx, &cats); err != nil {
		return nil, fmt.Errorf("decode categories: %w", err)
	}

	itemCur, err := s.db.Collection("menuItems").Find(ctx,
		bson.M{"restaurantId": restaurantID},
		options.Find().SetSort(bson.D{{Key: "sortOrder", Value: 1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("find items: %w", err)
	}
	var items []domain.MenuItem
	if err := itemCur.All(ctx, &items); err != nil {
		return nil, fmt.Errorf("decode items: %w", err)
	}

	return assemble(cats, items), nil
}

// assemble groups items under their category, preserving category sort order.
func assemble(cats []domain.MenuCategory, items []domain.MenuItem) []CategoryWithItems {
	itemsByCat := make(map[bson.ObjectID][]domain.MenuItem, len(cats))
	for _, it := range items {
		itemsByCat[it.CategoryID] = append(itemsByCat[it.CategoryID], it)
	}
	out := make([]CategoryWithItems, 0, len(cats))
	for _, c := range cats {
		its := itemsByCat[c.ID]
		if its == nil {
			// A category with no items must serialize as [] not null, or the
			// client crashes doing items.length / items.map on it.
			its = []domain.MenuItem{}
		}
		out = append(out, CategoryWithItems{
			MenuCategory: c,
			Items:        its,
		})
	}
	return out
}

// ItemInput is the create/update payload. CategoryName is resolved to (or
// creates) a category; the rest maps straight onto domain.MenuItem.
type ItemInput struct {
	CategoryName     string       `json:"categoryName"`
	Name             string       `json:"name"`
	Description      string       `json:"description"`
	Price            domain.Kurus `json:"price"`
	KDVOrani         int          `json:"kdvOrani"`
	OTVVar           bool         `json:"otvVar"`
	POSDepartmanKodu string       `json:"posDepartmanKodu"`
	KitchenPrint     bool         `json:"kitchenPrint"`
	ImageURL         string       `json:"imageUrl"`
	Available        bool         `json:"available"`

	// Fiks menü: when IsFix, FixIncludes defines the composition (category +
	// count per person).
	IsFix       bool              `json:"isFix"`
	FixIncludes []FixIncludeInput `json:"fixIncludes"`
}

type FixIncludeInput struct {
	CategoryID string `json:"categoryId"`
	Count      int    `json:"count"`
	PerPeople  int    `json:"perPeople"` // items per N people; 0/1 = per person
}

// fixComponents maps the validated input to domain (nil when not a fix).
func (in *ItemInput) fixComponents() []domain.FixComponent {
	if !in.IsFix {
		return nil
	}
	out := make([]domain.FixComponent, 0, len(in.FixIncludes))
	for _, c := range in.FixIncludes {
		id, _ := bson.ObjectIDFromHex(c.CategoryID) // validated in normalizeAndValidate
		per := c.PerPeople
		if per < 1 {
			per = 1
		}
		out = append(out, domain.FixComponent{CategoryID: id, Count: c.Count, PerPeople: per})
	}
	return out
}

var posKoduRe = regexp.MustCompile(`^[A-H]$`)

func (in *ItemInput) normalizeAndValidate() error {
	in.CategoryName = strings.TrimSpace(in.CategoryName)
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)
	in.POSDepartmanKodu = strings.ToUpper(strings.TrimSpace(in.POSDepartmanKodu))
	in.ImageURL = strings.TrimSpace(in.ImageURL)

	if in.CategoryName == "" {
		return ErrValidation{"Kategori zorunlu"}
	}
	if in.Name == "" {
		return ErrValidation{"Ürün adı zorunlu"}
	}
	if in.Price <= 0 {
		return ErrValidation{"Fiyat 0'dan büyük olmalı"}
	}
	if in.KDVOrani <= 0 {
		return ErrValidation{"KDV oranı seçilmeli"}
	}
	if in.POSDepartmanKodu != "" && !posKoduRe.MatchString(in.POSDepartmanKodu) {
		return ErrValidation{"POS departman kodu A-H arası olmalı"}
	}
	if in.IsFix {
		if len(in.FixIncludes) == 0 {
			return ErrValidation{"Fiks menü için en az bir içerik satırı gerekli"}
		}
		for _, c := range in.FixIncludes {
			if c.Count <= 0 {
				return ErrValidation{"Fiks içerik adedi 0'dan büyük olmalı"}
			}
			if _, err := bson.ObjectIDFromHex(c.CategoryID); err != nil {
				return ErrValidation{"Fiks içerik kategorisi geçersiz"}
			}
		}
	}
	return nil
}

// EnsureCategory returns the id of the category with the given name (case-
// insensitive), creating it if it does not exist. New categories go to the end.
func (s *Service) EnsureCategory(ctx context.Context, restaurantID bson.ObjectID, name string) (bson.ObjectID, error) {
	coll := s.db.Collection("menuCategories")
	// Case-insensitive exact match, anchored so "Ana" doesn't match "Ana Yemek".
	filter := bson.M{
		"restaurantId": restaurantID,
		"name":         bson.M{"$regex": "^" + regexp.QuoteMeta(name) + "$", "$options": "i"},
	}
	var existing domain.MenuCategory
	err := coll.FindOne(ctx, filter).Decode(&existing)
	if err == nil {
		return existing.ID, nil
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return bson.NilObjectID, fmt.Errorf("find category: %w", err)
	}

	n, err := coll.CountDocuments(ctx, bson.M{"restaurantId": restaurantID})
	if err != nil {
		return bson.NilObjectID, fmt.Errorf("count categories: %w", err)
	}
	cat := domain.MenuCategory{
		ID:           bson.NewObjectID(),
		RestaurantID: restaurantID,
		Name:         name,
		SortOrder:    int(n),
		Active:       true,
	}
	if _, err := coll.InsertOne(ctx, cat); err != nil {
		return bson.NilObjectID, fmt.Errorf("insert category: %w", err)
	}
	return cat.ID, nil
}

// CreateItem validates the input, resolves its category, and inserts the item.
func (s *Service) CreateItem(ctx context.Context, restaurantID bson.ObjectID, in ItemInput) (*domain.MenuItem, error) {
	if err := in.normalizeAndValidate(); err != nil {
		return nil, err
	}
	catID, err := s.EnsureCategory(ctx, restaurantID, in.CategoryName)
	if err != nil {
		return nil, err
	}

	items := s.db.Collection("menuItems")
	n, err := items.CountDocuments(ctx, bson.M{"restaurantId": restaurantID, "categoryId": catID})
	if err != nil {
		return nil, fmt.Errorf("count items: %w", err)
	}

	item := domain.MenuItem{
		ID:               bson.NewObjectID(),
		RestaurantID:     restaurantID,
		CategoryID:       catID,
		Name:             in.Name,
		Description:      in.Description,
		Price:            in.Price,
		KDVOrani:         in.KDVOrani,
		OTVVar:           in.OTVVar,
		POSDepartmanKodu: in.POSDepartmanKodu,
		Available:        in.Available,
		SortOrder:        int(n),
		KitchenPrint:     in.KitchenPrint,
		ImageURL:         in.ImageURL,
		IsFix:            in.IsFix,
		FixIncludes:      in.fixComponents(),
	}
	if _, err := items.InsertOne(ctx, item); err != nil {
		return nil, fmt.Errorf("insert item: %w", err)
	}
	return &item, nil
}

// DeleteItem permanently removes an item. Existing orders snapshot the item
// onto their lines, so deleting from the menu never rewrites order history.
func (s *Service) DeleteItem(ctx context.Context, restaurantID, itemID bson.ObjectID) error {
	res, err := s.db.Collection("menuItems").DeleteOne(ctx,
		bson.M{"_id": itemID, "restaurantId": restaurantID},
	)
	if err != nil {
		return fmt.Errorf("delete item: %w", err)
	}
	if res.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateItem replaces the editable fields of an existing item.
func (s *Service) UpdateItem(ctx context.Context, restaurantID, itemID bson.ObjectID, in ItemInput) (*domain.MenuItem, error) {
	if err := in.normalizeAndValidate(); err != nil {
		return nil, err
	}
	catID, err := s.EnsureCategory(ctx, restaurantID, in.CategoryName)
	if err != nil {
		return nil, err
	}

	items := s.db.Collection("menuItems")
	update := bson.M{"$set": bson.M{
		"categoryId":       catID,
		"name":             in.Name,
		"description":      in.Description,
		"price":            in.Price,
		"kdvOrani":         in.KDVOrani,
		"otvVar":           in.OTVVar,
		"posDepartmanKodu": in.POSDepartmanKodu,
		"available":        in.Available,
		"kitchenPrint":     in.KitchenPrint,
		"imageUrl":         in.ImageURL,
		"isFix":            in.IsFix,
		"fixIncludes":      in.fixComponents(),
	}}
	res := items.FindOneAndUpdate(ctx,
		bson.M{"_id": itemID, "restaurantId": restaurantID},
		update,
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)
	var updated domain.MenuItem
	if err := res.Decode(&updated); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("update item: %w", err)
	}
	return &updated, nil
}
