//go:build darwin

package scanner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// errMissingLinkLib is returned when a C library registry entry has no link_lib field.
var errMissingLinkLib = errors.New("missing link_lib")

// SDKPath returns the path to the active macOS SDK via xcrun.
func SDKPath() (string, error) {
	out, err := exec.CommandContext(context.Background(), "xcrun", "--show-sdk-path").Output()
	if err != nil {
		return "", fmt.Errorf("xcrun --show-sdk-path: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// SDKVersion returns the macOS SDK version string (e.g. "26.5").
func SDKVersion() (string, error) {
	out, err := exec.CommandContext(context.Background(), "xcrun", "--show-sdk-version").Output()
	if err != nil {
		return "", fmt.Errorf("xcrun --show-sdk-version: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// ClangVersion returns the version line of the clang binary that DumpAST
// invokes (e.g. "Apple clang version 21.0.0 (clang-2100.3.9.2)"). Recorded
// in scanned metadata because clang releases have changed AST output in ways
// that affect extraction (Clang 21 stopped emitting availability versions).
func ClangVersion() (string, error) {
	out, err := exec.CommandContext(context.Background(), "xcrun", "clang", "--version").Output()
	if err != nil {
		return "", fmt.Errorf("xcrun clang --version: %w", err)
	}
	line, _, _ := strings.Cut(string(out), "\n")
	return strings.TrimSpace(line), nil
}

// XcodeVersion returns the active Xcode version and build, joined on one line
// (e.g. "Xcode 26.0 Build version 17A321"). Recorded in scanned metadata as
// toolchain provenance.
func XcodeVersion() (string, error) {
	out, err := exec.CommandContext(context.Background(), "xcodebuild", "-version").Output()
	if err != nil {
		return "", fmt.Errorf("xcodebuild -version: %w", err)
	}
	var parts []string
	for line := range strings.SplitSeq(string(out), "\n") {
		if l := strings.TrimSpace(line); l != "" {
			parts = append(parts, l)
		}
	}
	return strings.Join(parts, " "), nil
}

// ListFrameworks returns the sorted names of all frameworks available in sdkPath
// that have a valid umbrella header (Framework.framework/Headers/Framework.h).
// If sdkPath is empty it is auto-detected via xcrun.
// Sub-frameworks nested inside umbrella frameworks (e.g. Carbon/HIToolbox) are
// included using "Parent/Child" notation so the scanner can locate them.
func ListFrameworks(sdkPath string) ([]string, error) {
	if sdkPath == "" {
		p, err := SDKPath()
		if err != nil {
			return nil, err
		}
		sdkPath = p
	}
	pattern := filepath.Join(sdkPath, "System", "Library", "Frameworks", "*.framework")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob frameworks: %w", err)
	}
	var names []string
	for _, fw := range matches {
		name := strings.TrimSuffix(filepath.Base(fw), ".framework")
		header := filepath.Join(fw, "Headers", name+".h")
		if _, err := os.Stat(header); err != nil {
			continue
		}
		names = append(names, name)
		// Discover sub-frameworks within this framework's Frameworks/ directory.
		names = append(names, listSubFrameworks(fw, name)...)
	}
	sort.Strings(names)
	return names, nil
}

// listSubFrameworks returns "Parent/Child" names for sub-frameworks found in
// parentBundle/Frameworks/*.framework, each with a matching umbrella header.
func listSubFrameworks(parentBundle, parentName string) []string {
	pattern := filepath.Join(parentBundle, "Frameworks", "*.framework")
	matches, _ := filepath.Glob(pattern)
	var results []string
	for _, fw := range matches {
		name := strings.TrimSuffix(filepath.Base(fw), ".framework")
		header := filepath.Join(fw, "Headers", name+".h")
		if _, err := os.Stat(header); err != nil {
			continue
		}
		results = append(results, parentName+"/"+name)
	}
	return results
}

// IsSubFramework reports whether name uses "Parent/Child" sub-framework notation.
func IsSubFramework(name string) bool {
	return strings.Contains(name, "/")
}

// SubFrameworkParts splits a "Parent/Child" name into parent and child.
func SubFrameworkParts(name string) (parent, child string) {
	parts := strings.SplitN(name, "/", 2)
	return parts[0], parts[1]
}

// FrameworkBundlePath returns the path to a framework's .framework bundle.
// It handles both top-level frameworks and "Parent/Child" sub-frameworks.
func FrameworkBundlePath(sdkPath, name string) string {
	base := filepath.Join(sdkPath, "System", "Library", "Frameworks")
	if IsSubFramework(name) {
		parent, child := SubFrameworkParts(name)
		return filepath.Join(base, parent+".framework", "Frameworks", child+".framework")
	}
	return filepath.Join(base, name+".framework")
}

// IsSwiftOnly reports whether the framework at bundlePath has no ObjC surface
// and is implemented entirely in Swift (indicated by a .swiftmodule directory).
func IsSwiftOnly(bundlePath string) bool {
	modulesDir := filepath.Join(bundlePath, "Modules")
	entries, err := os.ReadDir(modulesDir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() && strings.HasSuffix(e.Name(), ".swiftmodule") {
			return true
		}
	}
	return false
}

// DetectSubFrameworkNames returns the names of sub-frameworks inside bundlePath/Frameworks/
// that have a valid umbrella header. Returns nil when the framework is not an umbrella.
func DetectSubFrameworkNames(bundlePath string) []string {
	pattern := filepath.Join(bundlePath, "Frameworks", "*.framework")
	matches, _ := filepath.Glob(pattern)
	var names []string
	for _, fw := range matches {
		name := strings.TrimSuffix(filepath.Base(fw), ".framework")
		header := filepath.Join(fw, "Headers", name+".h")
		if _, err := os.Stat(header); err != nil {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// FrameworkHeader returns the path to the umbrella header for a framework.
// framework may be "Name" (top-level) or "Parent/Child" (sub-framework).
// Falls back to the C library header path for known Apple C libraries.
func FrameworkHeader(sdkPath, framework string) string {
	if IsCLibrary(sdkPath, framework) {
		return CLibraryHeader(sdkPath, framework)
	}
	name := framework
	if IsSubFramework(framework) {
		_, name = SubFrameworkParts(framework)
	}
	bundle := FrameworkBundlePath(sdkPath, framework)
	return filepath.Join(bundle, "Headers", name+".h")
}

// FrameworkHeaderDir returns the directory containing all headers for a framework.
// framework may be "Name" (top-level) or "Parent/Child" (sub-framework).
// Falls back to the C library header directory for known Apple C libraries.
func FrameworkHeaderDir(sdkPath, framework string) string {
	if IsCLibrary(sdkPath, framework) {
		return CLibraryHeaderDir(sdkPath, framework)
	}
	bundle := FrameworkBundlePath(sdkPath, framework)
	return filepath.Join(bundle, "Headers")
}

// CLibraryDef describes a known Apple C library under {SDK}/usr/include/.
type CLibraryDef struct {
	// LinkLib is the dylib name passed to -l (e.g. "EndpointSecurity" → -lEndpointSecurity).
	LinkLib string `json:"link_lib"`
	// Header is the umbrella header path relative to {SDK}/usr/include/.
	// Empty means use the default {Name}/{Name}.h convention.
	Header string `json:"header,omitempty"`
	// HeaderDir is the filter path relative to {SDK}/usr/include/ used to decide
	// whether an AST node belongs to this library. Empty means derive from Header.
	// Use a directory path (e.g. "bsm/") to accept all headers under that directory,
	// or a file path (e.g. "sandbox.h") to match only that exact file.
	HeaderDir string `json:"header_dir,omitempty"`
}

// defaultCLibraries is the built-in registry of Apple C libraries. These live
// under {SDK}/usr/include/ rather than System/Library/Frameworks/ and must be
// linked with -l rather than -framework. The committed metadata/clibraries.json
// (loaded via LoadCLibrariesFile) is the authoritative, reviewable copy — this
// map only keeps the scanner functional when no config file is present.
var defaultCLibraries = map[string]CLibraryDef{
	"EndpointSecurity": {LinkLib: "EndpointSecurity"},
	// default header: EndpointSecurity/EndpointSecurity.h, filter: EndpointSecurity/

	"Sandbox": {LinkLib: "sandbox", Header: "sandbox.h", HeaderDir: "sandbox.h"},
	// single-file header in usr/include root; filter to exact file

	"bsm": {LinkLib: "bsm", Header: "bsm/libbsm.h", HeaderDir: "bsm/"},
	// libbsm.h is the umbrella; other bsm/*.h headers are also included by it

	"Compression": {LinkLib: "compression", Header: "compression.h", HeaderDir: "compression.h"},
	// single-file header in usr/include root; filter to exact file

	"AppleArchive": {LinkLib: "AppleArchive"},
	// default header: AppleArchive/AppleArchive.h, filter: AppleArchive/

	"xar": {LinkLib: "xar"},
	// default header: xar/xar.h, filter: xar/

	"libproc": {LinkLib: "proc", Header: "libproc.h", HeaderDir: "libproc.h"},
	// single-file header; links as -lproc (dylib is libproc.tbd)

	"oslog": {LinkLib: "System", Header: "os/log.h", HeaderDir: "os/log.h"},
	// os_log functions are in libSystem (always linked); -lSystem is redundant but valid

	"dispatch": {LinkLib: "System", Header: "dispatch/dispatch.h", HeaderDir: "dispatch/"},
	// GCD; dispatch functions live in libSystem (libdispatch is re-exported)

	"xpc": {LinkLib: "System", Header: "xpc/xpc.h", HeaderDir: "xpc/"},
	// XPC objects/dictionaries/connections; also in libSystem (libxpc re-exported)
}

// knownCLibraries is the active registry consulted by all C library helpers.
// Starts as the built-in defaults; LoadCLibrariesFile replaces it.
var knownCLibraries = defaultCLibraries

// LoadCLibrariesFile replaces the active C library registry with the contents
// of a JSON config file (map of library name → CLibraryDef). Adding a new
// Apple C library is then a data change plus re-scan, not a Go change.
// Returns false without error when the file does not exist.
func LoadCLibrariesFile(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("reading C library config %s: %w", path, err)
	}
	var libraries map[string]CLibraryDef
	if err := json.Unmarshal(data, &libraries); err != nil {
		return false, fmt.Errorf("parsing C library config %s: %w", path, err)
	}
	for name, def := range libraries {
		if def.LinkLib == "" {
			return false, fmt.Errorf(
				"parsing C library config %s: entry %q: %w",
				path,
				name,
				errMissingLinkLib,
			)
		}
	}
	knownCLibraries = libraries
	return true, nil
}

// IsCLibraryName reports whether name is registered as a known Apple C library,
// without performing any filesystem check. Use this when an SDK path is not
// available (e.g. when constructing metadata output paths during a scan).
func IsCLibraryName(name string) bool {
	_, ok := knownCLibraries[name]
	return ok
}

// IsCLibrary reports whether name is a known Apple C library whose umbrella
// header exists on the filesystem.
func IsCLibrary(sdkPath, name string) bool {
	if !IsCLibraryName(name) {
		return false
	}
	_, err := os.Stat(CLibraryHeader(sdkPath, name))
	return err == nil
}

// CLibraryHeader returns the umbrella header path for a known Apple C library.
// If the library definition specifies a custom Header path, that is used;
// otherwise the default {Name}/{Name}.h convention applies.
func CLibraryHeader(sdkPath, name string) string {
	if def := knownCLibraries[name]; def.Header != "" {
		return filepath.Join(sdkPath, "usr", "include", filepath.FromSlash(def.Header))
	}
	return filepath.Join(sdkPath, "usr", "include", name, name+".h")
}

// CLibraryHeaderRelative returns the umbrella header include path for a known
// Apple C library, relative to {SDK}/usr/include/ and slash-separated — the
// form used in a generated "#include <…>" directive (e.g. "compression.h",
// "os/log.h", "bsm/libbsm.h"). Defaults to the {Name}/{Name}.h convention
// when the library definition has no custom Header.
func CLibraryHeaderRelative(name string) string {
	if def := knownCLibraries[name]; def.Header != "" {
		return def.Header
	}
	return name + "/" + name + ".h"
}

// CLibraryHeaderDir returns the filter path for a known Apple C library.
// This is the path prefix used to decide whether an AST node belongs to the
// library. It may be a directory (e.g. "bsm/") or an exact file (e.g.
// "sandbox.h") depending on how the library's headers are laid out.
func CLibraryHeaderDir(sdkPath, name string) string {
	def := knownCLibraries[name]
	switch {
	case def.HeaderDir != "":
		return filepath.Join(sdkPath, "usr", "include", filepath.FromSlash(def.HeaderDir))
	case def.Header != "":
		// Derive the filter dir from the custom header's parent directory.
		return filepath.Dir(
			filepath.Join(sdkPath, "usr", "include", filepath.FromSlash(def.Header)),
		)
	default:
		return filepath.Join(sdkPath, "usr", "include", name)
	}
}

// ListCLibraries returns the sorted names of known Apple C libraries present
// in sdkPath. If sdkPath is empty it is auto-detected via xcrun.
func ListCLibraries(sdkPath string) ([]string, error) {
	if sdkPath == "" {
		p, err := SDKPath()
		if err != nil {
			return nil, err
		}
		sdkPath = p
	}
	var names []string
	for name := range knownCLibraries {
		if IsCLibrary(sdkPath, name) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names, nil
}

// DumpAST invokes xcrun clang -ast-dump=json on the framework's umbrella header
// and returns the parsed top-level ASTNode. The arch parameter should be
// "arm64" or "x86_64".
func DumpAST(sdkPath, framework, arch string) (*ASTNode, error) {
	header := FrameworkHeader(sdkPath, framework)

	args := []string{
		"clang",
		"-x", "objective-c",
		"-target", arch + "-apple-macos",
		"-isysroot", sdkPath,
		"-Xclang", "-ast-dump=json",
		"-fsyntax-only",
		"-Wno-everything", // suppress all warnings; we only want the AST
		"-fno-color-diagnostics",
		header,
	}

	cmd := exec.CommandContext(context.Background(), "xcrun", args...)

	// The AST JSON goes to stdout; stderr has diagnostics we ignore for now.
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// clang exits non-zero even on success when there are warnings.
		// Only treat it as a real error if stdout is empty.
		if stdout.Len() == 0 {
			return nil, fmt.Errorf("clang ast-dump failed for %s: %w\nstderr: %s",
				framework, err, stderr.String())
		}
	}

	var root ASTNode
	if err := json.Unmarshal(stdout.Bytes(), &root); err != nil {
		return nil, fmt.Errorf("parsing AST JSON for %s: %w", framework, err)
	}
	return &root, nil
}
