//go:build darwin

package idiofw

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/raw/frameworks"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/idiomatic/frameworks/render"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/idiomatic/frameworks/view"
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
	framework *meta.FrameworkMeta,
	takenNames map[string]bool,
) error {
	keys := make([]string, 0, len(framework.Enums))
	for key := range framework.Enums {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var sentinels []view.ErrorSentinel
	for _, key := range keys {
		enum := framework.Enums[key]
		if enum.Availability.IsUnavailable || enum.IsAnon {
			continue
		}
		goType := naming.GoTypeName(key)
		if !strings.HasSuffix(goType, "ErrorCode") {
			continue
		}
		domain := strings.TrimSuffix(goType, "Code") + "Domain" // VZErrorCode -> VZErrorDomain
		memberPrefix := strings.TrimSuffix(goType, "Code")      // VZError

		seenValue := map[string]bool{}
		for _, mem := range enum.Members {
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
			sentinels = append(sentinels, view.ErrorSentinel{
				GoName: sentinel,
				CommentBlock: fmt.Sprintf(
					"// %s matches the %s error %s.\n",
					sentinel,
					framework.Framework,
					mem.Name,
				),
				Domain: domain,
				Code:   mem.Value,
			})
		}
	}

	if len(sentinels) == 0 {
		return nil
	}
	body, err := render.Sentinels(sentinels)
	if err != nil {
		return err
	}
	imports := map[string]string{"errkit": errkitImportPath}
	fname := pkgName + "_errors_generated.go"
	return rawfw.WriteGoFile(filepath.Join(outDir, fname), assembleFile(pkgName, imports, body))
}
