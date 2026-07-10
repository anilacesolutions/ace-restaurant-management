package printer

import (
	"strings"
	"testing"
	"time"
)

func TestMoney(t *testing.T) {
	cases := map[int64]string{
		0:       "0,00",
		5:       "0,05",
		1500:    "15,00",
		187000:  "1.870,00",
		1234567: "12.345,67",
	}
	for in, want := range cases {
		if got := money(in); got != want {
			t.Errorf("money(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestFormatReceipt(t *testing.T) {
	p := New(ModeStdout, "", 32)
	out := string(p.formatReceipt(Receipt{
		RestaurantName: "Gün Güzelbahçe",
		TableNumber:    13,
		Items: []ReceiptLine{
			{Qty: 2, Name: "Humus", LineTotal: 17000},
			{Qty: 1, Name: "Köfte", LineTotal: 22000, Note: "az pişmiş"},
			{Qty: 3, Name: "Ayran", LineTotal: 9000},
		},
		KDV:        map[string]int64{"10": 4363, "20": 0},
		GrandTotal: 48000,
		PrintedAt:  time.Date(2026, 7, 10, 18, 45, 0, 0, time.UTC),
	}))

	for _, want := range []string{"GUN GUZELBAHCE", "MASA 13", "HUMUS", "KOFTE", "AZ PISMIS", "KDV %10", "TOPLAM", "480,00 TL"} {
		if !strings.Contains(out, want) {
			t.Errorf("receipt missing %q", want)
		}
	}
	// visual preview: go test -run TestFormatReceipt -v ./internal/printer
	t.Log("\n" + out)
}
