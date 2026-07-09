//go:build darwin

package idiofw

import (
	"sort"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/meta"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/naming"
)

// cnames.go derives idiomatic Go spellings for C-style (snake_case) enum type
// names, turning Hv_gic_distributor_reg_t into GICDistributorReg. The rename
// table is computed once per framework (buildCEnumLocalNames) and consulted by
// every site that names an enum type — signature resolution, the enum
// definition pass, struct fields, and the taken-name reservation — so a
// signature can never reference a spelling the definition pass did not emit.
// ObjC-style enum names (no underscore) keep the existing class-prefix strip.

// buildCEnumLocalNames maps each snake_case enum's mechanical Go type name
// (naming.GoTypeName of the metadata key, e.g. Hv_exit_reason_t) — and the raw
// key itself — to its idiomatic local spelling (ExitReason). Renames are
// resolved deterministically over sorted keys; a candidate that is invalid,
// collides with another enum's name (candidate or mechanical), or collides
// with a class wrapper name falls back to the mechanical spelling, so the
// table is always total and collision-free.
func buildCEnumLocalNames(
	framework *meta.FrameworkMeta,
	classGoNames map[string]bool,
) map[string]string {
	var keys []string
	for key, enum := range framework.Enums {
		if enum.Availability.IsUnavailable || enum.IsAnon || !strings.Contains(key, "_") {
			continue
		}
		if !isExportedGoIdent(naming.GoTypeName(key)) {
			continue
		}
		keys = append(keys, key)
	}
	if len(keys) == 0 {
		return nil
	}
	sort.Strings(keys)

	// The C namespace prefix (hv_, vmnet_, …) is the dominant first segment
	// among the snake_case enums — dominant rather than unanimous, because SDK
	// headers leak the odd foreign typedef (idtype_t, mpo_flags_t) into a
	// framework's scan. An enum whose own first segment differs simply keeps it
	// (cEnumTypeName strips only a matching prefix). Ties break lexically so
	// the choice is deterministic.
	segmentCount := map[string]int{}
	for _, key := range keys {
		segmentCount[firstSnakeSegment(key)]++
	}
	namespacePrefix := ""
	for segment, count := range segmentCount {
		if count > segmentCount[namespacePrefix] ||
			(count == segmentCount[namespacePrefix] && segment < namespacePrefix) {
			namespacePrefix = segment
		}
	}
	if segmentCount[namespacePrefix] < 2 {
		namespacePrefix = "" // one-off segments are not a namespace
	}

	mechanical := make(map[string]bool, len(keys))
	for _, key := range keys {
		mechanical[naming.GoTypeName(key)] = true
	}

	out := make(map[string]string, len(keys)*2)
	used := make(map[string]bool, len(keys))
	for _, key := range keys {
		goType := naming.GoTypeName(key)
		candidate := cEnumTypeName(key, namespacePrefix)
		if candidate == "" || used[candidate] || mechanical[candidate] || classGoNames[candidate] {
			candidate = goType
		}
		used[candidate] = true
		out[goType] = candidate
		out[key] = candidate
	}
	return out
}

// cEnumTypeName derives the idiomatic Go type name for one snake_case C enum
// key: drop the shared namespace prefix segment and a trailing _t, PascalCase
// the rest, and re-case known initialisms (hv_gic_distributor_reg_t →
// GICDistributorReg). When dropping the prefix would leave nothing, or leave
// the unusable name "Error", the prefix is kept (hv_error → HvError). Returns
// "" when no exported name can be derived.
func cEnumTypeName(key, namespacePrefix string) string {
	segments := snakeSegments(key)
	if len(segments) > 0 && segments[len(segments)-1] == "t" {
		segments = segments[:len(segments)-1]
	}
	stripped := segments
	if namespacePrefix != "" && len(segments) > 1 && segments[0] == namespacePrefix {
		stripped = segments[1:]
	}
	name := applyInitialisms(pascalSegments(stripped))
	if name == "" || name == "Error" || !isExportedGoIdent(name) {
		name = applyInitialisms(pascalSegments(segments))
	}
	if !isExportedGoIdent(name) {
		return ""
	}
	return name
}

// cEnumMemberNames derives the const names for a C enum's members: the local
// type name plus each member's distinguishing suffix — its SCREAMING_SNAKE
// segments minus the longest leading segment run all members share
// (HV_EXIT_REASON_CANCELED → ExitReasonCanceled under type ExitReason). At
// least one segment is always kept per member. Returns nil when the members do
// not look like C constants, signalling the caller to keep the existing
// naming.
func cEnumMemberNames(localTypeName string, members []meta.EnumMember) map[string]string {
	var names [][]string
	for _, member := range members {
		if member.Availability.IsUnavailable {
			continue
		}
		if member.Name == "" || member.Name != strings.ToUpper(member.Name) ||
			!strings.Contains(member.Name, "_") {
			return nil // not SCREAMING_SNAKE C constants
		}
		names = append(names, snakeSegments(member.Name))
	}
	if len(names) == 0 {
		return nil
	}

	// Longest common leading segment run, capped so every member keeps at
	// least one segment of its own.
	common := len(names[0]) - 1
	for _, segments := range names[1:] {
		limit := len(segments) - 1
		if limit < common {
			common = limit
		}
		for i := 0; i < common; i++ {
			if !strings.EqualFold(segments[i], names[0][i]) {
				common = i
				break
			}
		}
	}
	if common < 0 {
		common = 0
	}

	out := make(map[string]string, len(names))
	used := make(map[string]bool, len(names))
	for _, member := range members {
		if member.Availability.IsUnavailable {
			continue
		}
		segments := snakeSegments(member.Name)
		suffix := applyInitialisms(pascalSegments(segments[common:]))
		constName := localTypeName + suffix
		if suffix == "" || used[constName] {
			constName = naming.GoTypeName(member.Name) // fallback: mechanical
		}
		used[constName] = true
		out[member.Name] = constName
	}
	return out
}

// snakeSegments splits a snake_case identifier into its lower-cased non-empty
// segments.
func snakeSegments(name string) []string {
	var segments []string
	for segment := range strings.SplitSeq(name, "_") {
		if segment == "" {
			continue
		}
		segments = append(segments, strings.ToLower(segment))
	}
	return segments
}

// pascalSegments joins lower-cased segments into a PascalCase identifier.
func pascalSegments(segments []string) string {
	var sb strings.Builder
	for _, segment := range segments {
		sb.WriteString(capitalizeFirst(segment))
	}
	return sb.String()
}

// firstSnakeSegment returns a snake_case identifier's first non-empty segment,
// lower-cased ("" when there is none).
func firstSnakeSegment(name string) string {
	segments := snakeSegments(name)
	if len(segments) == 0 {
		return ""
	}
	return segments[0]
}
