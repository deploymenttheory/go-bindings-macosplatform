//go:build darwin

package idiomatic

import (
	"fmt"
	"io"
	"strings"
)

// writeMockInterface emits an exported <Type>able interface listing Unwrap and
// every generated method, plus a compile-time assertion that the wrapper
// implements it. Consumers accept the interface for dependency injection/mocking.
func writeMockInterface(
	w io.Writer,
	typeName string,
	withMethods []withEntry,
	methods []methodEntry,
) {
	lines := make([]string, 0, len(withMethods)+len(methods))
	for _, we := range withMethods {
		lines = append(lines, interfaceWithLine(we, typeName))
	}
	for _, me := range methods {
		lines = append(lines, interfaceMethodLine(me))
	}
	renderTemplate(w, "mock_interface", mockInterfaceView{
		IfaceName: typeName + "able",
		TypeName:  typeName,
		Lines:     lines,
	})
}

// mockInterfaceView is the template data for mock_interface.tmpl.
type mockInterfaceView struct {
	IfaceName string
	TypeName  string
	Lines     []string
}

// interfaceWithLine renders the With-setter signature for the mockable interface.
func interfaceWithLine(we withEntry, typeName string) string {
	if we.isNSArray {
		return fmt.Sprintf("%s(items ...%s) *%s", we.goName, we.sliceElemType, typeName)
	}
	return fmt.Sprintf("%s(%s %s) *%s", we.goName, we.param.goName, we.param.goType, typeName)
}

// interfaceMethodLine renders the method signature for the mockable interface,
// mirroring the writeMethod variants exactly.
func interfaceMethodLine(me methodEntry) string {
	switch me.kind {
	case kindAsync:
		parts := []string{"ctx context.Context"}
		for _, p := range me.asyncNonBlockParams {
			parts = append(parts, p.goName+" "+p.goType)
		}
		if me.asyncResultGoType != "" {
			return fmt.Sprintf(
				"%s(%s) (%s, error)",
				me.goName,
				strings.Join(parts, ", "),
				me.asyncResultGoType,
			)
		}
		return fmt.Sprintf("%s(%s) error", me.goName, strings.Join(parts, ", "))
	case kindBoolNSError:
		return fmt.Sprintf("%s() error", me.goName)
	case kindSlice:
		if me.sliceHasError {
			return fmt.Sprintf("%s() ([]%s, error)", me.goName, me.sliceElemGoType)
		}
		return fmt.Sprintf("%s() []%s", me.goName, me.sliceElemGoType)
	case kindPlain:
		parts := make([]string, 0, len(me.plainParams))
		for _, p := range me.plainParams {
			if p.isOut {
				continue // out-parameters are return values, not signature params
			}
			parts = append(parts, p.goName+" "+p.goType)
		}
		return fmt.Sprintf("%s(%s)%s", me.goName, strings.Join(parts, ", "), plainRetSig(me))
	}
	return ""
}
