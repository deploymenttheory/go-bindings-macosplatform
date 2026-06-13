package classify

import (
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/meta"
)

// PatternTag names a recognisable Objective-C ceremony pattern.
type PatternTag string

const (
	AsyncCompletion    PatternTag = "AsyncCompletion"
	BoolNSError        PatternTag = "BoolNSError"
	NSErrorOut         PatternTag = "NSErrorOut"
	DelegateHolder     PatternTag = "DelegateHolder"
	DelegateProtocol   PatternTag = "DelegateProtocol"
	CollectionReturn   PatternTag = "CollectionReturn"
	CollectionParam    PatternTag = "CollectionParam"
	PropertyPair       PatternTag = "PropertyPair"
	MainThreadRequired PatternTag = "MainThreadRequired"
	BlockEnumeration   PatternTag = "BlockEnumeration"
)

// TaggedMethod pairs a method with its detected pattern tags.
type TaggedMethod struct {
	Method meta.Method
	Tags   []PatternTag
}

// DelegateProperty records a property that holds a delegate or data source.
type DelegateProperty struct {
	PropertyName string
	ProtocolName string // extracted from "id<ProtocolName>"
}

// collection type substrings recognised by CollectionReturn / CollectionParam.
var collectionSubstrings = []string{
	"NSArray", "NSMutableArray",
	"NSSet", "NSMutableSet",
	"NSDictionary", "NSMutableDictionary",
	"NSOrderedSet", "NSMutableOrderedSet",
}

// containsCollection reports whether an ObjC type string mentions any NS collection.
func containsCollection(objcType string) bool {
	for _, sub := range collectionSubstrings {
		if strings.Contains(objcType, sub) {
			return true
		}
	}
	return false
}

// blockReturnsVoid reports whether a block ObjC type string has a void return.
// Typical shapes: "void (^)(NSData *, NSError *)", "void (^)(void)"
func blockReturnsVoid(objcType string) bool {
	t := strings.TrimSpace(objcType)
	return strings.HasPrefix(t, "void (^")
}

// extractProtocolFromID returns the protocol name from "id<ProtocolName>" or "".
func extractProtocolFromID(objcType string) string {
	t := strings.TrimSpace(objcType)
	// Must start with "id<"
	const prefix = "id<"
	if !strings.HasPrefix(t, prefix) {
		return ""
	}
	inner := t[len(prefix):]
	name, _, ok := strings.Cut(inner, ">")
	if !ok {
		return ""
	}
	return strings.TrimSpace(name)
}

// isAsyncCompletion detects the "void method with a void-returning block as last arg" pattern.
// BlockEnumeration is excluded first to avoid false positives.
func isAsyncCompletion(m meta.Method) bool {
	if strings.TrimSpace(m.Return.ObjCType) != "void" {
		return false
	}
	if len(m.Params) == 0 {
		return false
	}
	last := m.Params[len(m.Params)-1]
	if !last.IsBlock {
		return false
	}
	if !blockReturnsVoid(last.ObjCType) {
		return false
	}
	// Exclude enumerate-family methods — those are BlockEnumeration, not AsyncCompletion.
	if strings.Contains(strings.ToLower(m.Selector), "enumerate") && strings.Contains(last.ObjCType, "BOOL *") {
		return false
	}
	return true
}

func isBoolNSError(m meta.Method) bool {
	return m.IsNSError && strings.TrimSpace(m.Return.ObjCType) == "BOOL"
}

func isNSErrorOut(m meta.Method) bool {
	return m.IsNSError && strings.TrimSpace(m.Return.ObjCType) != "BOOL"
}

func isCollectionReturn(m meta.Method) bool {
	return containsCollection(m.Return.ObjCType)
}

func isCollectionParam(m meta.Method) bool {
	for _, arg := range m.Params {
		if containsCollection(arg.ObjCType) {
			return true
		}
	}
	return false
}

// isBlockEnumeration detects "enumerate…UsingBlock:" methods whose block accepts a BOOL * stop param.
func isBlockEnumeration(m meta.Method) bool {
	if !strings.Contains(strings.ToLower(m.Selector), "enumerate") {
		return false
	}
	if len(m.Params) == 0 {
		return false
	}
	last := m.Params[len(m.Params)-1]
	if !last.IsBlock {
		return false
	}
	return strings.Contains(last.ObjCType, "BOOL *")
}

// isPropertyPair reports whether a method selector corresponds to a getter or setter
// of a non-readonly property declared on the class.
func isPropertyPair(m meta.Method, cls meta.Class) bool {
	sel := m.Selector
	for _, p := range cls.Properties {
		if p.IsReadOnly {
			continue
		}
		getter := p.Name
		if p.Getter != "" {
			getter = p.Getter
		}
		setter := p.Setter
		if setter == "" {
			// Derive default setter: setName:
			if len(p.Name) > 0 {
				setter = "set" + strings.ToUpper(p.Name[:1]) + p.Name[1:] + ":"
			}
		}
		if sel == getter || sel == setter {
			return true
		}
	}
	return false
}

// ClassifyMethod returns pattern tags for a single method in a given class context.
// It does NOT return DelegateHolder or DelegateProtocol — those are class/protocol-level;
// use DelegateProperties and IsDelegateProtocol for those.
func ClassifyMethod(m meta.Method, cls meta.Class, _ *meta.FrameworkMeta) []PatternTag {
	var tags []PatternTag

	// Order matters: BlockEnumeration must be checked before AsyncCompletion so that
	// enumerate-family methods with stop blocks are not also tagged AsyncCompletion.
	if isBlockEnumeration(m) {
		tags = append(tags, BlockEnumeration)
	} else if isAsyncCompletion(m) {
		tags = append(tags, AsyncCompletion)
	}

	if isBoolNSError(m) {
		tags = append(tags, BoolNSError)
	} else if isNSErrorOut(m) {
		tags = append(tags, NSErrorOut)
	}

	if isCollectionReturn(m) {
		tags = append(tags, CollectionReturn)
	}
	if isCollectionParam(m) {
		tags = append(tags, CollectionParam)
	}

	if isPropertyPair(m, cls) {
		tags = append(tags, PropertyPair)
	}

	if m.IsMainThreadRequired {
		tags = append(tags, MainThreadRequired)
	}

	return tags
}

// ClassifyFramework returns a two-level map: className → selector → []PatternTag.
func ClassifyFramework(framework *meta.FrameworkMeta) map[string]map[string][]PatternTag {
	result := make(map[string]map[string][]PatternTag, len(framework.Classes))
	for className, cls := range framework.Classes {
		bySelector := make(map[string][]PatternTag, len(cls.Methods))
		for _, m := range cls.Methods {
			tags := ClassifyMethod(m, cls, framework)
			if len(tags) > 0 {
				bySelector[m.Selector] = tags
			}
		}
		if len(bySelector) > 0 {
			result[className] = bySelector
		}
	}
	return result
}

// IsDelegateProtocol reports whether a protocol exposes at least one optional method,
// which is the canonical marker of a delegate protocol in Cocoa.
func IsDelegateProtocol(proto meta.Protocol) bool {
	for _, m := range proto.Methods {
		if m.IsOptional {
			return true
		}
	}
	return false
}

// DelegateProperties returns the properties on a class whose name is "delegate"
// or "dataSource" and whose type is id<ProtocolName>.
func DelegateProperties(cls meta.Class) []DelegateProperty {
	var out []DelegateProperty
	for _, p := range cls.Properties {
		if p.Name != "delegate" && p.Name != "dataSource" {
			continue
		}
		proto := extractProtocolFromID(p.ObjCType)
		if proto == "" {
			continue
		}
		out = append(out, DelegateProperty{
			PropertyName: p.Name,
			ProtocolName: proto,
		})
	}
	return out
}

// PropertyPairs returns all non-readonly properties from a class.
// These are the candidates for PropertyPair ergonomic wrappers.
func PropertyPairs(cls meta.Class) []meta.Property {
	var out []meta.Property
	for _, p := range cls.Properties {
		if !p.IsReadOnly {
			out = append(out, p)
		}
	}
	return out
}
