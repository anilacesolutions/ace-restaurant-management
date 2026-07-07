// Package printer drives the kitchen thermal printer.
//
// Output is intentionally ASCII upper-case (KOFTE, KOLA, AYRAN) — see the
// top-level CLAUDE.md "Language and printing" section for the reason.
package printer

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"
	"unicode"
)

type Mode string

const (
	ModeStdout  Mode = "stdout"  // dev — write to stdout
	ModeNetwork Mode = "network" // ESC/POS over TCP
	ModeUSB     Mode = "usb"     // ESC/POS over raw device (Linux: /dev/usb/lp0)
)

type Printer struct {
	mode    Mode
	addr    string
	cols    int
	openNet func() (io.WriteCloser, error) // override in tests
}

func New(mode Mode, addr string, cols int) *Printer {
	return &Printer{mode: mode, addr: addr, cols: cols}
}

// Ticket is the print-ready data the bridge receives via MQTT.
type Ticket struct {
	Header   string      // e.g. "MASA 13 — GARSON: ALI"
	OrderID  string      // short id printed at the bottom
	Items    []LineItem  // ordered
	Footer   string      // optional, e.g. "OZEL: AZ TUZLU"
	PrintedAt time.Time
}

type LineItem struct {
	Qty  int
	Name string
	Note string // optional, prints on a second line indented
}

// Print formats and sends the ticket. Cut at the end. ESC/POS init prefix.
func (p *Printer) Print(t Ticket) error {
	body := p.format(t)

	var dst io.WriteCloser
	switch p.mode {
	case ModeStdout:
		dst = nopCloser{os.Stdout}
	case ModeNetwork:
		c, err := net.DialTimeout("tcp", p.addr, 3*time.Second)
		if err != nil {
			return fmt.Errorf("printer dial: %w", err)
		}
		dst = c
	case ModeUSB:
		f, err := os.OpenFile(p.addr, os.O_WRONLY, 0)
		if err != nil {
			return fmt.Errorf("printer open %q: %w", p.addr, err)
		}
		dst = f
	default:
		return fmt.Errorf("unknown printer mode %q", p.mode)
	}
	defer dst.Close()

	_, err := dst.Write(body)
	return err
}

func (p *Printer) format(t Ticket) []byte {
	var b strings.Builder

	// ESC @ — initialize printer
	if p.mode != ModeStdout {
		b.WriteString("\x1b@")
	}

	line := strings.Repeat("-", p.cols)
	b.WriteString(line + "\n")
	b.WriteString(asciiCaps(t.Header) + "\n")
	b.WriteString(line + "\n")

	for _, it := range t.Items {
		left := fmt.Sprintf("%dx %s", it.Qty, asciiCaps(it.Name))
		b.WriteString(left + "\n")
		if it.Note != "" {
			b.WriteString("    > " + asciiCaps(it.Note) + "\n")
		}
	}

	b.WriteString(line + "\n")
	if t.Footer != "" {
		b.WriteString(asciiCaps(t.Footer) + "\n")
	}
	b.WriteString("ID: " + t.OrderID + "\n")
	if !t.PrintedAt.IsZero() {
		b.WriteString(t.PrintedAt.Format("02.01.2006 15:04") + "\n")
	}

	// Feed + cut for ESC/POS modes
	if p.mode != ModeStdout {
		b.WriteString("\n\n\n\n")
		b.WriteString("\x1dV\x00") // GS V 0 — full cut
	} else {
		b.WriteString("\n")
	}

	return []byte(b.String())
}

// asciiCaps strips Turkish diacritics and upper-cases.
// CAPS letters degrade cleanly to ASCII (Ç→C, Ğ→G, İ→I, Ö→O, Ş→S, Ü→U).
func asciiCaps(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range strings.ToUpper(s) {
		switch r {
		case 'Ç':
			b.WriteRune('C')
		case 'Ğ':
			b.WriteRune('G')
		case 'İ', 'I':
			b.WriteRune('I')
		case 'Ö':
			b.WriteRune('O')
		case 'Ş':
			b.WriteRune('S')
		case 'Ü':
			b.WriteRune('U')
		default:
			if r > unicode.MaxASCII {
				// Fall back to '?' rather than emit non-ASCII to the printer
				b.WriteRune('?')
			} else {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}

type nopCloser struct{ io.Writer }

func (nopCloser) Close() error { return nil }
