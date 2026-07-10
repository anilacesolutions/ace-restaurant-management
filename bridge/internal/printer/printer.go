// Package printer drives the thermal printer (kitchen tickets + customer
// adisyon receipts).
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
	ModeNetwork Mode = "network" // ESC/POS over TCP (addr host:port)
	ModeUSB     Mode = "usb"     // ESC/POS over raw device (Linux: /dev/usb/lp0)
	ModeWindows Mode = "windows" // ESC/POS via Windows spooler, RAW (addr = printer name)
)

type Printer struct {
	mode Mode
	addr string
	cols int
}

func New(mode Mode, addr string, cols int) *Printer {
	if cols <= 0 {
		cols = 32
	}
	return &Printer{mode: mode, addr: addr, cols: cols}
}

// --- kitchen ticket -------------------------------------------------------

// Ticket is the kitchen print data the bridge receives via MQTT.
type Ticket struct {
	Header    string     // e.g. "MASA 13 — GARSON: ALI"
	OrderID   string     // short id printed at the bottom
	Items     []LineItem // ordered
	Footer    string     // optional, e.g. "OZEL: AZ TUZLU"
	PrintedAt time.Time
}

type LineItem struct {
	Qty  int
	Name string
	Note string // optional, prints on a second line indented
}

// Print formats and sends a kitchen ticket. Cut at the end.
func (p *Printer) Print(t Ticket) error {
	return p.send(p.format(t))
}

func (p *Printer) format(t Ticket) []byte {
	var b strings.Builder
	p.init(&b)

	line := strings.Repeat("-", p.cols)
	b.WriteString(line + "\n")
	b.WriteString(asciiCaps(t.Header) + "\n")
	b.WriteString(line + "\n")

	for _, it := range t.Items {
		b.WriteString(fmt.Sprintf("%dx %s\n", it.Qty, asciiCaps(it.Name)))
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
	p.cut(&b)
	return []byte(b.String())
}

// --- customer adisyon (receipt) -------------------------------------------

// Receipt is the customer bill data the bridge receives via MQTT. Money fields
// are integer kuruş (no floats), matching the backend.
type Receipt struct {
	RestaurantName string
	TableNumber    int
	Items          []ReceiptLine
	KDV            map[string]int64 // rate ("10","20") -> tax portion in kuruş
	GrandTotal     int64            // kuruş, KDV-included gross
	PrintedAt      time.Time
}

type ReceiptLine struct {
	Qty       int
	Name      string
	LineTotal int64 // kuruş (unitPrice*qty)
	Note      string
}

// PrintReceipt formats and sends the customer adisyon.
func (p *Printer) PrintReceipt(r Receipt) error {
	return p.send(p.formatReceipt(r))
}

func (p *Printer) formatReceipt(r Receipt) []byte {
	var b strings.Builder
	p.init(&b)

	dash := strings.Repeat("-", p.cols)
	eq := strings.Repeat("=", p.cols)

	name := asciiCaps(r.RestaurantName)
	if name == "" {
		name = "ADISYON"
	}
	b.WriteString(center(name, p.cols) + "\n")
	b.WriteString(center(fmt.Sprintf("MASA %d", r.TableNumber), p.cols) + "\n")
	if !r.PrintedAt.IsZero() {
		b.WriteString(center(r.PrintedAt.Format("02.01.2006 15:04"), p.cols) + "\n")
	}
	b.WriteString(eq + "\n")

	for _, it := range r.Items {
		left := fmt.Sprintf("%dx %s", it.Qty, asciiCaps(it.Name))
		b.WriteString(row(left, money(it.LineTotal), p.cols) + "\n")
		if it.Note != "" {
			b.WriteString("   > " + asciiCaps(it.Note) + "\n")
		}
	}

	b.WriteString(dash + "\n")
	for _, rate := range []string{"1", "10", "20"} {
		if v, ok := r.KDV[rate]; ok && v > 0 {
			b.WriteString(row("KDV %"+rate, money(v), p.cols) + "\n")
		}
	}
	b.WriteString(dash + "\n")
	b.WriteString(row("TOPLAM", money(r.GrandTotal)+" TL", p.cols) + "\n")
	b.WriteString(eq + "\n")
	b.WriteString(center("AFIYET OLSUN", p.cols) + "\n")
	b.WriteString(center("ADISYON - MALI BELGE DEGILDIR", p.cols) + "\n")

	p.cut(&b)
	return []byte(b.String())
}

// --- shared ESC/POS + transport ------------------------------------------

func (p *Printer) init(b *strings.Builder) {
	if p.mode != ModeStdout {
		b.WriteString("\x1b@") // ESC @ — initialize
	}
}

func (p *Printer) cut(b *strings.Builder) {
	if p.mode != ModeStdout {
		b.WriteString("\n\n\n\n")
		b.WriteString("\x1dV\x00") // GS V 0 — full cut
	} else {
		b.WriteString("\n")
	}
}

// send opens the configured transport, writes the payload, and closes it.
func (p *Printer) send(body []byte) error {
	dst, err := p.open()
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = dst.Write(body)
	return err
}

func (p *Printer) open() (io.WriteCloser, error) {
	switch p.mode {
	case ModeStdout:
		return nopCloser{os.Stdout}, nil
	case ModeNetwork:
		c, err := net.DialTimeout("tcp", p.addr, 3*time.Second)
		if err != nil {
			return nil, fmt.Errorf("printer dial %q: %w", p.addr, err)
		}
		return c, nil
	case ModeUSB:
		f, err := os.OpenFile(p.addr, os.O_WRONLY, 0)
		if err != nil {
			return nil, fmt.Errorf("printer open %q: %w", p.addr, err)
		}
		return f, nil
	case ModeWindows:
		return openWindowsPrinter(p.addr) // build-tagged; real only on windows
	default:
		return nil, fmt.Errorf("unknown printer mode %q", p.mode)
	}
}

// --- text helpers ---------------------------------------------------------

// row lays out left text and a right-aligned value on a `cols`-wide line.
// If the left text is too long it is truncated so the value still fits.
func row(left, right string, cols int) string {
	if space := cols - len(right) - 1; len(left) > space {
		if space < 0 {
			space = 0
		}
		left = left[:space]
	}
	pad := cols - len(left) - len(right)
	if pad < 1 {
		pad = 1
	}
	return left + strings.Repeat(" ", pad) + right
}

func center(s string, cols int) string {
	if len(s) >= cols {
		return s
	}
	return strings.Repeat(" ", (cols-len(s))/2) + s
}

// money formats integer kuruş as Turkish currency: 187000 -> "1.870,00".
func money(k int64) string {
	neg := k < 0
	if neg {
		k = -k
	}
	lira := k / 100
	kurus := k % 100
	ls := fmt.Sprintf("%d", lira)
	var out strings.Builder
	for i, r := range ls {
		if i > 0 && (len(ls)-i)%3 == 0 {
			out.WriteByte('.')
		}
		out.WriteRune(r)
	}
	res := fmt.Sprintf("%s,%02d", out.String(), kurus)
	if neg {
		res = "-" + res
	}
	return res
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
				b.WriteRune('?') // never emit non-ASCII to the printer
			} else {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}

type nopCloser struct{ io.Writer }

func (nopCloser) Close() error { return nil }
