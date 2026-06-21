//go:build darwin

package idiomatic

import (
	"bytes"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/emit"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/meta"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/naming"
)

// emitErrorSentinels writes <pkgname>_errors_generated.go: a named error value
// for each member of the framework's error-code enum, so callers can match a
// returned error with errors.Is. An error-code enum is one whose Go type name
// ends in "ErrorCode" (e.g. VZErrorCode); the matching error domain is the
// same name with "Code" replaced by "Domain" (VZErrorDomain), which is the
// string the framework reports in its NSError objects.
//
// For example, member VZErrorInternalError = 2 becomes:
//
//	var ErrInternalError = errkit.New("VZErrorDomain", 2)
//
// allowing errors.Is(err, virtualization.ErrInternalError).
func emitErrorSentinels(
	outDir, pkgName string,
	fw *meta.FrameworkMeta,
	takenNames map[string]bool,
) error {
	keys := make([]string, 0, len(fw.Enums))
	for key := range fw.Enums {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var body bytes.Buffer
	for _, key := range keys {
		e := fw.Enums[key]
		if e.Availability.IsUnavailable || e.IsAnon {
			continue
		}
		goType := naming.GoTypeName(key)
		if !strings.HasSuffix(goType, "ErrorCode") {
			continue
		}
		domain := strings.TrimSuffix(goType, "Code") + "Domain"   // VZErrorCode -> VZErrorDomain
		memberPrefix := strings.TrimSuffix(goType, "Code")        // VZError

		seenValue := map[string]bool{}
		for _, mem := range e.Members {
			if mem.Availability.IsUnavailable || seenValue[mem.Value] {
				continue
			}
			seenValue[mem.Value] = true
			memberName := naming.GoTypeName(mem.Name)
			short := strings.TrimPrefix(memberName, memberPrefix)
			if short == "" || short[0] < 'A' || short[0] > 'Z' {
				short = memberName
			}
			sentinel := "Err" + short
			if takenNames[sentinel] {
				continue
			}
			takenNames[sentinel] = true
			fmt.Fprintf(&body, "// %s matches the %s error %s.\n", sentinel, fw.Framework, mem.Name)
			fmt.Fprintf(&body, "var %s = errkit.New(%q, %s)\n\n", sentinel, domain, mem.Value)
		}
	}

	if body.Len() == 0 {
		return nil
	}
	imports := map[string]string{"errkit": errkitImportPath}
	fname := pkgName + "_errors_generated.go"
	return emit.WriteGoFile(filepath.Join(outDir, fname), assembleFile(pkgName, imports, body.Bytes()))
}
