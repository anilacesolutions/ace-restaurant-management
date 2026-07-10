//go:build windows

package printer

import (
	"fmt"
	"io"
	"syscall"
	"unsafe"
)

// Windows thermal printers connected over USB are driven through the print
// spooler in RAW mode: we send ESC/POS bytes straight to a named printer,
// bypassing any driver rendering. The printer must be installed in Windows
// (its own driver, or "Generic / Text Only"); PRINTER_ADDR is that exact name.

var (
	winspool           = syscall.NewLazyDLL("winspool.drv")
	procOpenPrinter    = winspool.NewProc("OpenPrinterW")
	procStartDocPrinter = winspool.NewProc("StartDocPrinterW")
	procStartPagePrinter = winspool.NewProc("StartPagePrinter")
	procWritePrinter   = winspool.NewProc("WritePrinter")
	procEndPagePrinter = winspool.NewProc("EndPagePrinter")
	procEndDocPrinter  = winspool.NewProc("EndDocPrinter")
	procClosePrinter   = winspool.NewProc("ClosePrinter")
)

type docInfo1 struct {
	pDocName    *uint16
	pOutputFile *uint16
	pDatatype   *uint16
}

type winPrinter struct {
	handle syscall.Handle
}

// openWindowsPrinter opens the named printer and starts a RAW document/page.
// Bytes written go straight to the printer; Close ends the job.
func openWindowsPrinter(name string) (io.WriteCloser, error) {
	if name == "" {
		return nil, fmt.Errorf("windows printer name (PRINTER_ADDR) is empty")
	}
	namePtr, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return nil, fmt.Errorf("printer name: %w", err)
	}

	var h syscall.Handle
	r, _, e := procOpenPrinter.Call(uintptr(unsafe.Pointer(namePtr)), uintptr(unsafe.Pointer(&h)), 0)
	if r == 0 {
		return nil, fmt.Errorf("OpenPrinter %q: %v", name, e)
	}

	docName, _ := syscall.UTF16PtrFromString("Adisyon")
	raw, _ := syscall.UTF16PtrFromString("RAW")
	di := docInfo1{pDocName: docName, pDatatype: raw}

	r, _, e = procStartDocPrinter.Call(uintptr(h), 1, uintptr(unsafe.Pointer(&di)))
	if r == 0 {
		procClosePrinter.Call(uintptr(h))
		return nil, fmt.Errorf("StartDocPrinter: %v", e)
	}
	r, _, e = procStartPagePrinter.Call(uintptr(h))
	if r == 0 {
		procEndDocPrinter.Call(uintptr(h))
		procClosePrinter.Call(uintptr(h))
		return nil, fmt.Errorf("StartPagePrinter: %v", e)
	}
	return &winPrinter{handle: h}, nil
}

func (w *winPrinter) Write(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}
	var written uint32
	r, _, e := procWritePrinter.Call(
		uintptr(w.handle),
		uintptr(unsafe.Pointer(&b[0])),
		uintptr(len(b)),
		uintptr(unsafe.Pointer(&written)),
	)
	if r == 0 {
		return int(written), fmt.Errorf("WritePrinter: %v", e)
	}
	return int(written), nil
}

func (w *winPrinter) Close() error {
	procEndPagePrinter.Call(uintptr(w.handle))
	procEndDocPrinter.Call(uintptr(w.handle))
	procClosePrinter.Call(uintptr(w.handle))
	return nil
}
