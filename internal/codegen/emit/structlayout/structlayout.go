// Package structlayout holds the pure, pipeline-agnostic helpers for deciding
// whether a C struct can be surfaced as a plain Go value struct that reproduces
// the C ABI, and for width-correcting struct field Go types.
//
// These functions operate only on strings, []string, and ints — they have no
// dependency on the frameworks (purego) or libraries (cgo) metadata models or
// type mappers — so both pipelines can share them. Struct emission itself stays
// pipeline-specific; only the ABI/layout reasoning lives here.
package structlayout

import (
	"strconv"
	"strings"
)

// Primitives is the set of Go primitive types a value-struct field may use
// directly. A field of any other bare type must be another emittable struct in
// the same package (checked via the emittable fixpoint in the caller).
var Primitives = map[string]bool{
	"bool": true, "byte": true, "rune": true, "uintptr": true,
	"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
	"float32": true, "float64": true,
}

// StructFieldGoType corrects a struct field's Go type to the exact C ABI width
// where the type mapper widened a native C int to Go int/uint (8 bytes) even
// though a C int is 4. Struct fields — unlike function parameters — must match the
// C width exactly, or the Go struct would mislay every field after the first int.
// Arrays are corrected element-wise (int[4] → [4]int32). Other types (short→int16,
// long→int, stdint) already match and pass through.
func StructFieldGoType(objcType, mapped string) string {
	base := objcType
	for _, q := range []string{"const", "volatile", "_Nonnull", "_Nullable", "_Null_unspecified"} {
		base = strings.ReplaceAll(base, q, " ")
	}
	if i := strings.IndexByte(base, '['); i >= 0 { // strip array dimensions
		base = base[:i]
	}
	base = strings.Join(strings.Fields(base), " ")
	var from, to string
	switch base {
	case "int", "signed int":
		from, to = "int", "int32"
	case "unsigned int", "unsigned":
		from, to = "uint", "uint32"
	default:
		return mapped
	}
	if mapped == from {
		return to
	}
	if strings.HasSuffix(mapped, "]"+from) {
		return mapped[:len(mapped)-len(from)] + to
	}
	return mapped
}

// FieldSizer resolves the size and alignment (bytes) of a non-primitive field Go
// type — an enum (by its underlying integer width) or a nested value struct (by
// its own layout) — and ok=false when goType is neither. It lets the layout
// helpers validate structs whose fields are not plain scalars. May be nil.
type FieldSizer func(goType string) (size, align int, ok bool)

// ScalarGoSizeAlign returns the size and alignment (in bytes, arm64/amd64 LP64)
// of a hermetic scalar Go type, and ok=false for anything else (a nested struct,
// pointer, slice, or unresolved type). Only fixed-width types are admitted;
// notably Go int/uint/uintptr are 8 bytes here, so a field the mapper resolved to
// a platform-width int is treated as 8 — matching the LP64 C `long`/pointer it
// came from.
func ScalarGoSizeAlign(goType string) (size, align int, ok bool) {
	return sizeAlign(goType, nil)
}

// sizeAlign is ScalarGoSizeAlign with an optional resolver for non-primitive
// (enum / nested-struct) field types, threaded through array elements so an
// array of enums or nested structs also resolves.
func sizeAlign(goType string, sizer FieldSizer) (size, align int, ok bool) {
	// A fixed-size array [N]elem lays out as N contiguous elements with the
	// element's alignment; recurse on the element (which may itself be an array).
	// A slice "[]T" or malformed dimension is not a fixed array and is rejected.
	if strings.HasPrefix(goType, "[") {
		closeIdx := strings.IndexByte(goType, ']')
		if closeIdx <= 1 {
			return 0, 0, false
		}
		n, err := strconv.Atoi(goType[1:closeIdx])
		if err != nil || n < 0 {
			return 0, 0, false
		}
		elemSize, elemAlign, elemOK := sizeAlign(goType[closeIdx+1:], sizer)
		if !elemOK {
			return 0, 0, false
		}
		return n * elemSize, elemAlign, true
	}
	switch goType {
	case "bool", "int8", "uint8", "byte":
		return 1, 1, true
	case "int16", "uint16":
		return 2, 2, true
	case "int32", "uint32", "float32", "rune":
		return 4, 4, true
	case "int64", "uint64", "float64", "int", "uint", "uintptr":
		return 8, 8, true
	}
	if sizer != nil {
		return sizer(goType)
	}
	return 0, 0, false
}

// LayoutSafeFromGoTypes reports whether a plain Go struct reproduces the C ABI
// field offsets of a struct — the condition under which reading fields through a
// pointer to the C data is correct. An unpacked struct always qualifies (Go and C
// use the same natural alignment for these scalar fields). A packed struct
// qualifies when every field is already naturally aligned at its packed offset,
// so Go inserts no inter-field padding. Trailing padding is deliberately NOT
// required: a packed struct ending on an odd size (e.g. IOUSBConfiguration
// Descriptor's 9 bytes) makes the Go value one byte larger, which is immaterial
// for pointer field access (the only way these are surfaced) though it means the
// value should not be bulk-copied out of a tightly-sized C allocation. A field
// whose Go type is not a fixed-width scalar makes the result false (conservative:
// the opaque path handles it).
func LayoutSafeFromGoTypes(packed bool, goTypes []string) bool {
	if !packed {
		return true
	}
	offset := 0
	for _, gt := range goTypes {
		sz, al, ok := ScalarGoSizeAlign(gt)
		if !ok {
			return false
		}
		if offset%al != 0 {
			return false // Go would insert padding before this field
		}
		offset += sz
	}
	return true
}

// GoStructLayout computes each field's byte offset and the total size of a Go
// struct with the given (width-corrected) field types, under Go's natural
// alignment (packed=false) or tight packing (packed=true). ok is false if any
// field's size is unknown.
func GoStructLayout(goTypes []string, packed bool, sizer FieldSizer) (offsets []int, size int, ok bool) {
	pos, maxAlign := 0, 1
	for _, gt := range goTypes {
		sz, al, fieldOK := sizeAlign(gt, sizer)
		if !fieldOK {
			return nil, 0, false
		}
		if !packed {
			if al > maxAlign {
				maxAlign = al
			}
			if r := pos % al; r != 0 {
				pos += al - r
			}
		}
		offsets = append(offsets, pos)
		pos += sz
	}
	if !packed {
		if r := pos % maxAlign; r != 0 {
			pos += maxAlign - r
		}
	}
	return offsets, pos, true
}

// LayoutMatchesAuthoritative verifies the width-corrected Go layout reproduces
// clang's authoritative record layout (per-field offsets + total size) when one
// was captured (size != 0). It returns true when no authoritative layout exists
// (nothing to cross-check — clang only lays out value-used structs) or when the
// Go layout matches exactly. A mismatch means the Go struct would not reproduce
// the C ABI, so the caller should keep the struct opaque.
//
// fieldOffsets holds the authoritative byte offset of each field (in field
// order); its length is the struct's field count.
func LayoutMatchesAuthoritative(size int, packed bool, fieldOffsets []int, goTypes []string, sizer FieldSizer) bool {
	if size == 0 {
		return true
	}
	offsets, computed, ok := GoStructLayout(goTypes, packed, sizer)
	if !ok {
		// A field the layout computer still cannot size (a nested struct the sizer
		// does not resolve, a pointer, …) — skip the cross-check rather than wrongly
		// excluding the struct; the emittable fixpoint gates field types separately.
		return true
	}
	if computed != size || len(offsets) != len(fieldOffsets) {
		return false
	}
	for i := range offsets {
		if offsets[i] != fieldOffsets[i] {
			return false
		}
	}
	return true
}

// IsPrimitiveOrArrayOf reports whether goType is a bare identifier (or a
// fixed-size array, possibly multi-dimensional, of one) whose element satisfies
// ok. A slice "[]T", a pointer, or a qualified/expression type is rejected.
func IsPrimitiveOrArrayOf(goType string, ok func(elem string) bool) bool {
	elem := goType
	for strings.HasPrefix(elem, "[") {
		closeIdx := strings.IndexByte(elem, ']')
		if closeIdx < 0 {
			return false
		}
		inner := elem[1:closeIdx]
		if inner == "" { // a slice "[]T", not a fixed array — not ABI-safe
			return false
		}
		elem = elem[closeIdx+1:]
	}
	if elem == "" || strings.ContainsAny(elem, ".*[] ") {
		return false
	}
	return ok(elem)
}
