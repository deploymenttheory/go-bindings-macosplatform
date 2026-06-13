//go:build darwin

package scanner

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ScanConfig holds per-framework scan customisations.
type ScanConfig struct {
	// ExtraIncludeDirs lists additional directories (relative to the SDK root,
	// slash-separated) whose headers count as belonging to this framework.
	ExtraIncludeDirs []string `json:"extra_include_dirs,omitempty"`
}

// defaultScanConfigs is the built-in per-framework scan configuration. The
// committed metadata/scanconfig.json (loaded via LoadScanConfigFile) is the
// authoritative, reviewable copy — this map only keeps the scanner functional
// when no config file is present.
//
// Foundation: NSObject and NSProxy are defined in usr/include/objc/ rather
// than Foundation.framework/Headers/, so those files are accepted when
// scanning Foundation to capture the ObjC root classes under the correct
// framework.
var defaultScanConfigs = map[string]ScanConfig{
	"Foundation": {ExtraIncludeDirs: []string{"usr/include/objc"}},
}

// scanConfigs is the active per-framework scan configuration registry.
var scanConfigs = defaultScanConfigs

// LoadScanConfigFile replaces the active per-framework scan configuration with
// the contents of a JSON config file (map of framework name → ScanConfig).
// Returns false without error when the file does not exist.
func LoadScanConfigFile(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("reading scan config %s: %w", path, err)
	}
	var configs map[string]ScanConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return false, fmt.Errorf("parsing scan config %s: %w", path, err)
	}
	scanConfigs = configs
	return true, nil
}

// frameworkFilter decides whether an AST node originates from a specific
// framework's own headers. For ObjC classes it uses a TBD-derived allowlist
// when available — the linker's ground-truth export table — eliminating false
// positives from classes that leak into the AST via header inclusion but are
// not exported by this framework's dylib. For all other node types (enums,
// structs, functions) it falls back to Clang's differential file-location
// tracking.
type frameworkFilter struct {
	headerDir      string          // e.g. /path/to/SDK/Foundation.framework/Headers
	extraDirs      []string        // additional accepted directories (Foundation only)
	currentFile    string          // last non-empty file path seen while walking nodes
	sdkPath        string          // root SDK path
	allowedClasses map[string]bool // ObjC class names from .tbd; nil → use path filter
}

func newFilter(sdkPath, framework string) frameworkFilter {
	f := frameworkFilter{
		headerDir: FrameworkHeaderDir(sdkPath, framework),
		sdkPath:   sdkPath,
	}
	// Per-framework scan config: extra directories whose headers belong to
	// this framework (e.g. Foundation owns usr/include/objc/ — see
	// defaultScanConfigs and metadata/scanconfig.json).
	for _, dir := range scanConfigs[framework].ExtraIncludeDirs {
		f.extraDirs = append(f.extraDirs, filepath.Join(sdkPath, filepath.FromSlash(dir)))
	}
	// Load the TBD-derived class allowlist.
	//   - errNoTBD: .tbd file absent → silent path-based fallback (normal for many frameworks).
	//   - other error: file exists but is corrupt → warn so the user knows accuracy may be lower.
	bundlePath := FrameworkBundlePath(sdkPath, framework)
	classes, err := loadTBDClasses(bundlePath, framework)
	if err != nil && !errors.Is(err, errNoTBD) {
		fmt.Fprintf(
			os.Stderr,
			"warning: %s TBD parse error (%v) — falling back to header-path ownership\n",
			framework,
			err,
		)
	}
	f.allowedClasses = classes
	return f
}

// UpdateFile updates the rolling current-file tracker from a node's location.
// Call this for every top-level node before Accept or AcceptClass.
func (f *frameworkFilter) UpdateFile(loc *Location) {
	if loc == nil {
		return
	}
	if file := loc.ResolvedFile(); file != "" {
		f.currentFile = file
	}
}

// Accept returns true if currentFile belongs to this framework's headers.
// Used for enums, structs, functions, externs, and protocols.
func (f *frameworkFilter) Accept() bool {
	if f.currentFile == "" {
		return false
	}
	if strings.HasPrefix(f.currentFile, f.headerDir) {
		return true
	}
	for _, dir := range f.extraDirs {
		if strings.HasPrefix(f.currentFile, dir) {
			return true
		}
	}
	return false
}

// AcceptClass returns true if the named ObjC class belongs to this framework.
// The TBD-derived allowlist is treated as additive, not exclusive: a class is
// accepted if it is listed in this framework's .tbd OR if its declaration file
// lies under this framework's own header directory. The path-filter fallback
// recovers toll-free-bridged foundational classes (NSArray, NSDictionary,
// NSObject, …) whose @interface lives in Foundation/NS*.h but whose ObjC
// runtime symbols are exported by re-exported libraries (CoreFoundation,
// libobjc). Path acceptance remains safe because declarations from re-exported
// headers live under a different framework's header directory.
func (f *frameworkFilter) AcceptClass(name string) bool {
	if f.allowedClasses != nil && f.allowedClasses[name] {
		return true
	}
	return f.Accept()
}
