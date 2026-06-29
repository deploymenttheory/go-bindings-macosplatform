//go:build darwin

package idiomatic

import (
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/shared/fileasm"
)

// assembleFile renders a complete generated file — the scaffold (header, build
// tag, package clause, import block) plus the already-rendered body — through
// the shared fileasm assembler. Imports are grouped standard-library first, then
// everything else (the same grouping goimports produces).
func assembleFile(pkgName string, imports map[string]string, body []byte) []byte {
	return fileasm.Assemble(fileasm.File{
		Header:      strings.TrimRight(generatedHeader, "\n"),
		BuildTag:    strings.TrimRight(buildTag, "\n"),
		PkgName:     pkgName,
		ImportLines: fileasm.ImportLinesStdlibMod(imports),
		Body:        string(body),
	})
}
