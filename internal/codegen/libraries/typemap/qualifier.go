package typemap

import (
	"fmt"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/naming"
)

// BsdModulePath is the fixed import path of the POSIX/BSD support package. It
// stays public (outside bindings/internal/raw) because value structs it defines
// — bsd.Timespec, bsd.EtherAddr, … — appear in the signatures of both the raw
// libraries and the public idiomatic ones.
const BsdModulePath = "github.com/deploymenttheory/go-bindings-macosplatform/bindings/libraries/bsd"

// qualifier resolves cross-framework type references and records the import
// side effects and diagnostics that result from doing so.
// It holds only the fields required for qualification; it is constructable
// independently of a full Mapper, making each method unit-testable in isolation.
type qualifier struct {
	framework      string
	modulePrefix   string
	frameworkOwner map[string]string
	knownProtocols map[string]string
	blockedImports map[string]map[string]bool
	usedImports    map[string]string // populated as a side effect
	diagnostics    *[]string         // pointer to Mapper.Diagnostics
}

// buildQualifier constructs a qualifier from a Mapper, the per-call Context, and an
// explicit ImportSet. Stable, run-wide fields come from m; per-call fields
// (framework) come from ctx; imports are the caller-supplied collector.
func (m *Mapper) buildQualifier(ctx Context, imports ImportSet) qualifier {
	return qualifier{
		framework:      ctx.Framework,
		modulePrefix:   m.ModulePrefix,
		frameworkOwner: m.OwnerIndex,
		knownProtocols: m.ProtocolIndex,
		blockedImports: m.BlockedImports,
		usedImports:    imports,
		diagnostics:    &m.Diagnostics,
	}
}

// isBlocked returns true when importing toFramework from q.framework is forbidden.
func (q qualifier) isBlocked(toFramework string) bool {
	return q.blockedImports[q.framework][toFramework]
}

// recordImport side-effects the pkgAlias→importPath pair into usedImports.
func (q qualifier) recordImport(pkgAlias string) {
	if q.usedImports != nil && q.modulePrefix != "" {
		q.usedImports[pkgAlias] = q.modulePrefix + "/" + pkgAlias
	}
}

// appendDiag appends a formatted diagnostic about a cycle-forced unsafe.Pointer.
func (q qualifier) appendDiag(format string, args ...any) {
	*q.diagnostics = append(*q.diagnostics, fmt.Sprintf(format, args...))
}

// classType returns the Go pointer type expression for an ObjC class reference,
// package-qualifying it when the class is owned by a different framework.
func (q qualifier) classType(class, typeExpr string) string {
	owner := q.frameworkOwner[class]
	if owner == "" || strings.EqualFold(owner, q.framework) {
		return "*" + typeExpr
	}
	if q.isBlocked(owner) {
		q.appendDiag("%s: %s.%s replaced with unsafe.Pointer (import cycle %s→%s)",
			q.framework, strings.ToLower(owner), class,
			strings.ToLower(q.framework), strings.ToLower(owner))
		return "unsafe.Pointer"
	}
	pkgAlias := strings.ToLower(owner)
	q.recordImport(pkgAlias)
	return "*" + pkgAlias + "." + typeExpr
}

// enumType returns the Go type expression for a named ObjC enum, qualifying
// with a package alias when the enum belongs to a different framework.
// No leading "*" — enums are value types.
func (q qualifier) enumType(name, owner string) string {
	goName := naming.ExportedTypeName(name)
	if owner == "" || strings.EqualFold(owner, q.framework) {
		return goName
	}
	if q.isBlocked(owner) {
		q.appendDiag("%s: enum %s.%s replaced with unsafe.Pointer (import cycle %s→%s)",
			q.framework, strings.ToLower(owner), goName,
			strings.ToLower(q.framework), strings.ToLower(owner))
		return "unsafe.Pointer"
	}
	pkgAlias := strings.ToLower(owner)
	q.recordImport(pkgAlias)
	return pkgAlias + "." + goName
}

// protocolType returns the Go interface type expression for an id<Protocol…>
// reference. Single-protocol → bare or package-qualified interface name.
// Multi-protocol → inline anonymous composite "interface { P; Q }".
// Returns "" when none of the protocols are in knownProtocols.
func (q qualifier) protocolType(protos []string) string {
	exprs := make([]string, 0, len(protos))
	for _, p := range protos {
		owner, ok := q.knownProtocols[p]
		if !ok {
			continue
		}
		name := q.protocolGoName(p, owner)
		if name == "" {
			continue // cycle — drop this constraint
		}
		exprs = append(exprs, name)
	}
	if len(exprs) == 0 {
		return ""
	}
	if len(exprs) == 1 {
		return exprs[0]
	}
	return "interface { " + strings.Join(exprs, "; ") + " }"
}

// protocolIDType returns the Go pointer type for a protocol id<P> wrapper struct
// (e.g. *VZVirtualMachineDelegateIDProtocol) or "" when not known or import-blocked.
// knownIDProtocols maps protocol name → owning framework.
func (q qualifier) protocolIDType(proto string, knownIDProtocols map[string]string) string {
	owner, ok := knownIDProtocols[proto]
	if !ok {
		return ""
	}
	goProto := naming.ProtocolGoTypeName(proto, q.frameworkOwner)
	idTypeName := goProto + "IDProtocol"
	if owner == "" || strings.EqualFold(owner, q.framework) {
		return "*" + idTypeName
	}
	if q.isBlocked(owner) {
		return ""
	}
	pkgAlias := strings.ToLower(owner)
	q.recordImport(pkgAlias)
	return "*" + pkgAlias + "." + idTypeName
}

// protocolGoName returns the Go interface name for a protocol, package-qualified
// when cross-framework. Returns "" to signal a blocked import cycle to callers.
func (q qualifier) protocolGoName(proto, owner string) string {
	goProto := naming.ProtocolGoTypeName(proto, q.frameworkOwner)
	if owner == "" || strings.EqualFold(owner, q.framework) {
		return goProto
	}
	if q.isBlocked(owner) {
		q.appendDiag("%s: protocol %s.%s replaced with unsafe.Pointer (import cycle %s→%s)",
			q.framework, strings.ToLower(owner), goProto,
			strings.ToLower(q.framework), strings.ToLower(owner))
		return ""
	}
	pkgAlias := strings.ToLower(owner)
	q.recordImport(pkgAlias)
	return pkgAlias + "." + goProto
}

// structType returns the Go expression for a struct value type.
// Same-framework: bare "<Name>". Cross-framework: "<packageName>.<Name>".
// addPointer prepends "*" for pointer-to-struct contexts.
func (q qualifier) structType(name, owner string, addPointer bool) string {
	prefix := ""
	if addPointer {
		prefix = "*"
	}
	goName := naming.ExportedTypeName(name)
	if owner == "" || strings.EqualFold(owner, q.framework) {
		return prefix + goName
	}
	if q.isBlocked(owner) {
		q.appendDiag("%s: struct %s.%s replaced with unsafe.Pointer (import cycle %s→%s)",
			q.framework, strings.ToLower(owner), goName,
			strings.ToLower(q.framework), strings.ToLower(owner))
		return "unsafe.Pointer"
	}
	pkgAlias := strings.ToLower(owner)
	q.recordImport(pkgAlias)
	return prefix + pkgAlias + "." + goName
}

// bsdType returns the Go type for a POSIX/BSD struct from the bsd package
// (e.g. bsd.Timespec, *bsd.EtherAddr). goName is the exported Go type name
// in the bsd package (e.g. "Timespec", "EtherAddr"). The bsd import path is
// pinned to the public BsdModulePath (bsd stays outside bindings/internal/raw).
func (q qualifier) bsdType(goName string, addPointer bool) string {
	prefix := ""
	if addPointer {
		prefix = "*"
	}
	const pkgAlias = "bsd"
	if q.isBlocked(pkgAlias) {
		return "unsafe.Pointer"
	}
	if q.usedImports != nil {
		// bsd stays PUBLIC at bindings/libraries/bsd even after the raw bindings
		// move under bindings/internal/raw, so its import path is pinned rather
		// than derived from the (now internal) module prefix.
		q.usedImports[pkgAlias] = BsdModulePath
	}
	return prefix + pkgAlias + "." + goName
}

// cfType returns the Go type for a CoreFoundation opaque typedef (e.g. CFStringRef).
// Within the CoreFoundation package: "*CFStringRef". Elsewhere: "*corefoundation.CFStringRef".
func (q qualifier) cfType(name string) string {
	const cfFramework = "CoreFoundation"
	goName := naming.ExportedTypeName(name)
	if strings.EqualFold(q.framework, cfFramework) {
		return "*" + goName
	}
	if q.isBlocked(cfFramework) {
		q.appendDiag("%s: CF type %s replaced with unsafe.Pointer (import cycle %s→corefoundation)",
			q.framework, name, strings.ToLower(q.framework))
		return "unsafe.Pointer"
	}
	const pkgAlias = "corefoundation"
	q.recordImport(pkgAlias)
	return "*" + pkgAlias + "." + goName
}

// RebuildGoNameIndices populates the Go-name-keyed lookup sets (structGoNames,
// enumGoIntByGoName) from the C-name-keyed StructIndex and EnumGoTypeIndex, using
// the same naming.ExportedTypeName mapping the emitters apply to type
// declarations. Because that mapping (snake_case → PascalCase) is not invertible,
// classification code cannot recover a C name from a resolved Go name; these
// forward-built sets let it look the Go name up directly. Call once after the
// C-name-keyed indices are assigned.
func (m *Mapper) RebuildGoNameIndices() {
	m.structGoNames = make(map[string]bool, len(m.StructIndex))
	for cName := range m.StructIndex {
		m.structGoNames[naming.ExportedTypeName(cName)] = true
	}
	m.enumGoIntByGoName = make(map[string]string, len(m.EnumGoTypeIndex))
	for cName, goInt := range m.EnumGoTypeIndex {
		m.enumGoIntByGoName[naming.ExportedTypeName(cName)] = goInt
	}
}

// IsStructGoName reports whether goName (a resolved Go type name with any package
// prefix already stripped) is the exported name of a registered value struct.
func (m *Mapper) IsStructGoName(goName string) bool {
	return m.structGoNames[goName]
}

// capitaliseFirst uppercases the first ASCII character of s.
func capitaliseFirst(s string) string {
	if s == "" {
		return s
	}
	if s[0] >= 'a' && s[0] <= 'z' {
		return string(s[0]-32) + s[1:]
	}
	return s
}

// frameworkCFType returns the Go pointer type for a framework-owned CF opaque
// typedef (e.g. CFHTTPMessageRef owned by CFNetwork).
// Same-framework: "*TypedefName". Cross-framework: "*pkgAlias.TypedefName".
// Import-blocked: "unsafe.Pointer".
func (q qualifier) frameworkCFType(name, owner string) string {
	goName := naming.ExportedTypeName(name)
	if owner == "" || strings.EqualFold(owner, q.framework) {
		return "*" + goName
	}
	pkgAlias := strings.ToLower(owner)
	if q.isBlocked(owner) {
		q.appendDiag("%s: CF type %s.%s replaced with unsafe.Pointer (import cycle %s→%s)",
			q.framework, pkgAlias, name, strings.ToLower(q.framework), pkgAlias)
		return "unsafe.Pointer"
	}
	q.recordImport(pkgAlias)
	return "*" + pkgAlias + "." + goName
}

// --- Mapper thin wrappers ---

func (m *Mapper) qualifiedType(class, typeExpr string, ctx Context, imports ImportSet) string {
	return m.buildQualifier(ctx, imports).classType(class, typeExpr)
}

func (m *Mapper) qualifiedFrameworkCFType(
	name, owner string,
	ctx Context,
	imports ImportSet,
) string {
	return m.buildQualifier(ctx, imports).frameworkCFType(name, owner)
}

func (m *Mapper) qualifiedEnumType(name, owner string, ctx Context, imports ImportSet) string {
	return m.buildQualifier(ctx, imports).enumType(name, owner)
}

func (m *Mapper) qualifiedProtocolType(protos []string, ctx Context, imports ImportSet) string {
	return m.buildQualifier(ctx, imports).protocolType(protos)
}

func (m *Mapper) qualifiedProtocolIDType(proto string, ctx Context, imports ImportSet) string {
	return m.buildQualifier(ctx, imports).protocolIDType(proto, m.ProtocolProxyIndex)
}

func (m *Mapper) qualifiedStructType(
	name, owner string,
	addPointer bool,
	ctx Context,
	imports ImportSet,
) string {
	return m.buildQualifier(ctx, imports).structType(name, owner, addPointer)
}

func (m *Mapper) qualifiedCFType(name string, ctx Context, imports ImportSet) string {
	return m.buildQualifier(ctx, imports).cfType(name)
}

func (m *Mapper) qualifiedBSDType(
	goName string,
	addPointer bool,
	ctx Context,
	imports ImportSet,
) string {
	return m.buildQualifier(ctx, imports).bsdType(goName, addPointer)
}

// BlockedImportNote returns a comment string when the ObjC type qt would be
// replaced with unsafe.Pointer due to an import cycle. Returns "" otherwise.
// Intended for use by emitters to annotate generated code.
func (m *Mapper) BlockedImportNote(qt string, ctx Context) string {
	n := Normalise(qt)
	class := ClassName(n)
	if class == "" {
		return ""
	}
	owner := m.OwnerIndex[class]
	if owner == "" || strings.EqualFold(owner, ctx.Framework) {
		return ""
	}
	if m.BlockedImports[ctx.Framework][owner] {
		return fmt.Sprintf("// %s.%s replaced with unsafe.Pointer: import cycle between %s and %s",
			strings.ToLower(owner), class, strings.ToLower(ctx.Framework), strings.ToLower(owner))
	}
	return ""
}
