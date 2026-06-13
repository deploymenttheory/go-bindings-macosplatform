//go:build darwin

package scanner

import (
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
)

// entitlementRe matches quoted com.apple.developer.* and com.apple.security.*
// entitlement keys in header comments. These are the actual provisioning
// entitlement keys that apps must include in their entitlements file.
// Other com.apple.* strings (notification names, UTIs, print settings) are
// not matched by this regex.
var entitlementRe = regexp.MustCompile(`"(com\.apple\.(?:developer|security|vm)\.[^"]+)"`)

// fileInfo holds the per-file parsed data extracted from a single header scan.
type fileInfo struct {
	docs         map[int]string            // 1-indexed line → doc comment text
	avails       map[int]macosplatformmetadata.Availability // 1-indexed line → availability annotation on that line
	entitlements map[int][]string          // 1-indexed line → entitlement keys found in preceding doc block
	optionalAt   map[int]bool              // 1-indexed line → true when inside an @optional protocol section
	mainThreadAt map[int]bool              // 1-indexed line → true when NS_SWIFT_UI_ACTOR appears on that line
}

// fileCache caches parsed file info keyed by absolute path.
var fileCache sync.Map // string → *fileInfo

// loadFile returns the cached fileInfo for absFile, parsing it if needed.
func loadFile(absFile string) *fileInfo {
	if absFile == "" {
		return &fileInfo{}
	}
	if v, ok := fileCache.Load(absFile); ok {
		return v.(*fileInfo)
	}
	fi, err := parseHeaderFile(absFile)
	if err != nil {
		fi = &fileInfo{
			docs:         map[int]string{},
			avails:       map[int]macosplatformmetadata.Availability{},
			entitlements: map[int][]string{},
			optionalAt:   map[int]bool{},
			mainThreadAt: map[int]bool{},
		}
	}
	fileCache.Store(absFile, fi)
	return fi
}

// docForNode returns the doc comment text for the declaration at (absFile, line), or "".
func docForNode(absFile string, line int) string {
	if absFile == "" || line == 0 {
		return ""
	}
	return loadFile(absFile).docs[line]
}

// entitlementsForNode returns entitlement keys found in the doc comment
// immediately preceding the declaration at (absFile, line).
func entitlementsForNode(absFile string, line int) []string {
	if absFile == "" || line == 0 {
		return nil
	}
	return loadFile(absFile).entitlements[line]
}

// isOptionalAt reports whether the declaration at (absFile, line) falls within
// an @optional section of an ObjC protocol body.
func isOptionalAt(absFile string, line int) bool {
	if absFile == "" || line == 0 {
		return false
	}
	return loadFile(absFile).optionalAt[line]
}

// isMainThreadAt reports whether the declaration at (absFile, lineNo) is annotated
// NS_SWIFT_UI_ACTOR. It checks the declaration line and the two preceding lines to
// handle annotations placed on a separate line before the declaration.
func isMainThreadAt(absFile string, lineNo int) bool {
	if absFile == "" || lineNo == 0 {
		return false
	}
	fi := loadFile(absFile)
	for _, offset := range []int{0, -1, -2} {
		if fi.mainThreadAt[lineNo+offset] {
			return true
		}
	}
	return false
}

// lineAvailability returns availability parsed from the raw source for the declaration
// at (absFile, lineNo). It searches a small window around lineNo to handle annotations
// placed on a preceding line (API_AVAILABLE prefix style) or trailing lines
// (multi-line declarations).
func lineAvailability(absFile string, lineNo int) macosplatformmetadata.Availability {
	if absFile == "" || lineNo == 0 {
		return macosplatformmetadata.Availability{}
	}
	fi := loadFile(absFile)
	// Check the declaration line and the lines preceding it.
	// Preceding lines first (annotation before the declaration), then same line.
	// We do NOT scan forward — forward scanning picks up NS_UNAVAILABLE on methods
	// inside the class body, which causes class declarations to be falsely marked
	// as unavailable (e.g. "- (instancetype)init NS_UNAVAILABLE;" at line N+2).
	for _, offset := range []int{0, -1, -2, -3, -4} {
		if av, ok := fi.avails[lineNo+offset]; ok {
			if av.MacOSIntroduced != "" || av.MacOSDeprecated != "" || av.IsUnavailable {
				return av
			}
		}
	}
	return macosplatformmetadata.Availability{}
}

// parseHeaderFile reads the SDK header at path and extracts:
//   - doc comment text immediately preceding each declaration
//   - availability annotations on each line
func parseHeaderFile(path string) (*fileInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	rawLines := strings.Split(string(data), "\n")
	n := len(rawLines)

	type lineKind int
	const (
		kindCode    lineKind = iota
		kindComment          // /// or /*! */ or /** */  — contributes to docs
		kindAnyComment       // plain /* */ — contributes to entitlement scan but not docs
		kindBlank
	)

	kinds := make([]lineKind, n)
	texts := make([]string, n) // comment text for kindComment/kindAnyComment lines

	inBlock := false
	isDocBlock := false
	for i, raw := range rawLines {
		line := strings.TrimSpace(raw)

		if inBlock {
			if before, _, found := strings.Cut(line, "*/"); found {
				inBlock = false
				before = strings.TrimPrefix(before, "*")
				texts[i] = strings.TrimSpace(before)
			} else {
				part := strings.TrimPrefix(line, "*")
				texts[i] = strings.TrimSpace(part)
			}
			if isDocBlock {
				kinds[i] = kindComment
			} else {
				kinds[i] = kindAnyComment
			}
			continue
		}

		if line == "" {
			kinds[i] = kindBlank
			continue
		}

		// Doxygen / HeaderDoc block comment: /*! or /**
		if strings.HasPrefix(line, "/*!") || strings.HasPrefix(line, "/**") {
			isDocBlock = true
			inBlock = true
			kinds[i] = kindComment
			inner := line[3:] // strip /*! or /**
			if before, _, found := strings.Cut(inner, "*/"); found {
				inBlock = false
				texts[i] = strings.TrimSpace(before)
			} else {
				texts[i] = strings.TrimSpace(inner)
			}
			continue
		}

		// Non-doc block comment: /* (without ! or second *)
		if strings.HasPrefix(line, "/*") {
			isDocBlock = false
			inBlock = true
			kinds[i] = kindAnyComment
			inner := strings.TrimPrefix(line[2:], "*")
			if before, _, found := strings.Cut(inner, "*/"); found {
				inBlock = false
				texts[i] = strings.TrimSpace(before)
			} else {
				texts[i] = strings.TrimSpace(inner)
			}
			continue
		}

		// Swift-style doc comment
		if strings.HasPrefix(line, "///") {
			kinds[i] = kindComment
			texts[i] = strings.TrimSpace(strings.TrimPrefix(line, "///"))
			continue
		}

		// Plain // comment — not a doc comment, treat as blank to break association
		if strings.HasPrefix(line, "//") {
			kinds[i] = kindBlank
			continue
		}

		kinds[i] = kindCode
	}

	docs := make(map[int]string)
	for i := range rawLines {
		if kinds[i] != kindCode {
			continue
		}
		// Look backward for a contiguous comment block (no blanks between it and this line)
		var parts []string
		j := i - 1
		for j >= 0 && kinds[j] == kindComment {
			if texts[j] != "" {
				parts = append(parts, texts[j])
			}
			j--
		}
		if len(parts) == 0 {
			continue
		}
		// Reverse — collected bottom-up
		for l, r := 0, len(parts)-1; l < r; l, r = l+1, r-1 {
			parts[l], parts[r] = parts[r], parts[l]
		}
		doc := strings.TrimSpace(strings.Join(parts, " "))
		if doc != "" {
			docs[i+1] = doc // 1-indexed
		}
	}

	// Parse availability annotations on each line.
	avails := make(map[int]macosplatformmetadata.Availability)
	for i, raw := range rawLines {
		av := parseAvailAnnotation(raw)
		if av.MacOSIntroduced != "" || av.MacOSDeprecated != "" || av.IsUnavailable {
			avails[i+1] = av // 1-indexed
		}
	}

	// Extract entitlement keys from comment blocks preceding each declaration.
	// SDK headers document required entitlements as quoted "com.apple.*" strings
	// in comments before the declaration. Apple headers often insert attribute
	// macros (VZ_EXPORT, API_AVAILABLE, NW_RETURNS_RETAINED etc.) between the
	// comment and the @interface/@class/function line — allow up to 4 such
	// interleaved code lines before requiring a comment block.
	entitlements := make(map[int][]string)
	for i := range rawLines {
		if kinds[i] != kindCode {
			continue
		}
		j := i - 1
		// Skip over attribute/annotation code lines to reach the comment block.
		skipped := 0
		for j >= 0 && kinds[j] == kindCode && skipped < 4 {
			j--
			skipped++
		}
		var keys []string
		seen := map[string]bool{}
		for j >= 0 && (kinds[j] == kindComment || kinds[j] == kindAnyComment) {
			for _, m := range entitlementRe.FindAllStringSubmatch(texts[j], -1) {
				k := m[1]
				if !seen[k] {
					seen[k] = true
					keys = append(keys, k)
				}
			}
			j--
		}
		if len(keys) > 0 {
			// Reverse to preserve top-to-bottom order (collected bottom-up).
			for l, r := 0, len(keys)-1; l < r; l, r = l+1, r-1 {
				keys[l], keys[r] = keys[r], keys[l]
			}
			entitlements[i+1] = keys // 1-indexed
		}
	}

	// Track @optional / @required section transitions for protocol method optionality.
	// ObjC protocol bodies alternate between @optional and @required sections;
	// all lines within an @optional section are marked. @end resets to required.
	optionalAt := make(map[int]bool)
	inOptional := false
	for i, raw := range rawLines {
		trimmed := strings.TrimSpace(raw)
		switch {
		case trimmed == "@optional" || strings.HasPrefix(trimmed, "@optional ") || strings.HasPrefix(trimmed, "@optional\t"):
			inOptional = true
		case trimmed == "@required" || strings.HasPrefix(trimmed, "@required ") || strings.HasPrefix(trimmed, "@required\t"),
			trimmed == "@end":
			inOptional = false
		default:
			if inOptional {
				optionalAt[i+1] = true // 1-indexed
			}
		}
	}

	// Track lines containing NS_SWIFT_UI_ACTOR for main-thread method annotation.
	mainThreadAt := make(map[int]bool)
	for i, raw := range rawLines {
		if strings.Contains(raw, "NS_SWIFT_UI_ACTOR") {
			mainThreadAt[i+1] = true // 1-indexed
		}
	}

	return &fileInfo{
		docs:         docs,
		avails:       avails,
		entitlements: entitlements,
		optionalAt:   optionalAt,
		mainThreadAt: mainThreadAt,
	}, nil
}

// parseAvailAnnotation extracts macOS availability from a single raw source line.
// It recognises the most common Apple annotation macros.
func parseAvailAnnotation(line string) macosplatformmetadata.Availability {
	var av macosplatformmetadata.Availability

	// Pre-scan ALL *_UNAVAILABLE(macos...) occurrences on the line. This covers
	// API_UNAVAILABLE, IC_UNAVAILABLE, CB_UNAVAILABLE, and any other framework-
	// specific wrapper that delegates to API_UNAVAILABLE(macos).
	// Multiple unavailability macros often appear on the same line, so we scan
	// all occurrences before taking any early return path.
	for s := line; ; {
		i := strings.Index(s, "_UNAVAILABLE(")
		if i < 0 {
			break
		}
		content := balancedContent(s[i+len("_UNAVAILABLE("):])
		if strings.Contains(content, "macos") {
			av.IsUnavailable = true
		}
		s = s[i+len("_UNAVAILABLE("):]
	}
	// __OSX_UNAVAILABLE is a standalone token (no parentheses) meaning "not
	// available on macOS". Used by ExternalAccessory and other frameworks.
	if strings.Contains(line, "__OSX_UNAVAILABLE") {
		av.IsUnavailable = true
	}
	// *_UNAVAILABLE_BEGIN marks a block of declarations as unavailable on a platform.
	// The expansion location of AvailabilityAttr nodes inside the block points here.
	// Matches API_UNAVAILABLE_BEGIN, MP_UNAVAILABLE_BEGIN, and other framework macros.
	for s := line; ; {
		i := strings.Index(s, "_UNAVAILABLE_BEGIN(")
		if i < 0 {
			break
		}
		content := balancedContent(s[i+len("_UNAVAILABLE_BEGIN("):])
		if strings.Contains(content, "macos") {
			av.IsUnavailable = true
		}
		s = s[i+len("_UNAVAILABLE_BEGIN("):]
	}
	// API_OBSOLETED_WITH_REPLACEMENT marks a symbol as fully removed on a platform.
	// Any macOS entry means the symbol is gone and must not be called.
	if i := strings.Index(line, "API_OBSOLETED_WITH_REPLACEMENT("); i >= 0 {
		content := balancedContent(line[i+len("API_OBSOLETED_WITH_REPLACEMENT("):])
		if strings.Contains(content, "macos(") {
			av.IsUnavailable = true
		}
	}

	// API_DEPRECATED_WITH_REPLACEMENT must be checked before API_DEPRECATED
	// because the latter is a prefix of the former.
	if i := strings.Index(line, "API_DEPRECATED_WITH_REPLACEMENT("); i >= 0 {
		content := balancedContent(line[i+len("API_DEPRECATED_WITH_REPLACEMENT("):])
		av.ReplacedBy = extractFirstQuotedArg(content)
		if intro, depr := extractMacOSVersionRange(content); intro != "" || depr != "" {
			av.MacOSIntroduced = intro
			av.MacOSDeprecated = depr
		}
		return av
	}

	if i := strings.Index(line, "API_DEPRECATED("); i >= 0 {
		content := balancedContent(line[i+len("API_DEPRECATED("):])
		av.DeprecationMsg = extractFirstQuotedArg(content)
		av.ReplacedBy = parseReplacedBy(av.DeprecationMsg)
		if intro, depr := extractMacOSVersionRange(content); intro != "" || depr != "" {
			av.MacOSIntroduced = intro
			av.MacOSDeprecated = depr
		}
		return av
	}

	if i := strings.Index(line, "API_AVAILABLE("); i >= 0 {
		content := balancedContent(line[i+len("API_AVAILABLE("):])
		if intro, _ := extractMacOSVersionRange(content); intro != "" {
			av.MacOSIntroduced = intro
			return av
		}
		// API_AVAILABLE without a macOS entry is ambiguous: it can describe a
		// macOS-native API that also got Catalyst/iOS coverage (e.g. AppKit's
		// NSToolbarItem with API_AVAILABLE(ios(13.0))). The framework membership
		// is the disambiguator; we do NOT mark the declaration unavailable based
		// solely on missing macOS entries here. Truly iOS-only APIs additionally
		// carry API_UNAVAILABLE(macos) — those are caught by the pre-scan above.
		// Fall through to check for unavailability markers below.
	}

	// Older NS_AVAILABLE_MAC / NS_DEPRECATED_MAC styles
	if i := strings.Index(line, "NS_DEPRECATED_MAC("); i >= 0 {
		content := balancedContent(line[i+len("NS_DEPRECATED_MAC("):])
		parts := splitTopLevelCommas(content)
		if len(parts) >= 1 {
			av.MacOSIntroduced = normaliseVersion(strings.TrimSpace(parts[0]))
		}
		if len(parts) >= 2 {
			av.MacOSDeprecated = normaliseVersion(strings.TrimSpace(parts[1]))
		}
		return av
	}

	// NS_DEPRECATED(macIntro, macDepr, iosIntro, iosDepr) — the four-argument
	// cross-platform form. macOS intro = NA means the API never existed on
	// macOS and must be skipped at emit time (e.g. EKCalendarItem.UUID's
	// `NS_DEPRECATED(NA, NA, 5_0, 6_0)` is iOS-only).
	if i := strings.Index(line, "NS_DEPRECATED("); i >= 0 {
		content := balancedContent(line[i+len("NS_DEPRECATED("):])
		parts := splitTopLevelCommas(content)
		if len(parts) >= 1 {
			macIntro := strings.TrimSpace(parts[0])
			if macIntro == "NA" {
				av.IsUnavailable = true
				return av
			}
			if macIntro != "" {
				av.MacOSIntroduced = normaliseVersion(macIntro)
			}
		}
		if len(parts) >= 2 {
			macDepr := strings.TrimSpace(parts[1])
			if macDepr != "" && macDepr != "NA" {
				av.MacOSDeprecated = normaliseVersion(macDepr)
			}
		}
		return av
	}

	if i := strings.Index(line, "NS_AVAILABLE_MAC("); i >= 0 {
		content := balancedContent(line[i+len("NS_AVAILABLE_MAC("):])
		av.MacOSIntroduced = normaliseVersion(strings.TrimSpace(content))
		return av
	}

	// NS_CLASS_AVAILABLE(macOS, iOS) and NS_AVAILABLE(macOS, iOS): first arg is the
	// macOS version; "NA" means not available on macOS. Must be checked after
	// NS_AVAILABLE_MAC( and before NS_UNAVAILABLE to avoid substring collisions.
	for _, marker := range []string{"NS_CLASS_AVAILABLE(", "NS_AVAILABLE("} {
		if idx := strings.Index(line, marker); idx >= 0 {
			content := balancedContent(line[idx+len(marker):])
			parts := splitTopLevelCommas(content)
			if len(parts) >= 1 {
				macVer := strings.TrimSpace(parts[0])
				if macVer == "NA" {
					av.IsUnavailable = true
					return av
				}
				if macVer != "" {
					av.MacOSIntroduced = normaliseVersion(macVer)
					return av
				}
			}
		}
	}

	// NS_UNAVAILABLE: __attribute__((unavailable)) — unconditionally not callable.
	// Skip method declaration lines (starting with - or +) to avoid a method's
	// NS_UNAVAILABLE falsely propagating to a nearby class/type declaration when
	// lineAvailability scans backward.
	if strings.Contains(line, "NS_UNAVAILABLE") {
		trimmed := strings.TrimSpace(line)
		// Skip method declarations (- or + prefix) and property declarations
		// (@property) to avoid a member's NS_UNAVAILABLE falsely propagating to
		// a nearby class/type declaration when lineAvailability scans backward.
		isMemberDecl := strings.HasPrefix(trimmed, "-") ||
			strings.HasPrefix(trimmed, "+") ||
			strings.HasPrefix(trimmed, "@property")
		if !isMemberDecl {
			av.IsUnavailable = true
			return av
		}
	}

	// NS_AVAILABLE_IOS / NS_DEPRECATED_IOS / NS_CLASS_AVAILABLE_IOS /
	// NS_CLASS_DEPRECATED_IOS mark APIs that exist only on iOS (and related
	// platforms), never on macOS. Treat all of these as macOS-unavailable.
	for _, marker := range []string{
		"NS_AVAILABLE_IOS(",
		"NS_DEPRECATED_IOS(",
		"NS_CLASS_AVAILABLE_IOS(",
		"NS_CLASS_DEPRECATED_IOS(",
	} {
		if strings.Contains(line, marker) {
			av.IsUnavailable = true
			return av
		}
	}

	// SCREEN_CAPTURE_OBSOLETE(x,y,z) expands to
	// __attribute__((availability(macos,introduced=x,deprecated=y,obsoleted=z,...)))
	// The obsoleted=z means the API is fully removed; treat as unavailable.
	if strings.Contains(line, "SCREEN_CAPTURE_OBSOLETE(") {
		av.IsUnavailable = true
		return av
	}

	// CB_CM_API_AVAILABLE is a CoreBluetooth macro that expands to
	// API_AVAILABLE(ios(13.0), tvos(13.0), watchos(6.0)) API_UNAVAILABLE(macos)
	if strings.Contains(line, "CB_CM_API_AVAILABLE") {
		av.IsUnavailable = true
		return av
	}

	// FILEPROVIDER_API_AVAILABILITY_V1 and _V2 expand to
	// API_AVAILABLE(ios(...)) API_UNAVAILABLE(macos, macCatalyst) ...
	// V1_V2_V3 and later are available on macOS, so only V1 and V2 are excluded.
	if strings.Contains(line, "FILEPROVIDER_API_AVAILABILITY_V1") ||
		strings.Contains(line, "FILEPROVIDER_API_AVAILABILITY_V2") {
		// Only mark unavailable if the line does NOT also contain a macos-available variant.
		if !strings.Contains(line, "FILEPROVIDER_API_AVAILABILITY_V1_V2") &&
			!strings.Contains(line, "FILEPROVIDER_API_AVAILABILITY_V2_V3") {
			av.IsUnavailable = true
			return av
		}
	}

	// MODELCOLLECTION_SUNSET(msg, macos(...), ...) expands to API_UNAVAILABLE(macos)
	// on macOS. The class and its members are sunset on macOS.
	if strings.Contains(line, "MODELCOLLECTION_SUNSET(") {
		av.IsUnavailable = true
		return av
	}

	// If unavailability was already detected by the pre-scan above, return now
	// (before reaching NS_SWIFT_UNAVAILABLE which would overwrite the state).
	if av.IsUnavailable {
		return av
	}

	// NS_SWIFT_UNAVAILABLE marks symbols that should not be called from Swift
	// (and by convention from other non-ObjC callers). Record as a deprecation note.
	if i := strings.Index(line, "NS_SWIFT_UNAVAILABLE("); i >= 0 {
		content := balancedContent(line[i+len("NS_SWIFT_UNAVAILABLE("):])
		av.DeprecationMsg = strings.Trim(content, `"`)
		return av
	}

	return av
}

// balancedContent returns the content from the start of s up to (but not
// including) the matching closing paren, respecting nested parens.
func balancedContent(s string) string {
	depth := 1
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return s[:i]
			}
		}
	}
	return s
}

// extractMacOSVersionRange returns the (introduced, deprecated) versions from
// a platform list like "macos(10.6), ios(4.0)" or "macosx(10.0, 10.4), ...".
// deprecated is empty when only one version is present (API_AVAILABLE case).
func extractMacOSVersionRange(content string) (introduced, deprecated string) {
	for _, prefix := range []string{"macos(", "macosx("} {
		idx := strings.Index(content, prefix)
		if idx < 0 {
			continue
		}
		inner := balancedContent(content[idx+len(prefix):])
		parts := splitTopLevelCommas(inner)
		if len(parts) >= 1 {
			introduced = strings.TrimSpace(parts[0])
		}
		if len(parts) >= 2 {
			deprecated = strings.TrimSpace(parts[1])
		}
		return
	}
	return
}

// extractFirstQuotedArg returns the content of the first double-quoted string
// argument in an availability macro content string.
func extractFirstQuotedArg(content string) string {
	start := strings.Index(content, `"`)
	if start < 0 {
		return ""
	}
	end := strings.Index(content[start+1:], `"`)
	if end < 0 {
		return ""
	}
	return content[start+1 : start+1+end]
}

// splitTopLevelCommas splits a string on commas that are not inside parentheses.
func splitTopLevelCommas(s string) []string {
	var parts []string
	depth := 0
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// normaliseVersion converts underscore-separated SDK versions like "MAC_10_4"
// or "10_4" to dot notation "10.4". Already dot-separated versions pass through.
func normaliseVersion(v string) string {
	// Strip common prefixes
	for _, prefix := range []string{"MAC_", "IPHONE_", "OS_X_"} {
		v = strings.TrimPrefix(v, prefix)
	}
	return strings.ReplaceAll(v, "_", ".")
}
