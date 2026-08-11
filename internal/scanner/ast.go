//go:build darwin

package scanner

import (
	"encoding/json"
	"strings"
)

// RawValue holds an AST literal value which Clang may emit as a JSON number
// or as a JSON string. It normalises both to a plain string.
type RawValue string

func (r *RawValue) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	*r = RawValue(s)
	return nil
}

// MarshalJSON writes the value as a JSON string.
func (r RawValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(r))
}

// String returns the value as a plain Go string.
func (r RawValue) String() string { return string(r) }

// ASTNode is a node in the Clang JSON AST produced by clang -ast-dump=json.
// Only fields we actually use are decoded; unknown fields are silently ignored.
type ASTNode struct {
	ID          string    `json:"id"`
	Kind        string    `json:"kind"`
	Loc         *Location `json:"loc"`
	Range       *SrcRange `json:"range"`
	Name        string    `json:"name"`
	MangledName string    `json:"mangledName"`
	Type        *ASTType  `json:"type"`
	ReturnType  *ASTType  `json:"returnType"`
	Inner       []ASTNode `json:"inner"`

	// PreviousDecl is the hex ID string of the previous declaration of this
	// name in the same translation unit. When set, this node is a redeclaration
	// (e.g. @class Foo; repeated after the canonical @interface Foo...@end, or
	// an empty @class Foo; that precedes the canonical definition). Used in
	// extractClass to distinguish canonical definitions from foreign-class
	// forward declarations.
	PreviousDecl string `json:"previousDecl"`

	// ObjCInterfaceDecl / ObjCCategoryDecl
	Super     *ASTRef  `json:"super"`
	Protocols []ASTRef `json:"protocols"`
	// ObjCCategoryDecl: the class this category extends
	Interface *ASTRef `json:"interface"`

	// ObjCMethodDecl — Clang uses "instance": true for instance methods,
	// absence of the field (or false) means class method.
	IsInstance bool `json:"instance"`
	IsVariadic bool `json:"variadic"`
	IsImplicit bool `json:"implicit"`

	// ObjCPropertyDecl
	Getter *ASTRef `json:"getter"`
	Setter *ASTRef `json:"setter"`
	// property attribute flags: "readonly", "copy", "weak", "assign", etc.
	PropertyAttributes []string `json:"propertyAttributes"`

	// EnumDecl
	FixedUnderlyingType *ASTType `json:"fixedUnderlyingType"`

	// EnumConstantDecl
	// value lives inside Inner as an IntegerLiteral or ConstantExpr

	// RecordDecl (struct/union)
	TagUsed            string `json:"tagUsed"`            // "struct" or "union"
	CompleteDefinition bool   `json:"completeDefinition"` // true for full record decl, false/missing for forward decl

	// VarDecl
	StorageClass string `json:"storageClass"` // "extern"

	// IntegerLiteral / FloatingLiteral / ConstantExpr
	// Clang emits numeric literals as JSON numbers; use RawMessage to handle both
	// number and string variants uniformly.
	Value RawValue `json:"value"`

	// Availability
	Availability []ASTNode `json:"availability"` // AvailabilityAttr nodes

	// AvailabilityAttr fields
	Platform   string `json:"platform"` // "macos", "ios"
	Introduced string `json:"introduced"`
	Deprecated string `json:"deprecated"`
	Obsoleted  string `json:"obsoleted"`
	Message    string `json:"message"`

	// ObjCTypeParamDecl — generic type params on a class
	// appears as inner nodes of ObjCInterfaceDecl
	Bound *ASTType `json:"bound"` // upper bound (usually "id")

	// Nullability qualifier attached to a type
	NullabilityQual string `json:"nullabilityQual"` // "nullable", "nonnull", "unspecified"

	// SwiftNameAttr carries the NS_SWIFT_NAME value; appears as an inner node.
	SwiftName string `json:"swiftName,omitempty"`
}

// hasAttr reports whether this node has a direct child with the given kind.
func (n *ASTNode) hasAttr(kind string) bool {
	for i := range n.Inner {
		if n.Inner[i].Kind == kind {
			return true
		}
	}
	return false
}

// attrName returns the Name of the first inner child with the given kind, or "".
// Used to extract string-valued attributes such as SwiftNameAttr.
func (n *ASTNode) attrName(kind string) string {
	for i := range n.Inner {
		if n.Inner[i].Kind == kind {
			// The value may be in Name or SwiftName depending on how Clang serialises it.
			if n.Inner[i].SwiftName != "" {
				return n.Inner[i].SwiftName
			}
			return n.Inner[i].Name
		}
	}
	return ""
}

// IncludedFromRef holds the file that directly includes the file containing
// a source location. Clang emits this when the declaration file is not the
// translation unit root.
type IncludedFromRef struct {
	File string `json:"file"`
}

type Location struct {
	FilePath string `json:"file"`
	Line     int    `json:"line"`
	Col      int    `json:"col"`
	// When the loc is a macro expansion the actual file is in ExpansionLoc
	ExpansionLoc *Location `json:"expansionLoc"`
	// Spelling location (pre-macro-expansion)
	SpellingLoc *Location `json:"spellingLoc"`
	// IncludedFrom is the file that directly includes the file this loc is in.
	// Present when loc.file is absent (Clang cursor optimisation) or when Clang
	// wants to indicate the include chain.
	IncludedFrom *IncludedFromRef `json:"includedFrom"`
}

// ResolvedFile returns the most useful file path from this location, unwrapping
// macro expansions to find the real header file.
// For macro-expanded declarations (NS_ENUM, NS_OPTIONS, etc.) the expansion
// location is the call site in the SDK header, while the spelling location is
// the macro definition inside CoreFoundation. We want the call site.
func (l *Location) ResolvedFile() string {
	if l == nil {
		return ""
	}
	if l.ExpansionLoc != nil && l.ExpansionLoc.FilePath != "" {
		return l.ExpansionLoc.FilePath
	}
	if l.SpellingLoc != nil && l.SpellingLoc.FilePath != "" {
		return l.SpellingLoc.FilePath
	}
	return l.FilePath
}

// IncludedFromFile returns the path of the file that directly includes the file
// this location is in, or "" if none was recorded. Used by the framework filter
// to determine true framework ownership when loc.file is absent.
func (l *Location) IncludedFromFile() string {
	if l == nil || l.IncludedFrom == nil {
		return ""
	}
	return l.IncludedFrom.File
}

// expansionFile returns the macro expansion file (the call site in the SDK header),
// as opposed to ResolvedFile which prefers expansionLoc but falls back to spellingLoc.
// Used to locate AvailabilityAttr annotations for Apple Clang ≥ 21 which omits
// the platform field and relies on source-level parsing.
func (l *Location) expansionFile() string {
	if l == nil {
		return ""
	}
	if l.ExpansionLoc != nil && l.ExpansionLoc.FilePath != "" {
		return l.ExpansionLoc.FilePath
	}
	return l.FilePath
}

// expansionLine returns the expansion line number (macro call-site line),
// without falling back to the spelling location.
func (l *Location) expansionLine() int {
	if l == nil {
		return 0
	}
	if l.ExpansionLoc != nil && l.ExpansionLoc.Line > 0 {
		return l.ExpansionLoc.Line
	}
	return l.Line
}

// ResolvedLine returns the source line number, preferring spellingLoc over expansionLoc.
func (l *Location) ResolvedLine() int {
	if l == nil {
		return 0
	}
	if l.SpellingLoc != nil && l.SpellingLoc.Line > 0 {
		return l.SpellingLoc.Line
	}
	if l.ExpansionLoc != nil && l.ExpansionLoc.Line > 0 {
		return l.ExpansionLoc.Line
	}
	return l.Line
}

type SrcRange struct {
	Begin Location `json:"begin"`
	End   Location `json:"end"`
}

type ASTType struct {
	QualType          string `json:"qualType"` // e.g. "NSArray<ObjectType> *", "NSUInteger"
	TypeAliasDeclID   string `json:"typeAliasDeclId"`
	DesugaredQualType string `json:"desugaredQualType"` // after typedef expansion
}

type ASTRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
