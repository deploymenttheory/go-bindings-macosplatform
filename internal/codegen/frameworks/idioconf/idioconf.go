// Package idioconf loads the per-framework idiomatic.json sidecar — the
// declarative configuration surface of the idiomatic emitter. Where
// overrides.json corrects the *scanned metadata* (and therefore affects every
// pipeline), idiomatic.json shapes only the *idiomatic layer's* output:
// curated Go names for methods and C functions whose mechanical translation is
// poor, the set of protocols surfaced as Go delegate interfaces, and the
// status-code typedefs (hv_return_t, kern_return_t, …) whose returns become Go
// errors.
//
// The file lives next to the framework's committed .gometa.json
// (metadata/frameworks/<name>/idiomatic.json), is discovered the same way as
// overrides.json, and follows the same rot guard: entries that no longer match
// any declaration produce warnings at generation time so a stale sidecar can
// never fail silently after an SDK re-scan.
package idioconf

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/meta"
)

// FileName is the sidecar file name, discovered next to a .gometa.json file.
const FileName = "idiomatic.json"

// ErrInvalidConfig marks a malformed idiomatic.json entry — one that can never
// have been correct, as opposed to a stale one (which only warns).
var ErrInvalidConfig = errors.New("invalid idiomatic config entry")

// File is the parsed idiomatic.json sidecar for one framework.
//
//nolint:tagliatelle // snake_case keys match the committed sidecar convention (overrides.json).
type File struct {
	// RenameMethods assigns curated Go names to ObjC methods whose mechanical
	// selector translation reads poorly (e.g. stringWithContentsOfFile:… →
	// NewStringFromFile).
	RenameMethods []MethodRename `json:"rename_methods,omitempty"`
	// RenameFunctions assigns curated Go names to C functions.
	RenameFunctions []FunctionRename `json:"rename_functions,omitempty"`
	// Delegates force-includes or force-excludes protocols from delegate
	// interface generation, overriding the emitter's auto-detection.
	Delegates DelegateConfig `json:"delegates,omitzero"`
	// ErrorTypedefs registers status-code typedefs whose function returns the
	// idiomatic layer converts to Go errors.
	ErrorTypedefs []ErrorTypedef `json:"error_typedefs,omitempty"`
	// InOutCountParams marks specific C-function parameters as in/out element-
	// count pointers.
	InOutCountParams []InOutCountParam `json:"inout_count_params,omitempty"`
}

// InOutCountParam marks one C-function parameter as an in/out element-count
// pointer — the C "(T *buf, int *count)" idiom where *count is the buffer
// capacity on input and the number of elements processed on output (e.g.
// vmnet_read / vmnet_write). Such a parameter is threaded through the idiomatic
// wrapper as a caller-supplied *N rather than lifted into a return value: the
// default emitter treats a lone scalar pointer as a pure out-parameter and
// declares it as a zero-initialised local, which for an in/out count would tell
// the framework the buffer holds zero elements and make the call a no-op. This
// opt-in is needed only where the count is read on input; pure out counts (the
// common case) are left to the default lift.
//
//nolint:tagliatelle // snake_case keys match the committed sidecar convention.
type InOutCountParam struct {
	CName string `json:"c_name"`
	Param string `json:"param"`
}

// MethodRename maps one ObjC method to a curated Go name.
//
//nolint:tagliatelle // snake_case keys match the committed sidecar convention.
type MethodRename struct {
	Class    string `json:"class"`
	Selector string `json:"selector"`
	// IsClassMethod restricts the match to class methods (true) or instance
	// methods (false). When omitted the rename matches either kind.
	IsClassMethod *bool  `json:"class_method,omitempty"`
	GoName        string `json:"go_name"`
}

// FunctionRename maps one C function to a curated Go name.
//
//nolint:tagliatelle // snake_case keys match the committed sidecar convention.
type FunctionRename struct {
	CName  string `json:"c_name"`
	GoName string `json:"go_name"`
}

// DelegateConfig adjusts which protocols the emitter surfaces as Go delegate
// interfaces beyond its auto-detection.
type DelegateConfig struct {
	Include []string `json:"include,omitempty"`
	Exclude []string `json:"exclude,omitempty"`
}

// ErrorTypedef declares a C status-code typedef (e.g. hv_return_t) whose
// non-success return values the idiomatic layer converts to Go errors.
//
//nolint:tagliatelle // snake_case keys match the committed sidecar convention.
type ErrorTypedef struct {
	Typedef string `json:"typedef"`
	// SuccessValue is the return value that means "no error" (usually 0).
	SuccessValue int64 `json:"success_value"`
	// Domain names the error domain used for errkit sentinels and messages.
	Domain string `json:"domain"`
	// SentinelEnum optionally names the C enum whose members become exported
	// errkit sentinels (e.g. hv_error → ErrBusy, ErrDenied, …).
	SentinelEnum string `json:"sentinel_enum,omitempty"`
}

// LoadAdjacent looks for an idiomatic.json next to metaPath. A missing file is
// not an error; found reports whether one was read.
func LoadAdjacent(metaPath string) (file *File, found bool, err error) {
	path := filepath.Join(filepath.Dir(metaPath), FileName)
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("reading idiomatic config %s: %w", path, err)
	}
	var parsed File
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&parsed); err != nil {
		return nil, false, fmt.Errorf("parsing idiomatic config %s: %w", path, err)
	}
	if err := parsed.checkWellFormed(); err != nil {
		return nil, false, fmt.Errorf("invalid idiomatic config %s: %w", path, err)
	}
	return &parsed, true, nil
}

// checkWellFormed rejects entries with missing required fields. Malformed
// entries are hard errors (unlike stale ones, which only warn) because they
// can never have been correct.
func (f *File) checkWellFormed() error {
	for _, rename := range f.RenameMethods {
		if rename.Class == "" || rename.Selector == "" || rename.GoName == "" {
			return fmt.Errorf(
				"%w: rename_methods needs class, selector and go_name (got class=%q selector=%q go_name=%q)",
				ErrInvalidConfig,
				rename.Class,
				rename.Selector,
				rename.GoName,
			)
		}
		if !isExportedIdent(rename.GoName) {
			return fmt.Errorf(
				"%w: rename_methods go_name %q is not an exported Go identifier",
				ErrInvalidConfig,
				rename.GoName,
			)
		}
	}
	for _, rename := range f.RenameFunctions {
		if rename.CName == "" || rename.GoName == "" {
			return fmt.Errorf(
				"%w: rename_functions needs c_name and go_name (got c_name=%q go_name=%q)",
				ErrInvalidConfig, rename.CName, rename.GoName)
		}
		if !isExportedIdent(rename.GoName) {
			return fmt.Errorf(
				"%w: rename_functions go_name %q is not an exported Go identifier",
				ErrInvalidConfig,
				rename.GoName,
			)
		}
	}
	for _, typedef := range f.ErrorTypedefs {
		if typedef.Typedef == "" || typedef.Domain == "" {
			return fmt.Errorf(
				"%w: error_typedefs needs typedef and domain (got typedef=%q domain=%q)",
				ErrInvalidConfig, typedef.Typedef, typedef.Domain)
		}
	}
	for _, e := range f.InOutCountParams {
		if e.CName == "" || e.Param == "" {
			return fmt.Errorf(
				"%w: inout_count_params needs c_name and param (got c_name=%q param=%q)",
				ErrInvalidConfig, e.CName, e.Param)
		}
	}
	return nil
}

// Validate returns a warning for every entry that matches nothing in the
// framework's metadata — stale after an SDK re-scan, or a typo.
func Validate(file *File, framework *meta.FrameworkMeta) []string {
	if file == nil {
		return nil
	}
	var warnings []string
	stale := func(format string, args ...any) {
		warnings = append(
			warnings,
			fmt.Sprintf("%s idiomatic config: ", framework.Framework)+
				fmt.Sprintf(format, args...)+" — stale entry?",
		)
	}

	for _, rename := range file.RenameMethods {
		class, ok := framework.Classes[rename.Class]
		if !ok {
			stale("rename_methods: no class %q", rename.Class)
			continue
		}
		if !classHasSelector(class, rename.Selector, rename.IsClassMethod) {
			stale("rename_methods: no method %s on %s", rename.Selector, rename.Class)
		}
	}

	for _, rename := range file.RenameFunctions {
		if !frameworkHasFunction(framework, rename.CName) {
			stale("rename_functions: no function %q", rename.CName)
		}
	}

	for _, protocolName := range file.Delegates.Include {
		if _, ok := framework.Protocols[protocolName]; !ok {
			stale("delegates.include: no protocol %q", protocolName)
		}
	}
	for _, protocolName := range file.Delegates.Exclude {
		if _, ok := framework.Protocols[protocolName]; !ok {
			stale("delegates.exclude: no protocol %q", protocolName)
		}
	}

	for _, typedef := range file.ErrorTypedefs {
		if _, ok := framework.Typedefs[typedef.Typedef]; !ok {
			stale("error_typedefs: no typedef %q", typedef.Typedef)
		}
		if typedef.SentinelEnum != "" {
			if _, ok := framework.Enums[typedef.SentinelEnum]; !ok {
				stale("error_typedefs: no sentinel enum %q", typedef.SentinelEnum)
			}
		}
	}

	for _, e := range file.InOutCountParams {
		fn, ok := frameworkFunction(framework, e.CName)
		if !ok {
			stale("inout_count_params: no function %q", e.CName)
			continue
		}
		if !functionHasParam(fn, e.Param) {
			stale("inout_count_params: function %q has no parameter %q", e.CName, e.Param)
		}
	}

	return warnings
}

// MethodGoName returns the curated Go name for the method, if one is
// configured. Safe to call on a nil receiver.
func (f *File) MethodGoName(className, selector string, isClassMethod bool) (string, bool) {
	if f == nil {
		return "", false
	}
	for _, rename := range f.RenameMethods {
		if rename.Class != className || rename.Selector != selector {
			continue
		}
		if rename.IsClassMethod != nil && *rename.IsClassMethod != isClassMethod {
			continue
		}
		return rename.GoName, true
	}
	return "", false
}

// FunctionGoName returns the curated Go name for the C function, if one is
// configured. Safe to call on a nil receiver.
func (f *File) FunctionGoName(cName string) (string, bool) {
	if f == nil {
		return "", false
	}
	for _, rename := range f.RenameFunctions {
		if rename.CName == cName {
			return rename.GoName, true
		}
	}
	return "", false
}

// IsDelegateIncluded reports whether the protocol is force-included for
// delegate interface generation. Safe to call on a nil receiver.
func (f *File) IsDelegateIncluded(protocolName string) bool {
	return f != nil && slices.Contains(f.Delegates.Include, protocolName)
}

// IsDelegateExcluded reports whether the protocol is force-excluded from
// delegate interface generation. Safe to call on a nil receiver.
func (f *File) IsDelegateExcluded(protocolName string) bool {
	return f != nil && slices.Contains(f.Delegates.Exclude, protocolName)
}

// IsInOutCountParam reports whether the named parameter of the C function is a
// configured in/out element-count pointer. Safe to call on a nil receiver.
func (f *File) IsInOutCountParam(cName, param string) bool {
	if f == nil {
		return false
	}
	for _, e := range f.InOutCountParams {
		if e.CName == cName && e.Param == param {
			return true
		}
	}
	return false
}

// ErrorTypedefFor returns the status-code registration for the typedef, if one
// is configured. Safe to call on a nil receiver.
func (f *File) ErrorTypedefFor(typedefName string) (ErrorTypedef, bool) {
	if f == nil {
		return ErrorTypedef{}, false
	}
	for _, typedef := range f.ErrorTypedefs {
		if typedef.Typedef == typedefName {
			return typedef, true
		}
	}
	return ErrorTypedef{}, false
}

func classHasSelector(class meta.Class, selector string, isClassMethod *bool) bool {
	for _, method := range class.Methods {
		if method.Selector != selector {
			continue
		}
		if isClassMethod != nil && method.IsClassMethod != *isClassMethod {
			continue
		}
		return true
	}
	return false
}

func frameworkHasFunction(framework *meta.FrameworkMeta, name string) bool {
	_, ok := frameworkFunction(framework, name)
	return ok
}

func frameworkFunction(framework *meta.FrameworkMeta, name string) (meta.Function, bool) {
	for _, function := range framework.Functions {
		if function.Name == name {
			return function, true
		}
	}
	return meta.Function{}, false
}

func functionHasParam(function meta.Function, param string) bool {
	for _, p := range function.Params {
		if p.Name == param {
			return true
		}
	}
	return false
}

func isExportedIdent(name string) bool {
	if name == "" || name[0] < 'A' || name[0] > 'Z' {
		return false
	}
	for i := 1; i < len(name); i++ {
		c := name[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '_':
		default:
			return false
		}
	}
	return true
}
