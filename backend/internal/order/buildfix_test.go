package order

import (
	"testing"
	"time"

	"github.com/ace-solutions/restaurant-backend/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Fiks menü composition is an UPPER LIMIT: 2 meze per person is a max, guests
// may take fewer to avoid waste, but never more.
func TestBuildFixMaxLimit(t *testing.T) {
	mezeCat := bson.NewObjectID()
	salataCat := bson.NewObjectID()

	// menu: a few mezes, one salata, and the fix parent.
	m1, m2, m3 := bson.NewObjectID(), bson.NewObjectID(), bson.NewObjectID()
	sal := bson.NewObjectID()
	fixID := bson.NewObjectID()

	menu := map[string]domain.MenuItem{
		m1.Hex():  {ID: m1, CategoryID: mezeCat, Name: "Humus", KDVOrani: 10},
		m2.Hex():  {ID: m2, CategoryID: mezeCat, Name: "Haydari", KDVOrani: 10},
		m3.Hex():  {ID: m3, CategoryID: mezeCat, Name: "Sigara", KDVOrani: 10},
		sal.Hex(): {ID: sal, CategoryID: salataCat, Name: "Coban", KDVOrani: 10},
	}
	fix := domain.MenuItem{
		ID: fixID, Name: "Fiks", Price: 50000, KDVOrani: 10, IsFix: true,
		FixIncludes: []domain.FixComponent{
			{CategoryID: mezeCat, Count: 2, PerPeople: 1}, // 2 meze / kişi
			{CategoryID: salataCat, Count: 1, PerPeople: 4},
		},
	}

	repeat := func(id bson.ObjectID, n int) []string {
		out := make([]string, n)
		for i := range out {
			out[i] = id.Hex()
		}
		return out
	}

	cases := []struct {
		name    string
		people  int
		meze    int
		salata  int
		wantErr bool
	}{
		{"5 kişi tam limit: 10 meze + 2 salata", 5, 10, 2, false},
		{"5 kişi az meze (ziyan olmasın): 8 meze + 2 salata", 5, 8, 2, false},
		{"5 kişi fazla meze: 11 meze reddedilir", 5, 11, 2, true},
		{"az salata da olur: 5 kişi 8 meze + 1 salata", 5, 8, 1, false},
		{"hiç içerik yok: reddedilir", 5, 0, 0, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ids := append(repeat(m1, c.meze), repeat(sal, c.salata)...)
			in := AddItemInput{MenuItemID: fixID.Hex(), Qty: c.people, FixIncludedItemIDs: ids}
			_, err := buildFix(in, fix, menu, bson.NewObjectID(), time.Now())
			if c.wantErr && err == nil {
				t.Errorf("beklenen hata alınmadı")
			}
			if !c.wantErr && err != nil {
				t.Errorf("beklenmeyen hata: %v", err)
			}
		})
	}
}
