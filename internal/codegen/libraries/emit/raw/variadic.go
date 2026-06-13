package raw

import "io"

// EmitFoundationVariadicWrappers writes convenience variadic constructors for the
// most commonly used Foundation collection classes (NSArray, NSMutableArray,
// NSSet, NSMutableSet, NSDictionary, NSMutableDictionary). The underlying ObjC
// nil-terminated variadics (+arrayWithObjects:, +dictionaryWithObjectsAndKeys:,
// etc.) are not bridgeable via CGo; these wrappers use either:
//
//   - A pure-Go path via existing non-variadic bridged methods (NSArray, NSSet),
//   - An embedded C helper (non-variadic wrapper) per the CGo pattern for
//     functions that have no suitable non-variadic ObjC equivalent (NSDictionary).
//
// The generated file is written only for the Foundation framework.
func EmitFoundationVariadicWrappers(w io.Writer, pkgName string) error {
	return executeTemplate(w, "variadic_wrappers_file", variadicWrappersFileModel{PkgName: pkgName})
}
