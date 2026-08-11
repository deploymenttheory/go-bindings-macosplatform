// Package parser parses Swift .swiftinterface module definition files and
// extracts type declarations (enums, error structs, value structs) into
// internal/swift/meta.FrameworkMeta.
package parser

import (
	"bufio"
	"regexp"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/swift/meta"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/swift/typemap"
)

// regexes for matching Swift declaration lines
var (
	reEnum   = regexp.MustCompile(`(?:^|\s)(?:@\S+\s+)*(?:public|open)(?:\s+\w+)*\s+enum\s+(\w+)\s*(?::\s*([^{]+?))?\s*\{`)
	reStruct = regexp.MustCompile(`(?:^|\s)(?:@\S+\s+)*(?:public|open)(?:\s+\w+)*\s+struct\s+(\w+)\s*(?::\s*([^{]+?))?\s*\{`)

	// reClassDecl matches a class or actor declaration opening brace.
	// Captures: 1=name, 2=conformance list (may be empty)
	reClassDecl = regexp.MustCompile(`(?:^|[\s@])(?:final\s+|open\s+)?(?:public|open)\s+(?:final\s+|open\s+)?(?:class|actor)\s+(\w+)\s*(?::\s*([^{]+?))?\s*\{`)

	// reClassKind extracts modifiers to determine isFinal/isOpen/isActor.
	reClassKind = regexp.MustCompile(`\b(final|open|actor)\b`)

	// extension SomeModule.SomeType [: Conformance] { — captures the qualified type name
	reExtension = regexp.MustCompile(`(?:^|\s)extension\s+([\w.]+)`)

	// `case name` or `case name = value` — not `case name(assocValue)`
	reCase = regexp.MustCompile(`^case\s+(\w+)(?:\s*=\s*(.+?))?$`)

	// stored property: `public [static] [final] var|let name: Type`
	// excludes computed properties (trailing `{ get }` stripped by isComputedProp check)
	reField = regexp.MustCompile(`^(?:public|open)(?:\s+static)?(?:\s+final)?(?:\s+lazy)?\s+(?:var|let)\s+(\w+)\s*:\s*(.+)$`)

	// static-let sentinel: `public static let name: Type`
	reStaticLet = regexp.MustCompile(`^(?:public|open)\s+static\s+(?:final\s+)?let\s+(\w+)\s*:`)

	// @available(...) annotation
	reAvailable = regexp.MustCompile(`^@available\(([^)]+)\)`)

	// reFuncDecl matches method declarations within a class.
	// Captures: 1=name, 2=raw params, 3="async" or "", 4="throws" or "", 5=return type or ""
	reFuncDecl = regexp.MustCompile(`\bfunc\s+(\w+)\s*(?:<[^>]*>)?\s*\(([^)]*)\)\s*(async\s*)?(throws\s*)?(?:->\s*(.+))?$`)

	// reInitDecl matches init declarations.
	// Captures: 1=raw params
	reInitDecl = regexp.MustCompile(`\binit[?!]?\s*\(([^)]*)\)`)

	// rePropMember matches property declarations within a class body.
	// Captures: 1=name, 2=type+optional accessor spec
	rePropMember = regexp.MustCompile(`\b(?:var|let)\s+(\w+)\s*:\s*(.+)$`)

	// reStaticPrefix detects "static" keyword before func/var/let on a line.
	reStaticPrefix = regexp.MustCompile(`\bstatic\b`)
)

// frame tracks one level of nested Swift type body.
type frame struct {
	name  string // flattened Go name, e.g. "TranslationSessionRequest"
	kind  string // "enum", "errorStruct", "struct", "class", "skip"
	depth int    // brace depth at which this frame's body opens
	enum  *meta.Enum
	errS  *meta.ErrorStruct
	strct *meta.Struct
	cls   *meta.Class
}

// ParseContent parses the text of a .swiftinterface file and returns
// framework metadata for the named framework.
// This function does not touch the filesystem and has no build constraints.
func ParseContent(frameworkName, content string) (*meta.FrameworkMeta, error) {
	framework := &meta.FrameworkMeta{Framework: frameworkName}

	// Extract module aliases from the header.
	// "-module-alias Module___Translation=Translation" means strip "Module___Translation." prefix.
	aliases := extractModuleAliases(content, frameworkName)

	var stack []frame
	depth := 0
	var pendingAvail *meta.Availability

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		rawLine := scanner.Text()
		line := strings.TrimSpace(rawLine)

		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		// Accumulate @available attributes.
		if m := reAvailable.FindStringSubmatch(line); m != nil {
			av := parseAvailability(m[1])
			if pendingAvail == nil {
				pendingAvail = &meta.Availability{}
			}
			mergeAvailability(pendingAvail, av)
			continue
		}

		// Lines starting with "@" may be pure attribute lines or inline-annotated declarations.
		// Pure attribute lines (no func/var/let/init in remainder) are skipped.
		// Inline-annotated declarations (e.g. "@_Concurrency.MainActor public func ...") are
		// processed below after stripping the leading attributes.
		// Keep attrLine as the pre-strip version so isMainActor can be detected downstream.
		attrLine := ""
		if strings.HasPrefix(line, "@") {
			if !lineHasDeclaration(line) {
				continue
			}
			// Preserve the original annotated line for isMainActor detection.
			attrLine = line
			line = stripLeadingAttributes(line)
		}

		// Count net brace change on this line.
		netBraces := strings.Count(line, "{") - strings.Count(line, "}")

		// Closing brace(s): pop frames that are now out of scope.
		if netBraces < 0 {
			depth += netBraces
			for len(stack) > 0 && stack[len(stack)-1].depth > depth {
				f := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				finalizeFrame(f, framework)
			}
			pendingAvail = nil
			continue
		}

		// Opening brace: new type scope.
		if netBraces > 0 {
			av := availOrNil(pendingAvail)
			pendingAvail = nil
			depth += netBraces

			if m := reEnum.FindStringSubmatch(line); m != nil {
				name := m[1]
				conformanceStr := strings.TrimSpace(m[2])
				qualName := qualPrefix(stack) + name
				rawGoType := ""
				if conformanceStr != "" {
					parts := splitConformances(conformanceStr)
					if len(parts) > 0 {
						rawGoType = typemap.RawEnumGoType(parts[0])
					}
				}
				e := &meta.Enum{Name: qualName, RawGoType: rawGoType, Availability: av}
				stack = append(stack, frame{name: qualName, kind: "enum", depth: depth, enum: e})
				continue
			}

			if m := reStruct.FindStringSubmatch(line); m != nil {
				name := m[1]
				conformanceStr := strings.TrimSpace(m[2])
				qualName := qualPrefix(stack) + name
				isErr := false
				if conformanceStr != "" {
					for _, c := range splitConformances(conformanceStr) {
						stripped := stripModuleQual(c)
						if stripped == "Error" || stripped == "LocalizedError" {
							isErr = true
							break
						}
					}
				}
				if isErr {
					e := &meta.ErrorStruct{Name: qualName, Availability: av}
					stack = append(stack, frame{name: qualName, kind: "errorStruct", depth: depth, errS: e})
				} else {
					s := &meta.Struct{Name: qualName, Availability: av}
					stack = append(stack, frame{name: qualName, kind: "struct", depth: depth, strct: s})
				}
				continue
			}

			// Top-level class/actor — only push a "class" frame at depth 1 (not nested).
			// Nested structs inside a class are still processed as structs/enums.
			if m := reClassDecl.FindStringSubmatch(line); m != nil && isTopLevelClass(stack) {
				name := m[1]
				conformanceStr := strings.TrimSpace(m[2])
				qualName := qualPrefix(stack) + name
				cls := &meta.Class{Name: qualName, Availability: av}
				// Detect modifiers from the raw line.
				for _, kw := range reClassKind.FindAllString(line, -1) {
					switch kw {
					case "final":
						cls.IsFinal = true
					case "open":
						cls.IsOpen = true
					case "actor":
						cls.IsActor = true
					}
				}
				if conformanceStr != "" {
					cls.Conformances = splitConformances(conformanceStr)
				}
				stack = append(stack, frame{name: qualName, kind: "class", depth: depth, cls: cls})
				continue
			}

			// If inside a class frame, check for a multi-line computed property opening:
			// `public var isReady: Swift.Bool {`
			if len(stack) > 0 && stack[len(stack)-1].kind == "class" {
				top := &stack[len(stack)-1]
				if m := rePropMember.FindStringSubmatch(line); m != nil {
					propName := m[1]
					rawTypeStr := strings.TrimSpace(m[2])
					// Strip trailing `{` — the brace is the opener of the property body.
					if idx := strings.Index(rawTypeStr, "{"); idx >= 0 {
						rawTypeStr = strings.TrimSpace(rawTypeStr[:idx])
					}
					rawType := resolveType(rawTypeStr, aliases)
					optional := strings.HasSuffix(rawType, "?")
					if optional {
						rawType = rawType[:len(rawType)-1]
					}
					// Check both the stripped line and the original annotated line for MainActor.
					checkLine := line
					if attrLine != "" {
						checkLine = attrLine
					}
					isMainActor := strings.Contains(checkLine, "@_Concurrency.MainActor") || strings.Contains(checkLine, "MainActor")
					isStatic := reStaticPrefix.MatchString(line)
					// isAsync and writable will be determined when we see the accessor body.
					// For now record the property; async/writable detection happens via "get async" body line.
					// We store a provisional property and update it in the skip frame's cleanup.
					// Simpler: read the next lines' content — but we can't look ahead.
					// Best effort: mark it not-async here; the majority of non-async props outnumber async ones.
					// The framework emitter will later check if it needs to call async path.
					// TODO Phase 2C: refine async detection for multi-line props.
					top.cls.Properties = append(top.cls.Properties, meta.Property{
						Name:         propName,
						Type:         rawType,
						Optional:     optional,
						Writable:     strings.Contains(line, " var "),
						IsAsync:      false, // updated by getAsyncBody scan below
						IsStatic:     isStatic,
						IsMainActor:  isMainActor,
						Availability: av,
					})
				}
			}

			// Extension, func body, or other: track name for nested type qualification.
			name := ""
			if m := reExtension.FindStringSubmatch(line); m != nil {
				name = stripModuleQual(m[1])
			} else if m := reClassDecl.FindStringSubmatch(line); m != nil {
				// Nested class inside another type — treat as skip with name.
				name = m[1]
			}
			stack = append(stack, frame{name: qualPrefix(stack) + name, kind: "skip", depth: depth})
			continue
		}

		// Net brace == 0: potential member declaration.
		if len(stack) == 0 {
			pendingAvail = nil
			continue
		}

		top := &stack[len(stack)-1]

		// Skip computed properties (same-line { get } etc.)
		if isComputedPropLine(line) {
			pendingAvail = nil
			continue
		}

		memberAvail := availOrNil(pendingAvail)
		pendingAvail = nil

		switch top.kind {
		case "enum":
			if m := reCase.FindStringSubmatch(line); m != nil {
				// Skip cases with associated values: `case foo(Int)`.
				if !strings.Contains(m[1], "(") {
					raw := ""
					if len(m) > 2 {
						raw = strings.TrimSpace(m[2])
					}
					top.enum.Cases = append(top.enum.Cases, meta.EnumCase{
						Name:         m[1],
						RawValue:     raw,
						Availability: memberAvail,
					})
				}
			}

		case "errorStruct":
			if m := reStaticLet.FindStringSubmatch(line); m != nil {
				top.errS.StaticCodes = append(top.errS.StaticCodes, meta.StaticCode{
					Name:         m[1],
					Availability: memberAvail,
				})
			}

		case "struct":
			if m := reStaticLet.FindStringSubmatch(line); m != nil {
				top.strct.StaticValues = append(top.strct.StaticValues, meta.StaticValue{
					Name:         m[1],
					Availability: memberAvail,
				})
				continue
			}
			if m := reField.FindStringSubmatch(line); m != nil {
				// Skip static fields (already handled by reStaticLet above).
				if strings.Contains(line[:strings.Index(line, m[1])], "static") {
					continue
				}
				fieldName := m[1]
				rawType := resolveType(strings.TrimSpace(m[2]), aliases)
				optional := strings.HasSuffix(rawType, "?")
				if optional {
					rawType = rawType[:len(rawType)-1]
				}
				goType := typemap.Map(rawType, optional)
				if goType == "" {
					continue // skip unmappable types
				}
				top.strct.Fields = append(top.strct.Fields, meta.StructField{
					Name:         fieldName,
					GoType:       goType,
					Optional:     optional,
					Availability: memberAvail,
				})
			}

		case "class":
			parseClassMember(line, attrLine, top.cls, memberAvail, aliases)

		case "skip":
			// Detect `get async` inside a class property body to backpatch IsAsync.
			if line == "get async" && len(stack) >= 2 {
				parent := &stack[len(stack)-2]
				if parent.kind == "class" && len(parent.cls.Properties) > 0 {
					parent.cls.Properties[len(parent.cls.Properties)-1].IsAsync = true
				}
			}
		}
	}

	// Flush any unclosed frames (malformed input guard).
	for _, f := range stack {
		finalizeFrame(f, framework)
	}

	return framework, scanner.Err()
}

// finalizeFrame appends a completed type to the framework metadata.
func finalizeFrame(f frame, framework *meta.FrameworkMeta) {
	switch f.kind {
	case "enum":
		if len(f.enum.Cases) > 0 {
			framework.Enums = append(framework.Enums, f.enum)
		}
	case "errorStruct":
		if len(f.errS.StaticCodes) > 0 {
			framework.ErrorStructs = append(framework.ErrorStructs, f.errS)
		}
	case "struct":
		s := f.strct
		// Mark as value-enum (opaque type with named constants) when there are no
		// stored fields but there are static-let members.
		if len(s.Fields) == 0 && len(s.StaticValues) > 0 {
			s.IsValueEnum = true
		}
		// Only emit structs with at least one field or static value.
		if len(s.Fields) > 0 || len(s.StaticValues) > 0 {
			framework.Structs = append(framework.Structs, s)
		}
	case "class":
		framework.Classes = append(framework.Classes, f.cls)
	}
}

// isTopLevelClass returns true when the current stack position is at the module level
// (no enclosing class/struct/enum) — only top-level classes get a "class" frame.
func isTopLevelClass(stack []frame) bool {
	for _, f := range stack {
		if f.kind == "class" || f.kind == "struct" || f.kind == "enum" || f.kind == "errorStruct" {
			return false
		}
	}
	return true
}

// lineHasDeclaration reports whether a line that starts with "@" also contains a
// func, var, let, or init declaration that should be parsed.
func lineHasDeclaration(line string) bool {
	return strings.Contains(line, " func ") ||
		strings.Contains(line, " var ") ||
		strings.Contains(line, " let ") ||
		strings.Contains(line, " init(") ||
		strings.Contains(line, " init?(") ||
		strings.Contains(line, " init!(")
}

// stripLeadingAttributes removes "@attr" prefixes from a line until a non-attribute
// token is reached. Returns the remainder starting at the first access modifier or keyword.
// Example: "@_Concurrency.MainActor public func foo()" → "public func foo()"
func stripLeadingAttributes(line string) string {
	s := strings.TrimSpace(line)
	for strings.HasPrefix(s, "@") {
		// Find the end of this attribute token (space or end of string).
		end := strings.IndexByte(s, ' ')
		if end < 0 {
			return "" // attribute-only line
		}
		s = strings.TrimSpace(s[end:])
	}
	return s
}

// parseClassMember parses a single member declaration line within a class body.
// line is the declaration (attributes stripped if they were present).
// attrLine is the original annotated line (before stripping), or "" if no attributes were present.
// It appends methods or properties to cls as appropriate.
func parseClassMember(line, attrLine string, cls *meta.Class, av meta.Availability, aliases map[string]string) {
	// Check both the declaration line and the original annotated line for MainActor.
	checkLine := line
	if attrLine != "" {
		checkLine = attrLine
	}
	isMainActor := strings.Contains(checkLine, "@_Concurrency.MainActor") || strings.Contains(checkLine, "MainActor")
	isStatic := reStaticPrefix.MatchString(line)

	// Strip leading @attributes that appear inline (e.g. @_Concurrency.MainActor).
	// After stripping, the line starts with public/open/final/convenience/required/mutating/func/var/let/init.
	clean := line
	if strings.HasPrefix(clean, "@") {
		clean = stripLeadingAttributes(clean)
	}

	// Skip operators, deinit, and other non-bridgeable members.
	if strings.Contains(clean, "deinit") {
		return
	}

	// Initializer.
	if m := reInitDecl.FindStringSubmatch(clean); m != nil {
		if strings.Contains(clean, "func ") {
			// Looks like a func with "init" in a param name — skip.
		} else {
			rawParams := strings.TrimSpace(m[1])
			cls.Methods = append(cls.Methods, meta.Method{
				Name:         "init",
				RawParams:    rawParams,
				IsInit:       true,
				IsAsync:      strings.Contains(clean, " async"),
				IsThrows:     strings.Contains(clean, " throws"),
				IsMainActor:  isMainActor,
				Availability: av,
			})
			return
		}
	}

	// Regular method.
	if m := reFuncDecl.FindStringSubmatch(clean); m != nil {
		name := m[1]
		// Skip operator methods, hash(into:), and encode(to:) boilerplate.
		if isOperatorOrBoilerplate(name, clean) {
			return
		}
		rawParams := strings.TrimSpace(m[2])
		retType := strings.TrimSpace(m[5])
		retType = resolveType(retType, aliases)
		optional := strings.HasSuffix(retType, "?")
		if optional {
			retType = retType[:len(retType)-1]
		}
		cls.Methods = append(cls.Methods, meta.Method{
			Name:      name,
			RawParams: rawParams,
			Return: meta.SwiftReturn{
				Type:     retType,
				Optional: optional,
			},
			IsAsync:      strings.TrimSpace(m[3]) == "async",
			IsThrows:     strings.TrimSpace(m[4]) == "throws",
			IsStatic:     isStatic,
			IsMainActor:  isMainActor,
			Availability: av,
		})
		return
	}

	// Property (single-line computed or stored).
	if isComputedPropLine(line) {
		if m := rePropMember.FindStringSubmatch(clean); m != nil {
			propName := m[1]
			typeAndAccessor := strings.TrimSpace(m[2])
			// Extract type before the `{`.
			rawType := typeAndAccessor
			if idx := strings.Index(typeAndAccessor, "{"); idx >= 0 {
				rawType = strings.TrimSpace(typeAndAccessor[:idx])
			}
			rawType = resolveType(rawType, aliases)
			optional := strings.HasSuffix(rawType, "?")
			if optional {
				rawType = rawType[:len(rawType)-1]
			}
			isAsync := strings.Contains(typeAndAccessor, "get async")
			// Stored let/var properties treated as writable if declared var.
			writable := strings.Contains(clean, " var ") && strings.Contains(typeAndAccessor, "set")
			cls.Properties = append(cls.Properties, meta.Property{
				Name:         propName,
				Type:         rawType,
				Optional:     optional,
				Writable:     writable,
				IsAsync:      isAsync,
				IsStatic:     isStatic,
				IsMainActor:  isMainActor,
				Availability: av,
			})
		}
		return
	}

	// Stored property (no braces — final let / var).
	if m := rePropMember.FindStringSubmatch(clean); m != nil {
		if !strings.Contains(clean, " func ") && !strings.Contains(clean, " init") {
			propName := m[1]
			rawType := resolveType(strings.TrimSpace(m[2]), aliases)
			optional := strings.HasSuffix(rawType, "?")
			if optional {
				rawType = rawType[:len(rawType)-1]
			}
			writable := strings.Contains(clean, " var ")
			cls.Properties = append(cls.Properties, meta.Property{
				Name:         propName,
				Type:         rawType,
				Optional:     optional,
				Writable:     writable,
				IsAsync:      false,
				IsStatic:     isStatic,
				IsMainActor:  isMainActor,
				Availability: av,
			})
		}
	}
}

// isOperatorOrBoilerplate returns true for method names that are operators or
// Equatable/Hashable boilerplate that we don't bridge.
func isOperatorOrBoilerplate(name, line string) bool {
	switch name {
	case "hash", "encode":
		return true
	}
	// Operator methods have symbols in them.
	if strings.ContainsAny(name, "=<>+-*/!&|^~%") {
		return true
	}
	// Static == and != operators.
	if strings.Contains(line, "static func ==") || strings.Contains(line, "static func !=") {
		return true
	}
	return false
}

// qualPrefix returns the Go name prefix for a nested type.
// If the stack top is a named frame, return that name; otherwise "".
func qualPrefix(stack []frame) string {
	for i := len(stack) - 1; i >= 0; i-- {
		if stack[i].name != "" && stack[i].kind != "skip" {
			return stack[i].name
		}
		// For "skip" frames that carry a class name, use it as prefix.
		if stack[i].kind == "skip" && stack[i].name != "" {
			return stack[i].name
		}
	}
	return ""
}

// splitConformances splits a `:` conformance list like "Swift.Equatable, Swift.Sendable"
// into individual items.
func splitConformances(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		t := strings.TrimSpace(part)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

// stripModuleQual strips the module prefix from a qualified name.
// "Foundation.LocalizedError" → "LocalizedError", "Swift.Error" → "Error"
func stripModuleQual(s string) string {
	s = strings.TrimSpace(s)
	if idx := strings.LastIndex(s, "."); idx >= 0 {
		return s[idx+1:]
	}
	return s
}

// resolveType strips module-alias prefixes from a raw Swift type string.
// aliases: {"Module___Translation" → "Translation"}.
// The type is further simplified by removing the current framework qualifier.
func resolveType(rawType string, aliases map[string]string) string {
	s := rawType
	// Strip trailing `{ get }` etc. from property declarations.
	if idx := strings.Index(s, "{"); idx >= 0 {
		s = strings.TrimSpace(s[:idx])
	}
	// Strip trailing `= default` values.
	if idx := strings.Index(s, "="); idx >= 0 {
		s = strings.TrimSpace(s[:idx])
	}
	// Apply module aliases: "Module___Translation." → ""
	for alias, real := range aliases {
		s = strings.ReplaceAll(s, alias+"."+real+".", "") // Module___X.X.Type → Type
		s = strings.ReplaceAll(s, alias+".", real+".")    // Module___X.Type → X.Type
	}
	// Strip known module prefixes.
	for _, pfx := range []string{"Swift.", "Foundation.", "CoreFoundation."} {
		s = strings.ReplaceAll(s, pfx, "")
	}
	return strings.TrimSpace(s)
}

// isComputedPropLine returns true when the line is a single-line computed property
// like `public var foo: Int { get }`. Multi-line computed properties open a `{` at
// end of line, which produces netBraces > 0 and is handled separately.
func isComputedPropLine(line string) bool {
	idx := strings.Index(line, "{")
	if idx < 0 {
		return false
	}
	body := strings.TrimSpace(line[idx+1:])
	return strings.HasPrefix(body, "get") ||
		strings.HasPrefix(body, "set") ||
		strings.HasPrefix(body, "mutating") ||
		strings.HasPrefix(body, "nonmutating") ||
		strings.HasPrefix(body, "_read") ||
		strings.HasPrefix(body, "_modify")
}

// availOrNil returns the Availability value pointed to by av, or a zero value.
func availOrNil(av *meta.Availability) meta.Availability {
	if av == nil {
		return meta.Availability{}
	}
	return *av
}

// mergeAvailability folds b into a, keeping the most restrictive/specific info.
func mergeAvailability(a *meta.Availability, b meta.Availability) {
	if b.MacOSUnavailable {
		a.MacOSUnavailable = true
	}
	if b.MacOSIntroduced != "" && a.MacOSIntroduced == "" {
		a.MacOSIntroduced = b.MacOSIntroduced
	}
}

// parseAvailability parses the argument list from @available(...).
// Input is the content inside the parentheses: "iOS 18.0, macOS 15.0, *"
func parseAvailability(args string) meta.Availability {
	var av meta.Availability
	parts := strings.Split(args, ",")
	i := 0
	for i < len(parts) {
		tok := strings.TrimSpace(parts[i])
		if strings.HasPrefix(tok, "macOS") {
			rest := strings.TrimSpace(tok[len("macOS"):])
			if rest == "" || rest == "," {
				// "macOS" alone might be followed by a version in next token? No — the
				// comma is already the split. So "macOS" alone with next token "unavailable"
				// means we need to check the next token.
				if i+1 < len(parts) {
					next := strings.TrimSpace(parts[i+1])
					if next == "unavailable" {
						av.MacOSUnavailable = true
						i += 2
						continue
					}
				}
			} else if rest == "unavailable" {
				av.MacOSUnavailable = true
			} else if rest != "" && rest[0] >= '0' && rest[0] <= '9' {
				// "macOS 15.0" → "15.0"
				av.MacOSIntroduced = rest
			}
		}
		i++
	}
	return av
}

// extractModuleAliases parses the swift-module-flags comment for -module-alias entries.
// Returns a map from alias → real name, e.g. {"Module___Translation": "Translation"}.
func extractModuleAliases(content, framework string) map[string]string {
	aliases := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(content))
	aliasRE := regexp.MustCompile(`-module-alias\s+(\S+)=(\S+)`)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "//") {
			break // header comments end at first non-comment line
		}
		for _, m := range aliasRE.FindAllStringSubmatch(line, -1) {
			aliases[m[1]] = m[2]
		}
	}
	// Also add the framework itself: "Translation.Foo" → just "Foo"
	aliases[framework] = framework
	return aliases
}
