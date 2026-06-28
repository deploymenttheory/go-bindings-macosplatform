//go:build darwin

package idiomatic

import (
	"bytes"
	"fmt"
	"path/filepath"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/emit"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/meta"
)

// frameworkDylibPath returns the macOS file path of the shared library that a
// generated package must load. A plain library (one with a LinkLib name) lives
// under /usr/lib; a normal framework lives under /System/Library/Frameworks,
// using its umbrella (parent) framework's name when it has one.
func frameworkDylibPath(framework *meta.FrameworkMeta) string {
	if framework.LinkLib != "" {
		return "/usr/lib/lib" + framework.LinkLib + ".dylib"
	}
	parent := framework.Framework
	if framework.ParentFramework != "" {
		parent = framework.ParentFramework
	}
	return fmt.Sprintf("/System/Library/Frameworks/%s.framework/%s", parent, parent)
}

// emitRuntimeBootstrap writes the <pkg>_runtime_generated.go file for one
// package: the code that loads the package's macOS framework and looks up
// Objective-C classes by name, which the package's constructors and
// class-level functions rely on.
func emitRuntimeBootstrap(outDir, pkgName string, framework *meta.FrameworkMeta) error {
	var buf bytes.Buffer
	renderTemplate(&buf, "runtime", struct {
		PkgName   string
		DylibPath string
	}{
		PkgName:   pkgName,
		DylibPath: frameworkDylibPath(framework),
	})
	return emit.WriteGoFile(filepath.Join(outDir, pkgName+"_runtime_generated.go"), buf.Bytes())
}
