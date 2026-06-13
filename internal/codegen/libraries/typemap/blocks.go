package typemap

import "strings"

// IsNSError returns true for the canonical ObjC error type NSError *.
func IsNSError(qt string) bool {
	n := strings.TrimSpace(Normalise(qt))
	return n == "NSError *"
}

// GoBlockArgType returns the primitive Go type for a block argument or return type.
// Unlike GoType, this always uses primitive representations: all ObjC objects and
// pointers return "unsafe.Pointer", enums return their underlying int type.
// Used to build func types that match the generated MakeBlock_* factory signatures.
func (m *Mapper) GoBlockArgType(qt string) string {
	if qt == "" {
		return ""
	}
	n := Normalise(qt)
	n = strings.TrimSpace(n)
	if n == "" || n == "void" {
		return ""
	}
	if IsBlock(n) || IsID(n) || IsClass(n) {
		return "unsafe.Pointer"
	}
	if IsSEL(n) {
		return "string"
	}
	if IsBOOL(n) {
		return "bool"
	}
	if g := primitiveGoType(n); g != "" {
		return g
	}
	if IsPointer(n) {
		return "unsafe.Pointer"
	}
	if cfTypedefSet[n] {
		return "unsafe.Pointer"
	}
	// Named enum type → underlying Go int type (e.g. NSComparisonResult → "int64")
	if name := ClassName(n); name != "" {
		if goIntType, ok := m.EnumGoTypeIndex[name]; ok {
			return goIntType
		}
	}
	// "enum Foo" pattern from typedef chains (e.g. CVAttachmentMode → enum CVAttachmentMode)
	if strings.HasPrefix(n, "enum ") {
		if enumName := strings.TrimSpace(n[5:]); !strings.ContainsAny(enumName, " *<>^()") {
			if goIntType, ok := m.EnumGoTypeIndex[enumName]; ok {
				return goIntType
			}
		}
	}
	// C-style lowercase enum names (e.g. vmnet_return_t, interface_event_t) are
	// not detected by ClassName. Check EnumGoTypeIndex with the raw ObjC key.
	if !strings.ContainsAny(n, " *<>^()") {
		if goIntType, ok := m.EnumGoTypeIndex[n]; ok {
			return goIntType
		}
	}
	// Typedef chain resolution (e.g. IOReturn → int → "int32",
	// AUAudioFrameCount → uint32_t → "uint32"). Follow the chain until we
	// reach a recognised primitive or exhaust the known typedefs. Only
	// single-identifier types (no spaces, stars, generics, blocks) can be
	// typedef aliases; complex types fall straight to unsafe.Pointer.
	if !strings.ContainsAny(n, " *<>^()") {
		if target, ok := m.TypedefIndex[n]; ok && target != n {
			if g := m.GoBlockArgType(target); g != "" && g != "unsafe.Pointer" {
				return g
			}
		}
	}
	return "unsafe.Pointer"
}

// buildBlockGoType returns a primitive Go func type for an ObjC block type string.
// Returns "" when the block type cannot be parsed.
// All argument and return types are reduced to their primitive Go representations
// so the resulting func type matches the generated MakeBlock_* factory signatures.
func (m *Mapper) buildBlockGoType(n string, _ Context) string {
	retType, args, ok := ParseBlock(n)
	if !ok {
		return ""
	}
	var goArgs []string
	for _, a := range args {
		if t := m.GoBlockArgType(a); t != "" {
			goArgs = append(goArgs, t)
		}
	}
	goRet := m.GoBlockArgType(retType)

	var sb strings.Builder
	sb.WriteString("func(")
	sb.WriteString(strings.Join(goArgs, ", "))
	sb.WriteString(")")
	if goRet != "" {
		sb.WriteString(" ")
		sb.WriteString(goRet)
	}
	return sb.String()
}

// GoBlockUserFuncType returns the user-facing Go function type for an ObjC block
// parameter. Unlike buildBlockGoType (which uses raw unsafe.Pointer for all ObjC
// objects to match MakeBlock_* trampoline signatures), this maps:
//   - NSError * → Go error interface (idiomatic error handling)
//   - Known ObjC class pointers in args → typed Go pointer (*ClassName or *pkg.ClassName)
//   - Known ObjC class pointer return types → typed Go pointer
//
// ctx is used to resolve class names; imports collects cross-framework import paths.
// Returns "" when the block type cannot be parsed.
func (m *Mapper) GoBlockUserFuncType(n string, ctx Context, imports ImportSet) string {
	retType, args, ok := ParseBlock(n)
	if !ok {
		return ""
	}
	// Use IsReturn=false so class pointers in block args/return resolve to typed names.
	argCtx := ctx
	argCtx.IsReturn = false
	var goArgs []string
	for _, a := range args {
		var t string
		if IsNSError(a) {
			t = "error"
		} else if IsBOOLPointer(a) {
			t = "*bool"
		} else if cls := ClassName(a); cls != "" && (argCtx.ClassNameIndex[cls] || (m.StructIndex[cls] != "" && IsPointer(a))) {
			// Known ObjC class pointer or C struct pointer: resolve to typed Go name
			// via GoType so that cross-framework imports are tracked in imports.
			// Bare struct values (e.g. NSRange without *) are intentionally excluded:
			// the void* trampoline system cannot convey a 16-byte struct through a single
			// register slot — struct-value block args remain unsafe.Pointer.
			if typed := m.GoType(a, argCtx, imports); typed != "unsafe.Pointer" {
				t = typed
			}
		} else if IsID(a) {
			// bare `id` in a user-facing block arg → cgo.Object interface
			t = "cgo.Object"
		} else if protos := IDProtocols(a); len(protos) > 0 {
			// id<Protocol> block arg: use return-position semantics so GoType yields
			// *ProtoIDProtocol (constructible via NewProto(ptr)) or cgo.Object —
			// both can be built from an unsafe.Pointer by blockArgCtor.
			// Arg-position would give the bare interface, which blockArgCtor cannot construct.
			retCtx := argCtx
			retCtx.IsReturn = true
			if typed := m.GoType(a, retCtx, imports); typed != "" && typed != "unsafe.Pointer" {
				t = typed
			} else {
				t = "cgo.Object"
			}
		} else {
			// Framework-specific opaque CF types (CGColorRef, CMSampleBufferRef, etc.):
			// resolve to typed Go pointer via GoType so cross-framework imports are tracked.
			n := strings.TrimSpace(Normalise(a))
			if _, ok := m.CFTypeIndex[n]; ok {
				t = m.GoType(a, argCtx, imports)
			} else if !strings.ContainsAny(n, " *<>^()") {
				// Bare ObjC generic type parameter (e.g. ObjectType in NSArray enumerate blocks).
				// Return cgo.Object — blockArgCtor converts via cgo.WrapObject.
				if len(argCtx.GenericParams) > 0 {
					for _, gp := range argCtx.GenericParams {
						if n == gp {
							t = "cgo.Object"
							break
						}
					}
				}
				if t == "" {
					// Lowercase typedef that resolves to a known ObjC class pointer
					// (e.g. ar_anchor_t → NSObject<OS_ar_anchor> * → *foundation.NSObject).
					if target, ok := m.TypedefIndex[n]; ok {
						if targetCls := ClassName(strings.TrimSpace(Normalise(target))); targetCls != "" && argCtx.ClassNameIndex[targetCls] {
							t = m.GoType(a, argCtx, imports)
						}
					}
				}
			}
		}
		if t == "" {
			if btype := m.GoBlockArgType(a); btype != "" {
				t = btype
			}
		}
		if t != "" {
			goArgs = append(goArgs, t)
		}
	}
	// Resolve the block's return type: use GoType for known ObjC class pointers
	// so the user-facing func type is typed (e.g. func(float64) *foundation.NSString
	// rather than func(float64) unsafe.Pointer).
	goRet := m.GoBlockArgType(retType)
	if IsID(retType) {
		goRet = "cgo.Object"
	} else if protos := IDProtocols(retType); len(protos) > 0 {
		// Block return type id<P>: same return-position semantics as above.
		retCtx := argCtx
		retCtx.IsReturn = true
		if typed := m.GoType(retType, retCtx, imports); typed != "" && typed != "unsafe.Pointer" {
			goRet = typed
		} else {
			goRet = "cgo.Object"
		}
	} else if cls := ClassName(retType); cls != "" && argCtx.ClassNameIndex[cls] {
		if typed := m.GoType(retType, argCtx, imports); typed != "" && typed != "unsafe.Pointer" {
			goRet = typed
		}
	}

	var sb strings.Builder
	sb.WriteString("func(")
	sb.WriteString(strings.Join(goArgs, ", "))
	sb.WriteString(")")
	if goRet != "" {
		sb.WriteString(" ")
		sb.WriteString(goRet)
	}
	return sb.String()
}
