// Package validate runs structural integrity checks over a set of loaded
// .gometa.json metadata files. The metadata artifact is the product of the
// scan phase and the input to every emitter — defects caught here surface as
// codegen panics, compile errors in generated packages, or runtime failures
// when caught later. `generate validate` runs these checks in CI so that
// scanner regressions and partial scans fail at the artifact boundary.
package validate

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/meta"
)

// Severity classifies a finding. Errors fail `generate validate`; warnings
// are reported but do not affect the exit code.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// Finding is a single integrity problem detected in the metadata set.
type Finding struct {
	Severity  Severity
	Framework string
	Message   string
}

func (f Finding) String() string {
	return fmt.Sprintf("%s: %s: %s", f.Severity, f.Framework, f.Message)
}

// Frameworks runs every check over the loaded metadata set and returns the
// combined findings, sorted for deterministic output.
func Frameworks(frameworks []*meta.FrameworkMeta) []Finding {
	findings := make([]Finding, 0, len(frameworks))
	findings = append(findings, checkDanglingSuperclasses(frameworks)...)
	findings = append(findings, checkSuperchainCycles(frameworks)...)
	findings = append(findings, checkOwnershipTies(frameworks)...)
	findings = append(findings, checkEnumConflicts(frameworks)...)
	findings = append(findings, checkEmptyFrameworks(frameworks)...)
	findings = append(findings, checkAvailability(frameworks)...)
	findings = append(findings, checkSDKConsistency(frameworks)...)

	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Severity != findings[j].Severity {
			return findings[i].Severity == SeverityError
		}
		if findings[i].Framework != findings[j].Framework {
			return findings[i].Framework < findings[j].Framework
		}
		return findings[i].Message < findings[j].Message
	})
	return findings
}

// HasErrors reports whether any finding carries SeverityError.
func HasErrors(findings []Finding) bool {
	for _, f := range findings {
		if f.Severity == SeverityError {
			return true
		}
	}
	return false
}

// allClassNames builds the global set of class names across all frameworks.
func allClassNames(frameworks []*meta.FrameworkMeta) map[string]bool {
	names := make(map[string]bool)
	for _, framework := range frameworks {
		for name := range framework.Classes {
			names[name] = true
		}
	}
	return names
}

// checkDanglingSuperclasses reports classes whose Super resolves to no known
// class in the entire metadata set. The class emitter walks superclass chains
// to build struct embedding; a dangling Super silently truncates the chain.
// Severity is warning, not error: the usual cause is a superclass that Apple
// marks API_UNAVAILABLE(macos) (iOS-only base class), which the scanner
// intentionally omits — but a sudden spike indicates a scanner regression.
func checkDanglingSuperclasses(frameworks []*meta.FrameworkMeta) []Finding {
	known := allClassNames(frameworks)
	var findings []Finding
	for _, framework := range frameworks {
		for name, class := range framework.Classes {
			if class.Super != "" && !known[class.Super] {
				findings = append(findings, Finding{
					Severity:  SeverityWarning,
					Framework: framework.Framework,
					Message: fmt.Sprintf(
						"class %s has unknown superclass %s (macOS-unavailable base class, or scanner miss)",
						name,
						class.Super,
					),
				})
			}
		}
	}
	return findings
}

// checkSuperchainCycles reports circular superclass chains (A→B→A). Valid
// ObjC metadata never contains these, but a malformed file would send the
// emitter into an infinite loop.
func checkSuperchainCycles(frameworks []*meta.FrameworkMeta) []Finding {
	super := make(map[string]string)
	for _, framework := range frameworks {
		for name, class := range framework.Classes {
			if class.Super != "" {
				// First definition wins; conflicting supers are reported by
				// checkOwnershipTies, not here.
				if _, ok := super[name]; !ok {
					super[name] = class.Super
				}
			}
		}
	}

	var findings []Finding
	checked := make(map[string]bool)
	for name := range super {
		if checked[name] {
			continue
		}
		seen := map[string]bool{}
		for cur := name; cur != ""; cur = super[cur] {
			if seen[cur] {
				findings = append(findings, Finding{
					Severity:  SeverityError,
					Framework: "(global)",
					Message:   fmt.Sprintf("superclass cycle through %s", cur),
				})
				break
			}
			seen[cur] = true
			checked[cur] = true
		}
	}
	return findings
}

// checkOwnershipTies reports classes whose canonical owner is ambiguous: two
// or more frameworks define the class with the same minimal non-zero method
// count. The loader breaks such ties alphabetically, which is deterministic
// but arbitrary — a tie usually means header leakage the scanner should have
// filtered.
func checkOwnershipTies(frameworks []*meta.FrameworkMeta) []Finding {
	type entry struct {
		framework string
		score     int
	}
	entries := make(map[string][]entry)
	for _, framework := range frameworks {
		for name, class := range framework.Classes {
			score := len(class.Methods) + len(class.Properties)
			if score > 0 {
				entries[name] = append(entries[name], entry{framework.Framework, score})
			}
		}
	}

	var findings []Finding
	for name, list := range entries {
		if len(list) < 2 {
			continue
		}
		minScore := list[0].score
		for _, e := range list[1:] {
			if e.score < minScore {
				minScore = e.score
			}
		}
		// Collect distinct framework names at the minimal score. The same name
		// can legitimately appear twice when a framework ships both top-level
		// and nested inside an umbrella (the loader deduplicates those).
		tiedSet := map[string]bool{}
		var tied []string
		for _, e := range list {
			if e.score == minScore && !tiedSet[e.framework] {
				tiedSet[e.framework] = true
				tied = append(tied, e.framework)
			}
		}
		if len(tied) > 1 {
			sort.Strings(tied)
			findings = append(findings, Finding{
				Severity:  SeverityWarning,
				Framework: tied[0],
				Message: fmt.Sprintf(
					"class %s ownership tie between %s (score %d each) — loader picks %s alphabetically",
					name,
					strings.Join(tied, ", "),
					minScore,
					tied[0],
				),
			})
		}
	}
	return findings
}

// checkEnumConflicts reports enums that appear in multiple frameworks with
// different underlying Go types. Duplicated names are normal (Apple's headers
// re-export each other); a conflicting base type is not — first-write-wins in
// the loader means one framework's generated casts would be wrong.
func checkEnumConflicts(frameworks []*meta.FrameworkMeta) []Finding {
	type entry struct {
		framework string
		goType    string
	}
	entries := make(map[string][]entry)
	for _, framework := range frameworks {
		for name, enum := range framework.Enums {
			if enum.GoType != "" && len(enum.Members) > 0 {
				entries[name] = append(entries[name], entry{framework.Framework, enum.GoType})
			}
		}
	}

	var findings []Finding
	for name, list := range entries {
		types := map[string][]string{}
		for _, e := range list {
			types[e.goType] = append(types[e.goType], e.framework)
		}
		if len(types) < 2 {
			continue
		}
		desc := make([]string, 0, len(types))
		for goType, fws := range types {
			sort.Strings(fws)
			desc = append(desc, fmt.Sprintf("%s in %s", goType, strings.Join(fws, ",")))
		}
		sort.Strings(desc)
		findings = append(findings, Finding{
			Severity:  SeverityWarning,
			Framework: "(global)",
			Message: fmt.Sprintf(
				"enum %s has conflicting base types: %s",
				name,
				strings.Join(desc, "; "),
			),
		})
	}
	return findings
}

// checkEmptyFrameworks reports frameworks whose metadata contains no
// declarations at all. Swift-only frameworks and umbrellas are legitimately
// empty; anything else usually means the scan partially failed.
func checkEmptyFrameworks(frameworks []*meta.FrameworkMeta) []Finding {
	var findings []Finding
	for _, framework := range frameworks {
		if framework.IsSwiftOnly || len(framework.UmbrellaFor) > 0 {
			continue
		}
		total := len(framework.Classes) + len(framework.Protocols) + len(framework.Enums) +
			len(framework.Structs) + len(framework.Functions) + len(framework.Externs) +
			len(framework.Typedefs)
		if total == 0 {
			findings = append(findings, Finding{
				Severity:  SeverityWarning,
				Framework: framework.Framework,
				Message:   "no declarations extracted — scan may have failed partially",
			})
		}
	}
	return findings
}

// checkAvailability reports availability anomalies on classes: malformed
// version strings and deprecated-before-introduced ranges.
func checkAvailability(frameworks []*meta.FrameworkMeta) []Finding {
	var findings []Finding
	for _, framework := range frameworks {
		for name, class := range framework.Classes {
			findings = append(
				findings,
				checkAvailabilityVersions(
					framework.Framework,
					"class "+name,
					class.Availability,
				)...)
		}
		for name, enum := range framework.Enums {
			findings = append(
				findings,
				checkAvailabilityVersions(framework.Framework, "enum "+name, enum.Availability)...)
		}
		for _, function := range framework.Functions {
			findings = append(
				findings,
				checkAvailabilityVersions(
					framework.Framework,
					"function "+function.Name,
					function.Availability,
				)...)
		}
	}
	return findings
}

func checkAvailabilityVersions(frameworkName, decl string, avail meta.Availability) []Finding {
	var findings []Finding
	intro, introOK := parseVersion(avail.MacOSIntroduced)
	if avail.MacOSIntroduced != "" && !introOK && !isDeprecationSentinel(avail.MacOSIntroduced) {
		findings = append(findings, Finding{
			Severity:  SeverityWarning,
			Framework: frameworkName,
			Message: fmt.Sprintf(
				"%s has malformed introduced version %q",
				decl,
				avail.MacOSIntroduced,
			),
		})
	}
	depr, deprOK := parseVersion(avail.MacOSDeprecated)
	if avail.MacOSDeprecated != "" && !deprOK && !isDeprecationSentinel(avail.MacOSDeprecated) {
		findings = append(findings, Finding{
			Severity:  SeverityWarning,
			Framework: frameworkName,
			Message: fmt.Sprintf(
				"%s has malformed deprecated version %q",
				decl,
				avail.MacOSDeprecated,
			),
		})
	}
	if introOK && deprOK && depr < intro && !isDeprecationSentinel(avail.MacOSDeprecated) {
		findings = append(findings, Finding{
			Severity:  SeverityWarning,
			Framework: frameworkName,
			Message: fmt.Sprintf("%s deprecated (%s) before introduced (%s)",
				decl, avail.MacOSDeprecated, avail.MacOSIntroduced),
		})
	}
	return findings
}

// isDeprecationSentinel reports whether s is Apple's "deprecated at some
// unspecified future version" marker rather than a real version: the
// API_TO_BE_DEPRECATED macro name itself, or its expansion 100000.
func isDeprecationSentinel(s string) bool {
	return s == "API_TO_BE_DEPRECATED" || s == "100000"
}

// parseVersion converts "10.12.4" to a comparable number (10*1e6 + 12*1e3 + 4).
// Returns false for empty or non-numeric strings.
func parseVersion(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	parts := strings.Split(s, ".")
	if len(parts) > 3 {
		return 0, false
	}
	v := 0
	for i := range 3 {
		n := 0
		if i < len(parts) {
			parsed, err := strconv.Atoi(parts[i])
			if err != nil || parsed < 0 {
				return 0, false
			}
			n = parsed
		}
		v = v*1000 + n
	}
	return v, true
}

// checkSDKConsistency reports metadata files scanned against a different SDK
// version or architecture than the majority — the signature of a partial
// re-scan that left the committed tree mixed.
func checkSDKConsistency(frameworks []*meta.FrameworkMeta) []Finding {
	sdkCount := map[string]int{}
	archCount := map[string]int{}
	for _, framework := range frameworks {
		sdkCount[framework.SDKVersion]++
		archCount[framework.Arch]++
	}
	majoritySDK := majorityKey(sdkCount)
	majorityArch := majorityKey(archCount)

	var findings []Finding
	for _, framework := range frameworks {
		if framework.SDKVersion != majoritySDK {
			findings = append(findings, Finding{
				Severity:  SeverityWarning,
				Framework: framework.Framework,
				Message: fmt.Sprintf(
					"scanned against SDK %s while most metadata is SDK %s — partial re-scan?",
					framework.SDKVersion,
					majoritySDK,
				),
			})
		}
		if framework.Arch != majorityArch {
			findings = append(findings, Finding{
				Severity:  SeverityWarning,
				Framework: framework.Framework,
				Message: fmt.Sprintf(
					"scanned for arch %s while most metadata is %s",
					framework.Arch,
					majorityArch,
				),
			})
		}
	}
	return findings
}

func majorityKey(counts map[string]int) string {
	best, bestN := "", -1
	for k, n := range counts {
		if n > bestN || (n == bestN && k < best) {
			best, bestN = k, n
		}
	}
	return best
}
