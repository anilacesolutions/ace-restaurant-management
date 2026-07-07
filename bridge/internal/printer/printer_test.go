package printer

import (
	"strings"
	"testing"
)

func TestAsciiCapsTurkishChars(t *testing.T) {
	cases := map[string]string{
		"köfte":     "KOFTE",
		"şiş":       "SIS",
		"İçecek":    "ICECEK",
		"Ayran":     "AYRAN",
		"ÇOK GÜZEL": "COK GUZEL",
	}
	for in, want := range cases {
		if got := asciiCaps(in); got != want {
			t.Errorf("asciiCaps(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFormatContainsItems(t *testing.T) {
	p := New(ModeStdout, "", 32)
	out := string(p.format(Ticket{
		Header:  "Masa 5 — Garson: Ali",
		OrderID: "abc123",
		Items: []LineItem{
			{Qty: 2, Name: "Köfte", Note: "az tuzlu"},
			{Qty: 1, Name: "Ayran"},
		},
	}))
	for _, want := range []string{"MASA 5", "KOFTE", "AZ TUZLU", "AYRAN", "abc123"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n---\n%s", want, out)
		}
	}
}
