// Package metadiff compares two metadata trees and produces a semantic API
// change report. An SDK bump rewrites hundreds of megabytes of .gometa.json —
// unreviewable as a raw git diff. This report makes the bump auditable
// (what was added, removed, or changed) and doubles as a consumer-facing
// changelog. A mass disappearance of one construct kind usually means a
// scanner regression rather than an Apple change — exactly what a reviewer
// needs to see before merging.
package metadiff

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/meta"
)

// SignatureChange records one declaration whose signature changed between trees.
type SignatureChange struct {
	Name string `json:"name"` // "NSString.stringWithFormat:" or "CFArrayCreate"
	Old  string `json:"old"`
	New  string `json:"new"`
}

// FrameworkDiff is the change set for a single framework present in both trees.
type FrameworkDiff struct {
	Framework string `json:"framework"`

	ClassesAdded   []string `json:"classes_added,omitempty"`
	ClassesRemoved []string `json:"classes_removed,omitempty"`

	MethodsAdded   []string          `json:"methods_added,omitempty"`   // "Class.selector"
	MethodsRemoved []string          `json:"methods_removed,omitempty"` // "Class.selector"
	MethodChanges  []SignatureChange `json:"method_changes,omitempty"`

	ProtocolsAdded   []string `json:"protocols_added,omitempty"`
	ProtocolsRemoved []string `json:"protocols_removed,omitempty"`

	EnumsAdded         []string `json:"enums_added,omitempty"`
	EnumsRemoved       []string `json:"enums_removed,omitempty"`
	EnumMembersAdded   []string `json:"enum_members_added,omitempty"`   // "Enum.Member"
	EnumMembersRemoved []string `json:"enum_members_removed,omitempty"` // "Enum.Member"
	// EnumBaseTypeChanges lists enums whose underlying Go type changed —
	// an ABI-relevant change for every generated cast on that enum.
	EnumBaseTypeChanges []SignatureChange `json:"enum_base_type_changes,omitempty"`

	StructsAdded   []string `json:"structs_added,omitempty"`
	StructsRemoved []string `json:"structs_removed,omitempty"`

	FunctionsAdded   []string          `json:"functions_added,omitempty"`
	FunctionsRemoved []string          `json:"functions_removed,omitempty"`
	FunctionChanges  []SignatureChange `json:"function_changes,omitempty"`

	ExternsAdded   []string `json:"externs_added,omitempty"`
	ExternsRemoved []string `json:"externs_removed,omitempty"`

	// DeprecationChanges lists classes whose deprecated version changed
	// (typically "" → some version: newly deprecated API).
	DeprecationChanges []string `json:"deprecation_changes,omitempty"`
}

// IsEmpty reports whether the framework saw no changes.
func (d *FrameworkDiff) IsEmpty() bool {
	return len(d.ClassesAdded)+len(d.ClassesRemoved)+
		len(d.MethodsAdded)+len(d.MethodsRemoved)+len(d.MethodChanges)+
		len(d.ProtocolsAdded)+len(d.ProtocolsRemoved)+
		len(d.EnumsAdded)+len(d.EnumsRemoved)+
		len(d.EnumMembersAdded)+len(d.EnumMembersRemoved)+len(d.EnumBaseTypeChanges)+
		len(d.StructsAdded)+len(d.StructsRemoved)+
		len(d.FunctionsAdded)+len(d.FunctionsRemoved)+len(d.FunctionChanges)+
		len(d.ExternsAdded)+len(d.ExternsRemoved)+
		len(d.DeprecationChanges) == 0
}

// Report is the full comparison between two metadata trees.
type Report struct {
	OldSDK string `json:"old_sdk"`
	NewSDK string `json:"new_sdk"`

	FrameworksAdded   []string         `json:"frameworks_added,omitempty"`
	FrameworksRemoved []string         `json:"frameworks_removed,omitempty"`
	Changed           []*FrameworkDiff `json:"changed,omitempty"`
}

// IsEmpty reports whether the two trees are semantically identical.
func (r *Report) IsEmpty() bool {
	return len(r.FrameworksAdded)+len(r.FrameworksRemoved)+len(r.Changed) == 0
}

// Compare diffs two metadata sets, keyed by framework name. Same-name
// duplicates within a set (a framework shipped both top-level and inside an
// umbrella) collapse to the top-level entry, mirroring the loader.
func Compare(oldFrameworks, newFrameworks []*meta.FrameworkMeta) *Report {
	oldByName := dedupeByName(oldFrameworks)
	newByName := dedupeByName(newFrameworks)

	report := &Report{
		OldSDK: majoritySDK(oldFrameworks),
		NewSDK: majoritySDK(newFrameworks),
	}

	for name := range oldByName {
		if _, ok := newByName[name]; !ok {
			report.FrameworksRemoved = append(report.FrameworksRemoved, name)
		}
	}
	for name, newFramework := range newByName {
		oldFramework, ok := oldByName[name]
		if !ok {
			report.FrameworksAdded = append(report.FrameworksAdded, name)
			continue
		}
		diff := compareFramework(oldFramework, newFramework)
		if !diff.IsEmpty() {
			report.Changed = append(report.Changed, diff)
		}
	}

	sort.Strings(report.FrameworksAdded)
	sort.Strings(report.FrameworksRemoved)
	sort.Slice(report.Changed, func(i, j int) bool {
		return report.Changed[i].Framework < report.Changed[j].Framework
	})
	return report
}

func dedupeByName(frameworks []*meta.FrameworkMeta) map[string]*meta.FrameworkMeta {
	byName := make(map[string]*meta.FrameworkMeta, len(frameworks))
	for _, framework := range frameworks {
		prev, exists := byName[framework.Framework]
		if !exists || (prev.ParentFramework != "" && framework.ParentFramework == "") {
			byName[framework.Framework] = framework
		}
	}
	return byName
}

func majoritySDK(frameworks []*meta.FrameworkMeta) string {
	counts := map[string]int{}
	for _, framework := range frameworks {
		counts[framework.SDKVersion]++
	}
	best, bestN := "", -1
	for v, n := range counts {
		if n > bestN || (n == bestN && v < best) {
			best, bestN = v, n
		}
	}
	return best
}

func compareFramework(oldFramework, newFramework *meta.FrameworkMeta) *FrameworkDiff {
	diff := &FrameworkDiff{Framework: newFramework.Framework}

	diff.ClassesAdded, diff.ClassesRemoved = keyDiff(
		keys(oldFramework.Classes),
		keys(newFramework.Classes),
	)
	diff.ProtocolsAdded, diff.ProtocolsRemoved = keyDiff(
		keys(oldFramework.Protocols),
		keys(newFramework.Protocols),
	)
	diff.EnumsAdded, diff.EnumsRemoved = keyDiff(keys(oldFramework.Enums), keys(newFramework.Enums))
	diff.StructsAdded, diff.StructsRemoved = keyDiff(
		keys(oldFramework.Structs),
		keys(newFramework.Structs),
	)

	// Methods, per class present in both trees.
	for className, newClass := range newFramework.Classes {
		oldClass, ok := oldFramework.Classes[className]
		if !ok {
			continue
		}
		compareMethods(className, oldClass.Methods, newClass.Methods, diff)
		if oldClass.Availability.MacOSDeprecated != newClass.Availability.MacOSDeprecated {
			diff.DeprecationChanges = append(diff.DeprecationChanges,
				fmt.Sprintf(
					"%s: deprecated %s → %s",
					className,
					orNone(
						oldClass.Availability.MacOSDeprecated,
					),
					orNone(newClass.Availability.MacOSDeprecated),
				))
		}
	}

	// Enum members and base types, per enum present in both trees.
	for enumName, newEnum := range newFramework.Enums {
		oldEnum, ok := oldFramework.Enums[enumName]
		if !ok {
			continue
		}
		if oldEnum.GoType != newEnum.GoType {
			diff.EnumBaseTypeChanges = append(diff.EnumBaseTypeChanges, SignatureChange{
				Name: enumName, Old: oldEnum.GoType, New: newEnum.GoType,
			})
		}
		oldMembers := map[string]bool{}
		for _, member := range oldEnum.Members {
			oldMembers[member.Name] = true
		}
		newMembers := map[string]bool{}
		for _, member := range newEnum.Members {
			newMembers[member.Name] = true
			if !oldMembers[member.Name] {
				diff.EnumMembersAdded = append(diff.EnumMembersAdded, enumName+"."+member.Name)
			}
		}
		for _, member := range oldEnum.Members {
			if !newMembers[member.Name] {
				diff.EnumMembersRemoved = append(diff.EnumMembersRemoved, enumName+"."+member.Name)
			}
		}
	}

	compareFunctions(oldFramework.Functions, newFramework.Functions, diff)
	compareExterns(oldFramework.Externs, newFramework.Externs, diff)

	for _, s := range [][]string{
		diff.MethodsAdded, diff.MethodsRemoved, diff.EnumMembersAdded, diff.EnumMembersRemoved,
		diff.FunctionsAdded, diff.FunctionsRemoved, diff.ExternsAdded, diff.ExternsRemoved,
		diff.DeprecationChanges,
	} {
		sort.Strings(s)
	}
	sort.Slice(
		diff.MethodChanges,
		func(i, j int) bool { return diff.MethodChanges[i].Name < diff.MethodChanges[j].Name },
	)
	sort.Slice(
		diff.FunctionChanges,
		func(i, j int) bool { return diff.FunctionChanges[i].Name < diff.FunctionChanges[j].Name },
	)
	sort.Slice(
		diff.EnumBaseTypeChanges,
		func(i, j int) bool { return diff.EnumBaseTypeChanges[i].Name < diff.EnumBaseTypeChanges[j].Name },
	)
	return diff
}

// methodKey identifies a method within a class: class methods are prefixed
// "+", instance methods "-", matching ObjC convention.
func methodKey(method meta.Method) string {
	if method.IsClassMethod {
		return "+" + method.Selector
	}
	return "-" + method.Selector
}

// methodSignature renders a comparable signature string for change detection.
func methodSignature(method meta.Method) string {
	params := make([]string, len(method.Params))
	for i, param := range method.Params {
		params[i] = param.ObjCType
	}
	return "(" + strings.Join(params, ", ") + ") → " + method.Return.ObjCType
}

func compareMethods(className string, oldMethods, newMethods []meta.Method, diff *FrameworkDiff) {
	oldByKey := map[string]meta.Method{}
	for _, method := range oldMethods {
		oldByKey[methodKey(method)] = method
	}
	newKeys := map[string]bool{}
	for _, method := range newMethods {
		key := methodKey(method)
		newKeys[key] = true
		oldMethod, ok := oldByKey[key]
		if !ok {
			diff.MethodsAdded = append(diff.MethodsAdded, className+" "+key)
			continue
		}
		oldSig, newSig := methodSignature(oldMethod), methodSignature(method)
		if oldSig != newSig {
			diff.MethodChanges = append(diff.MethodChanges, SignatureChange{
				Name: className + " " + key, Old: oldSig, New: newSig,
			})
		}
	}
	for _, method := range oldMethods {
		if key := methodKey(method); !newKeys[key] {
			diff.MethodsRemoved = append(diff.MethodsRemoved, className+" "+key)
		}
	}
}

func functionSignature(function meta.Function) string {
	params := make([]string, len(function.Params))
	for i, param := range function.Params {
		params[i] = param.ObjCType
	}
	return "(" + strings.Join(params, ", ") + ") → " + function.Return.ObjCType
}

// compareFunctions compares per-name signature multisets. C functions can be
// overloaded (clang's overloadable attribute — e.g. vecLib's SparseCleanup has
// 34 variants), so a name maps to a set of signatures, not one.
func compareFunctions(oldFunctions, newFunctions []meta.Function, diff *FrameworkDiff) {
	oldSigs := map[string][]string{}
	for _, function := range oldFunctions {
		oldSigs[function.Name] = append(oldSigs[function.Name], functionSignature(function))
	}
	newSigs := map[string][]string{}
	for _, function := range newFunctions {
		newSigs[function.Name] = append(newSigs[function.Name], functionSignature(function))
	}

	for name, newList := range newSigs {
		oldList, ok := oldSigs[name]
		if !ok {
			diff.FunctionsAdded = append(diff.FunctionsAdded, name)
			continue
		}
		sort.Strings(oldList)
		sort.Strings(newList)
		if equalStringSlices(oldList, newList) {
			continue
		}
		diff.FunctionChanges = append(diff.FunctionChanges, SignatureChange{
			Name: name,
			Old:  strings.Join(oldList, " | "),
			New:  strings.Join(newList, " | "),
		})
	}
	for name := range oldSigs {
		if _, ok := newSigs[name]; !ok {
			diff.FunctionsRemoved = append(diff.FunctionsRemoved, name)
		}
	}
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func compareExterns(oldExterns, newExterns []meta.Extern, diff *FrameworkDiff) {
	oldNames := map[string]bool{}
	for _, extern := range oldExterns {
		oldNames[extern.Name] = true
	}
	newNames := map[string]bool{}
	for _, extern := range newExterns {
		newNames[extern.Name] = true
		if !oldNames[extern.Name] {
			diff.ExternsAdded = append(diff.ExternsAdded, extern.Name)
		}
	}
	for _, extern := range oldExterns {
		if !newNames[extern.Name] {
			diff.ExternsRemoved = append(diff.ExternsRemoved, extern.Name)
		}
	}
}

func keys[V any](m map[string]V) map[string]bool {
	out := make(map[string]bool, len(m))
	for k := range m {
		out[k] = true
	}
	return out
}

func keyDiff(oldSet, newSet map[string]bool) (added, removed []string) {
	for k := range newSet {
		if !oldSet[k] {
			added = append(added, k)
		}
	}
	for k := range oldSet {
		if !newSet[k] {
			removed = append(removed, k)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	return added, removed
}

func orNone(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
}

// WriteMarkdown renders the report as reviewable markdown.
func (r *Report) WriteMarkdown(w io.Writer) {
	fmt.Fprintf(w, "# Metadata diff: SDK %s → %s\n\n", orNone(r.OldSDK), orNone(r.NewSDK))

	if r.IsEmpty() {
		fmt.Fprintln(w, "No semantic changes.")
		return
	}

	// Summary counts.
	var classesAdded, classesRemoved, methodsAdded, methodsRemoved, sigChanges int
	for _, d := range r.Changed {
		classesAdded += len(d.ClassesAdded)
		classesRemoved += len(d.ClassesRemoved)
		methodsAdded += len(d.MethodsAdded)
		methodsRemoved += len(d.MethodsRemoved)
		sigChanges += len(d.MethodChanges) + len(d.FunctionChanges)
	}
	fmt.Fprintf(w, "## Summary\n\n")
	fmt.Fprintf(
		w,
		"- Frameworks: %d added, %d removed, %d changed\n",
		len(r.FrameworksAdded),
		len(r.FrameworksRemoved),
		len(r.Changed),
	)
	fmt.Fprintf(w, "- Classes: %d added, %d removed\n", classesAdded, classesRemoved)
	fmt.Fprintf(
		w,
		"- Methods: %d added, %d removed, %d signature change(s)\n\n",
		methodsAdded,
		methodsRemoved,
		sigChanges,
	)

	writeList(w, "Frameworks added", r.FrameworksAdded)
	writeList(w, "Frameworks removed", r.FrameworksRemoved)

	for _, d := range r.Changed {
		fmt.Fprintf(w, "## %s\n\n", d.Framework)
		writeList(w, "Classes added", d.ClassesAdded)
		writeList(w, "Classes removed", d.ClassesRemoved)
		writeList(w, "Methods added", d.MethodsAdded)
		writeList(w, "Methods removed", d.MethodsRemoved)
		writeChanges(w, "Method signature changes", d.MethodChanges)
		writeList(w, "Protocols added", d.ProtocolsAdded)
		writeList(w, "Protocols removed", d.ProtocolsRemoved)
		writeList(w, "Enums added", d.EnumsAdded)
		writeList(w, "Enums removed", d.EnumsRemoved)
		writeList(w, "Enum members added", d.EnumMembersAdded)
		writeList(w, "Enum members removed", d.EnumMembersRemoved)
		writeChanges(w, "Enum base type changes", d.EnumBaseTypeChanges)
		writeList(w, "Structs added", d.StructsAdded)
		writeList(w, "Structs removed", d.StructsRemoved)
		writeList(w, "Functions added", d.FunctionsAdded)
		writeList(w, "Functions removed", d.FunctionsRemoved)
		writeChanges(w, "Function signature changes", d.FunctionChanges)
		writeList(w, "Externs added", d.ExternsAdded)
		writeList(w, "Externs removed", d.ExternsRemoved)
		writeList(w, "Deprecation changes", d.DeprecationChanges)
	}
}

func writeList(w io.Writer, title string, items []string) {
	if len(items) == 0 {
		return
	}
	fmt.Fprintf(w, "### %s (%d)\n\n", title, len(items))
	for _, item := range items {
		fmt.Fprintf(w, "- `%s`\n", item)
	}
	fmt.Fprintln(w)
}

func writeChanges(w io.Writer, title string, changes []SignatureChange) {
	if len(changes) == 0 {
		return
	}
	fmt.Fprintf(w, "### %s (%d)\n\n", title, len(changes))
	for _, change := range changes {
		fmt.Fprintf(
			w,
			"- `%s`\n  - old: `%s`\n  - new: `%s`\n",
			change.Name,
			change.Old,
			change.New,
		)
	}
	fmt.Fprintln(w)
}
