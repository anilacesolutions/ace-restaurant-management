//go:build !windows

package printer

import (
	"fmt"
	"io"
)

// openWindowsPrinter is a stub on non-Windows builds so the bridge still
// compiles for dev on macOS/Linux. The "windows" printer mode only works when
// the bridge actually runs on Windows (the register PC).
func openWindowsPrinter(_ string) (io.WriteCloser, error) {
	return nil, fmt.Errorf("windows printer mode requires building/running on Windows")
}
