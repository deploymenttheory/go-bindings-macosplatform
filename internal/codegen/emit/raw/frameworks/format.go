package rawfw

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
)

// FormatGoSource runs src through go/format (canonical gofmt). It returns an
// error — failing generation loudly — when src is not valid Go, so an emitter
// bug surfaces at generation time rather than as broken output on disk.
func FormatGoSource(src []byte) ([]byte, error) {
	return format.Source(src)
}

// WriteGoFile formats src with go/format and writes the canonical result to
// path. It is the single choke point for every generated .go file so output is
// always gofmt-canonical and emitters/templates need not hand-manage whitespace,
// alignment, or import grouping.
func WriteGoFile(path string, src []byte) error {
	// Go ignores files beginning with an underscore, even in a valid package
	// such as _ScreenCaptureKit_SwiftUI. Keep the package path, but make its
	// generated declarations visible to the Go toolchain.
	if name := filepath.Base(path); strings.HasPrefix(name, "_") {
		path = filepath.Join(filepath.Dir(path), "sdk"+name)
	}
	formatted, err := format.Source(src)
	if err != nil {
		return fmt.Errorf("format %s: %w", path, err)
	}
	if err := os.WriteFile(path, formatted, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
