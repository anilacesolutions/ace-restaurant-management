package printer

import (
	"strings"
	"testing"
)

func TestLogoPreview(t *testing.T) {
	loadLogo()
	if logoW == 0 || logoH == 0 {
		t.Fatal("logo failed to decode")
	}
	t.Logf("logo %dx%d, raster %d bytes", logoW, logoH, len(logoRaster))

	// downscale to ~72 cols for an eyeball preview (block-sample)
	cols := 72
	step := logoW / cols
	if step < 1 {
		step = 1
	}
	var b strings.Builder
	b.WriteByte('\n')
	for y := 0; y < logoH; y += step * 2 { // *2 for char aspect ratio
		for x := 0; x < logoW; x += step {
			if logoBits[y][x] {
				b.WriteByte('#')
			} else {
				b.WriteByte(' ')
			}
		}
		b.WriteByte('\n')
	}
	t.Log(b.String())
}
