package rawlib

import (
	"io"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/raw/libraries/render"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/raw/libraries/view"
)

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
	return render.Execute(w, "variadic_wrappers_file", view.VariadicWrappersFileModel{PkgName: pkgName})
}
