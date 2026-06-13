//go:build darwin

package scanner

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// errNoTBD is returned by loadTBDClasses when the .tbd file does not exist
// for this framework. Callers distinguish this from a parse error so that a
// corrupt or empty .tbd file triggers a warning rather than silent fallback.
var errNoTBD = errors.New("no .tbd file found")

// loadTBDClasses parses a framework's .tbd file and returns the set of ObjC
// class names the framework exports.
//
// Returns (nil, errNoTBD) when no .tbd file exists — caller should fall back
// silently to path-based filtering.
// Returns (nil, err) for other errors (corrupt file, unreadable) — caller
// should log a warning before falling back.
// Returns (classes, nil) on success; classes may be empty if the .tbd exports
// no ObjC classes (Swift-only framework).
//
// .tbd files are TAPI v4 YAML stubs used by the linker. The objc-classes key
// in each exports section lists the exact ObjC classes a dylib exports — this
// is ground-truth ownership data, more reliable than header-path inference.
//
// Private SPI classes (names starting with '_') are excluded because they have
// no public headers and cannot be bridged.
func loadTBDClasses(bundlePath, framework string) (map[string]bool, error) {
	p := tbdFilePath(bundlePath, framework)
	if p == "" {
		return nil, errNoTBD
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, errNoTBD
	}
	classes, err := parseTBDClasses(string(data))
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", p, err)
	}
	return classes, nil
}

// tbdFilePath locates the .tbd stub inside a framework bundle. It tries the
// top-level symlink first (canonical SDK layout where Framework.tbd →
// Versions/A/Framework.tbd), then the versioned path used in unresolved
// framework dumps where the symlinks are absent. Returns "" when neither path
// exists.
func tbdFilePath(bundlePath, framework string) string {
	name := framework
	if idx := strings.LastIndex(framework, "/"); idx >= 0 {
		name = framework[idx+1:]
	}
	primary := filepath.Join(bundlePath, name+".tbd")
	if _, err := os.Stat(primary); err == nil {
		return primary
	}
	versioned := filepath.Join(bundlePath, "Versions", "A", name+".tbd")
	if _, err := os.Stat(versioned); err == nil {
		return versioned
	}
	return ""
}

// parseTBDClasses extracts ObjC class names from TBD v3/v4 YAML content.
//
// TBD YAML uses YAML flow sequences for objc-classes, always indented inside
// an exports: block (never at the top level). This parser looks for any line
// whose trimmed form starts with "objc-classes:" — treating it as a YAML
// mapping key regardless of indentation — then collects the inline flow
// sequence, spanning multiple lines when the closing ']' has not yet appeared.
// This avoids confusion from "objc-classes:" appearing inside a YAML string
// value because values are always quoted and would not start the trimmed line.
//
// macOS framework TBDs only list macOS classes so no target filtering is
// needed. Private classes (leading '_') are excluded.
func parseTBDClasses(content string) (map[string]bool, error) {
	classes := make(map[string]bool)
	lines := strings.Split(content, "\n")

	for i := 0; i < len(lines); i++ {
		stripped := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(stripped, "objc-classes:") {
			continue
		}

		// Collect the flow-sequence fragment starting after the colon. When the
		// opening '[' and closing ']' are not both on this line, continue onto
		// subsequent lines until the bracket depth reaches zero.
		rest := strings.TrimSpace(strings.TrimPrefix(stripped, "objc-classes:"))
		fragment := rest
		depth := strings.Count(rest, "[") - strings.Count(rest, "]")
		for depth > 0 && i+1 < len(lines) {
			i++
			next := strings.TrimSpace(lines[i])
			fragment += " " + next
			depth += strings.Count(next, "[") - strings.Count(next, "]")
		}

		addTBDClasses(fragment, classes)
	}

	return classes, nil
}

// addTBDClasses parses a TBD flow-sequence fragment (e.g. "[ ClassA, ClassB ]"
// or "ClassA, ClassB") and adds the class names to dst. Private SPI names
// (starting with '_') are excluded.
func addTBDClasses(fragment string, dst map[string]bool) {
	fragment = strings.TrimPrefix(strings.TrimSpace(fragment), "[")
	fragment = strings.TrimSuffix(strings.TrimSpace(fragment), "]")
	fragment = strings.TrimSpace(fragment)
	if fragment == "" {
		return
	}
	for _, part := range strings.Split(fragment, ",") {
		name := strings.TrimSpace(part)
		name = strings.Trim(name, "'\"")
		name = strings.TrimSpace(name)
		if name != "" && !strings.HasPrefix(name, "_") {
			dst[name] = true
		}
	}
}
