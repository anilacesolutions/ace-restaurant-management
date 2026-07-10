package printer

import (
	"bytes"
	_ "embed"
	"image/png"
	"sync"
)

//go:embed logo.png
var logoPNG []byte

// logo is decoded once into a 1-bit bitmap and the ready ESC/POS raster.
var (
	logoOnce   sync.Once
	logoRaster []byte    // ESC/POS GS v 0 raster (centered), empty if decode fails
	logoBits   [][]bool  // threshold matrix for ASCII preview / tests
	logoW      int
	logoH      int
)

// luminance threshold: pixels darker than this print as black dots.
const logoThreshold = 140

func loadLogo() {
	logoOnce.Do(func() {
		img, err := png.Decode(bytes.NewReader(logoPNG))
		if err != nil {
			return
		}
		b := img.Bounds()
		w := b.Dx()
		h := b.Dy()
		// width must be a multiple of 8 for the raster; trim the remainder.
		w -= w % 8
		if w == 0 {
			return
		}
		logoW, logoH = w, h

		logoBits = make([][]bool, h)
		for y := 0; y < h; y++ {
			row := make([]bool, w)
			for x := 0; x < w; x++ {
				r, g, bl, a := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
				if a>>8 < 128 { // transparent -> white
					continue
				}
				// perceived luminance (0..255)
				lum := (299*int(r>>8) + 587*int(g>>8) + 114*int(bl>>8)) / 1000
				if lum < logoThreshold {
					row[x] = true
				}
			}
			logoBits[y] = row
		}
		logoRaster = buildRaster(logoBits, w, h)
	})
}

// buildRaster packs the bitmap into an ESC/POS GS v 0 raster command, centred.
func buildRaster(bits [][]bool, w, h int) []byte {
	bytesPerRow := w / 8
	var out bytes.Buffer
	out.WriteString("\x1ba\x01")              // ESC a 1 — center
	out.WriteString("\x1dv0\x00")             // GS v 0, mode 0
	out.WriteByte(byte(bytesPerRow & 0xff))   // xL
	out.WriteByte(byte(bytesPerRow >> 8))     // xH
	out.WriteByte(byte(h & 0xff))             // yL
	out.WriteByte(byte(h >> 8))               // yH
	for y := 0; y < h; y++ {
		for xb := 0; xb < bytesPerRow; xb++ {
			var b byte
			for bit := 0; bit < 8; bit++ {
				if bits[y][xb*8+bit] {
					b |= 1 << (7 - bit)
				}
			}
			out.WriteByte(b)
		}
	}
	out.WriteString("\x1ba\x00") // ESC a 0 — left
	out.WriteString("\n")
	return out.Bytes()
}

// LogoRaster returns the ESC/POS bytes for the embedded logo (empty on failure).
func LogoRaster() []byte {
	loadLogo()
	return logoRaster
}
