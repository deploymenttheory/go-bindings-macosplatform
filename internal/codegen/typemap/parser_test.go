package typemap

import (
	"testing"
)

// ----------------------------------------------------------------------------
// TestNormalise
// ----------------------------------------------------------------------------

func TestNormalise(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// Nullability qualifiers
		{name: "strip _Nullable", input: "NSString * _Nullable", want: "NSString *"},
		{name: "strip _Nonnull", input: "NSString * _Nonnull", want: "NSString *"},
		{name: "strip _Null_unspecified", input: "NSString * _Null_unspecified", want: "NSString *"},
		{name: "strip __nullable", input: "NSString * __nullable", want: "NSString *"},
		{name: "strip __nonnull", input: "NSString * __nonnull", want: "NSString *"},
		{name: "strip __null_unspecified", input: "NSString * __null_unspecified", want: "NSString *"},
		{name: "nullable before star", input: "_Nullable NSString *", want: "NSString *"},

		// const / volatile / restrict qualifiers
		{name: "strip leading const", input: "const char *", want: "char *"},
		{name: "strip leading volatile", input: "volatile int", want: "int"},
		{name: "strip restrict", input: "restrict void *", want: "void *"},

		// ObjC attribute macros
		{name: "strip __strong", input: "__strong NSObject *", want: "NSObject *"},
		{name: "strip __weak", input: "__weak id", want: "id"},
		{name: "strip __unsafe_unretained", input: "__unsafe_unretained NSObject *", want: "NSObject *"},
		{name: "strip __autoreleasing", input: "__autoreleasing NSError **", want: "NSError **"},
		{name: "strip __kindof", input: "__kindof NSView *", want: "NSView *"},
		{name: "strip NS_RETURNS_RETAINED", input: "NS_RETURNS_RETAINED id", want: "id"},
		{name: "strip NS_RETURNS_NOT_RETAINED", input: "NS_RETURNS_NOT_RETAINED id", want: "id"},
		{name: "strip NS_RETURNS_INNER_POINTER", input: "NS_RETURNS_INNER_POINTER void *", want: "void *"},

		// Multiple qualifiers combined
		{name: "const + nullable", input: "const NSString * _Nullable", want: "NSString *"},
		{name: "__strong + _Nonnull", input: "__strong NSObject * _Nonnull", want: "NSObject *"},
		{name: "__kindof + __strong", input: "__kindof __strong NSView *", want: "NSView *"},

		// Whitespace normalisation
		{name: "extra internal spaces", input: "NSString   *", want: "NSString *"},
		{name: "leading whitespace", input: "  NSString *", want: "NSString *"},
		{name: "trailing whitespace", input: "NSString *  ", want: "NSString *"},
		{name: "tab-separated", input: "NSString\t*", want: "NSString *"},

		// Edge cases
		{name: "empty string", input: "", want: ""},
		{name: "void unchanged", input: "void", want: "void"},
		{name: "plain id", input: "id", want: "id"},
		{name: "already normalised", input: "NSString *", want: "NSString *"},
		{name: "double pointer", input: "NSError **", want: "NSError **"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Normalise(tc.input)
			if got != tc.want {
				t.Errorf("Normalise(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// ----------------------------------------------------------------------------
// TestIsVoid
// ----------------------------------------------------------------------------

func TestIsVoid(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "bare void", input: "void", want: true},
		{name: "void with leading space", input: " void ", want: true},
		{name: "void with const qualifier", input: "const void", want: true},
		{name: "void pointer is not void", input: "void *", want: false},
		{name: "NSObject is not void", input: "NSObject *", want: false},
		{name: "int is not void", input: "int", want: false},
		{name: "empty is not void", input: "", want: false},
		{name: "BOOL is not void", input: "BOOL", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsVoid(tc.input)
			if got != tc.want {
				t.Errorf("IsVoid(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

// ----------------------------------------------------------------------------
// TestIsPointer
// ----------------------------------------------------------------------------

func TestIsPointer(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "single pointer", input: "NSString *", want: true},
		{name: "double pointer", input: "NSError **", want: true},
		{name: "triple pointer", input: "void ***", want: true},
		{name: "pointer with qualifier", input: "NSString * _Nullable", want: true},
		{name: "void pointer", input: "void *", want: true},
		{name: "no pointer — int", input: "int", want: false},
		{name: "no pointer — BOOL", input: "BOOL", want: false},
		{name: "no pointer — id", input: "id", want: false},
		{name: "no pointer — void", input: "void", want: false},
		{name: "empty string", input: "", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsPointer(tc.input)
			if got != tc.want {
				t.Errorf("IsPointer(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

// ----------------------------------------------------------------------------
// TestIsDoublePointer
// ----------------------------------------------------------------------------

func TestIsDoublePointer(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "double pointer", input: "NSError **", want: true},
		{name: "triple pointer", input: "void ***", want: true},
		{name: "double pointer with qualifier", input: "__autoreleasing NSError **", want: true},
		{name: "double pointer nullable", input: "NSError * _Nullable *", want: true},
		{name: "single pointer", input: "NSString *", want: false},
		{name: "no pointer", input: "int", want: false},
		{name: "void", input: "void", want: false},
		{name: "empty", input: "", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsDoublePointer(tc.input)
			if got != tc.want {
				t.Errorf("IsDoublePointer(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

// ----------------------------------------------------------------------------
// TestIsBlock
// ----------------------------------------------------------------------------

func TestIsBlock(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "void block no args", input: "void (^)()", want: true},
		{name: "void block with args", input: "void (^)(NSString *, NSError *)", want: true},
		{name: "BOOL block", input: "BOOL (^)(NSString *, NSError *)", want: true},
		{name: "id block", input: "id (^)(void)", want: true},
		{name: "block with return pointer", input: "NSString * (^)(void)", want: true},
		{name: "not a block — plain pointer", input: "NSString *", want: false},
		{name: "not a block — void", input: "void", want: false},
		{name: "not a block — id", input: "id", want: false},
		{name: "not a block — function pointer", input: "void (*)(int)", want: false},
		{name: "empty", input: "", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsBlock(tc.input)
			if got != tc.want {
				t.Errorf("IsBlock(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

// ----------------------------------------------------------------------------
// TestIsInstancetype
// ----------------------------------------------------------------------------

func TestIsInstancetype(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "bare instancetype", input: "instancetype", want: true},
		{name: "instancetype with spaces", input: "  instancetype  ", want: true},
		{name: "not instancetype — NSObject", input: "NSObject *", want: false},
		{name: "not instancetype — id", input: "id", want: false},
		{name: "not instancetype — void", input: "void", want: false},
		{name: "empty", input: "", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsInstancetype(tc.input)
			if got != tc.want {
				t.Errorf("IsInstancetype(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

// ----------------------------------------------------------------------------
// TestIsID
// ----------------------------------------------------------------------------

func TestIsID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "bare id", input: "id", want: true},
		{name: "id with leading space", input: " id ", want: true},
		{name: "not id — id pointer", input: "id *", want: false},
		{name: "not id — NSObject", input: "NSObject *", want: false},
		{name: "not id — void", input: "void", want: false},
		{name: "not id — instancetype", input: "instancetype", want: false},
		{name: "empty", input: "", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsID(tc.input)
			if got != tc.want {
				t.Errorf("IsID(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

// ----------------------------------------------------------------------------
// TestIsClass
// ----------------------------------------------------------------------------

func TestIsClass(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "bare Class", input: "Class", want: true},
		{name: "Class with spaces", input: "  Class  ", want: true},
		{name: "not Class — NSObject", input: "NSObject *", want: false},
		{name: "not Class — class lowercase", input: "class", want: false},
		{name: "not Class — id", input: "id", want: false},
		{name: "empty", input: "", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsClass(tc.input)
			if got != tc.want {
				t.Errorf("IsClass(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

// ----------------------------------------------------------------------------
// TestIsSEL
// ----------------------------------------------------------------------------

func TestIsSEL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "bare SEL", input: "SEL", want: true},
		{name: "SEL with spaces", input: "  SEL  ", want: true},
		{name: "not SEL — NSSelector", input: "NSSelector *", want: false},
		{name: "not SEL — sel lowercase", input: "sel", want: false},
		{name: "not SEL — id", input: "id", want: false},
		{name: "empty", input: "", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsSEL(tc.input)
			if got != tc.want {
				t.Errorf("IsSEL(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

// ----------------------------------------------------------------------------
// TestIsBOOL
// ----------------------------------------------------------------------------

func TestIsBOOL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "bare BOOL", input: "BOOL", want: true},
		{name: "BOOL with spaces", input: "  BOOL  ", want: true},
		{name: "not BOOL — bool lowercase", input: "bool", want: false},
		{name: "not BOOL — int", input: "int", want: false},
		{name: "not BOOL — id", input: "id", want: false},
		{name: "empty", input: "", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsBOOL(tc.input)
			if got != tc.want {
				t.Errorf("IsBOOL(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

// ----------------------------------------------------------------------------
// TestClassName
// ----------------------------------------------------------------------------

func TestClassName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// Standard ObjC object pointers
		{name: "NSString pointer", input: "NSString *", want: "NSString"},
		{name: "NSObject pointer", input: "NSObject *", want: "NSObject"},
		{name: "NSArray pointer", input: "NSArray *", want: "NSArray"},

		// Qualified types — qualifier should be stripped before extraction
		{name: "nullable NSString", input: "NSString * _Nullable", want: "NSString"},
		{name: "nonnull NSView", input: "NSView * _Nonnull", want: "NSView"},
		{name: "__strong NSObject", input: "__strong NSObject *", want: "NSObject"},
		{name: "__kindof NSView", input: "__kindof NSView *", want: "NSView"},

		// Generic types — generics stripped
		{name: "NSArray with generic", input: "NSArray<NSString *> *", want: "NSArray"},
		{name: "NSDictionary with generics", input: "NSDictionary<NSString *, NSObject *> *", want: "NSDictionary"},
		{name: "NSSet with generic", input: "NSSet<ObjectType> *", want: "NSSet"},

		// Non-object / non-class types
		{name: "void pointer", input: "void *", want: ""},
		{name: "id", input: "id", want: ""},
		{name: "lowercase type", input: "int", want: ""},
		{name: "lowercase pointer", input: "char *", want: ""},
		{name: "empty string", input: "", want: ""},

		// Types with spaces after normalisation
		{name: "type with internal space", input: "unsigned long", want: ""},

		// Non-pointer but uppercase — still returns name
		{name: "SEL bare", input: "SEL", want: "SEL"},
		{name: "Class bare", input: "Class", want: "Class"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassName(tc.input)
			if got != tc.want {
				t.Errorf("ClassName(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// ----------------------------------------------------------------------------
// TestGenericParam
// ----------------------------------------------------------------------------

func TestGenericParam(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// Single generic parameter
		{name: "NSArray ObjectType", input: "NSArray<ObjectType> *", want: "ObjectType"},
		{name: "NSSet ObjectType", input: "NSSet<ObjectType> *", want: "ObjectType"},
		{name: "NSOrderedSet ObjectType", input: "NSOrderedSet<ObjectType> *", want: "ObjectType"},

		// Pointer generic parameter — trailing * stripped
		{name: "NSArray NSString pointer", input: "NSArray<NSString *> *", want: "NSString"},

		// Two generic parameters — GenericParam returns only the first param.
		{name: "NSDictionary two params", input: "NSDictionary<NSString *, NSObject *> *", want: "NSString"},

		// No generics
		{name: "NSString no generic", input: "NSString *", want: ""},
		{name: "void pointer", input: "void *", want: ""},
		{name: "id", input: "id", want: ""},
		{name: "empty", input: "", want: ""},

		// With qualifiers — qualifiers stripped before regex
		{name: "nullable NSArray with generic", input: "NSArray<ObjectType> * _Nullable", want: "ObjectType"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := GenericParam(tc.input)
			if got != tc.want {
				t.Errorf("GenericParam(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// ----------------------------------------------------------------------------
// TestGenericParams
// ----------------------------------------------------------------------------

func TestGenericParams(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{name: "single param", input: "NSArray<ObjectType> *", want: []string{"ObjectType"}},
		{name: "two params", input: "NSDictionary<NSString *, NSObject *> *", want: []string{"NSString", "NSObject"}},
		{name: "pointer param", input: "NSArray<NSString *> *", want: []string{"NSString"}},
		{name: "no generics", input: "NSString *", want: nil},
		{name: "with nullable", input: "NSArray<ObjectType> * _Nullable", want: []string{"ObjectType"}},
		{name: "empty", input: "", want: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := GenericParams(tc.input)
			if len(got) != len(tc.want) {
				t.Fatalf("GenericParams(%q) = %v, want %v", tc.input, got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("GenericParams(%q)[%d] = %q, want %q", tc.input, i, got[i], tc.want[i])
				}
			}
		})
	}
}

// ----------------------------------------------------------------------------
// TestParseBlock
// ----------------------------------------------------------------------------

func TestParseBlock(t *testing.T) {
	type result struct {
		retType string
		args    []string
		ok      bool
	}

	tests := []struct {
		name  string
		input string
		want  result
	}{
		// No-arg blocks
		{
			name:  "void block no args",
			input: "void (^)()",
			want:  result{retType: "void", args: nil, ok: true},
		},
		{
			name:  "void block explicit void arg",
			input: "void (^)(void)",
			want:  result{retType: "void", args: nil, ok: true},
		},
		{
			name:  "BOOL block no args",
			input: "BOOL (^)()",
			want:  result{retType: "BOOL", args: nil, ok: true},
		},

		// Single-arg blocks
		{
			name:  "void block NSString arg",
			input: "void (^)(NSString *)",
			want:  result{retType: "void", args: []string{"NSString *"}, ok: true},
		},
		{
			name:  "void block id arg",
			input: "void (^)(id)",
			want:  result{retType: "void", args: []string{"id"}, ok: true},
		},
		{
			name:  "void block BOOL arg",
			input: "void (^)(BOOL)",
			want:  result{retType: "void", args: []string{"BOOL"}, ok: true},
		},

		// Multi-arg blocks
		{
			name:  "BOOL block NSString NSError",
			input: "BOOL (^)(NSString *, NSError *)",
			want:  result{retType: "BOOL", args: []string{"NSString *", "NSError *"}, ok: true},
		},
		{
			name:  "void block three args",
			input: "void (^)(NSString *, NSInteger, BOOL)",
			want:  result{retType: "void", args: []string{"NSString *", "NSInteger", "BOOL"}, ok: true},
		},

		// Non-void return type
		{
			name:  "NSString return block",
			input: "NSString * (^)(void)",
			want:  result{retType: "NSString *", args: nil, ok: true},
		},
		{
			name:  "id return block with arg",
			input: "id (^)(NSString *)",
			want:  result{retType: "id", args: []string{"NSString *"}, ok: true},
		},

		// Block with qualifier embedded inside (^_Nullable): the caret regex
		// tolerates qualifiers after ^ so this correctly parses as a block.
		{
			name:  "nullable qualifier inside caret parens",
			input: "void (^_Nullable)(NSString *)",
			want:  result{retType: "void", args: []string{"NSString *"}, ok: true},
		},

		// Malformed / non-block inputs
		{
			name:  "plain pointer not a block",
			input: "NSString *",
			want:  result{retType: "", args: nil, ok: false},
		},
		{
			name:  "void not a block",
			input: "void",
			want:  result{retType: "", args: nil, ok: false},
		},
		{
			name:  "function pointer not a block",
			input: "void (*)(int)",
			want:  result{retType: "", args: nil, ok: false},
		},
		{
			name:  "empty string",
			input: "",
			want:  result{retType: "", args: nil, ok: false},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			retType, args, ok := ParseBlock(tc.input)
			if ok != tc.want.ok {
				t.Errorf("ParseBlock(%q) ok = %v, want %v", tc.input, ok, tc.want.ok)
				return
			}
			if !ok {
				return
			}
			if retType != tc.want.retType {
				t.Errorf("ParseBlock(%q) retType = %q, want %q", tc.input, retType, tc.want.retType)
			}
			if len(args) != len(tc.want.args) {
				t.Errorf("ParseBlock(%q) args = %v, want %v", tc.input, args, tc.want.args)
				return
			}
			for i, a := range args {
				if a != tc.want.args[i] {
					t.Errorf("ParseBlock(%q) args[%d] = %q, want %q", tc.input, i, a, tc.want.args[i])
				}
			}
		})
	}
}

// ----------------------------------------------------------------------------
// TestSplitArgs — exercised indirectly through ParseBlock
//
// splitArgs is unexported; the table below probes edge-cases by constructing
// block strings whose arg lists exercise the specific splitting behaviour.
// ----------------------------------------------------------------------------

func TestSplitArgsViaParseBlock(t *testing.T) {
	type result struct {
		args []string
		ok   bool
	}

	tests := []struct {
		name  string
		input string
		want  result
	}{
		// Nested angle brackets must not be split on commas inside them.
		{
			name:  "generic arg not split at inner comma",
			input: "void (^)(NSDictionary<NSString *, NSObject *> *)",
			want:  result{args: []string{"NSDictionary<NSString *, NSObject *> *"}, ok: true},
		},
		// Multiple top-level args, one of which is generic.
		{
			name:  "generic arg plus plain arg",
			input: "void (^)(NSArray<NSString *> *, BOOL)",
			want:  result{args: []string{"NSArray<NSString *> *", "BOOL"}, ok: true},
		},
		// Nested parens (e.g. a block arg inside a block) must not split on
		// commas inside them.
		{
			name:  "block arg inside block",
			input: "void (^)(void (^)(NSString *, BOOL), NSError *)",
			want:  result{args: []string{"void (^)(NSString *, BOOL)", "NSError *"}, ok: true},
		},
		// Single plain arg.
		{
			name:  "single plain arg",
			input: "void (^)(NSInteger)",
			want:  result{args: []string{"NSInteger"}, ok: true},
		},
		// No args — empty arg list.
		{
			name:  "empty arg list",
			input: "void (^)()",
			want:  result{args: nil, ok: true},
		},
		// void sentinel — treated as empty.
		{
			name:  "void sentinel arg",
			input: "void (^)(void)",
			want:  result{args: nil, ok: true},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, args, ok := ParseBlock(tc.input)
			if ok != tc.want.ok {
				t.Errorf("ParseBlock(%q) ok = %v, want %v", tc.input, ok, tc.want.ok)
				return
			}
			if !ok {
				return
			}
			if len(args) != len(tc.want.args) {
				t.Errorf("ParseBlock(%q) args = %v, want %v", tc.input, args, tc.want.args)
				return
			}
			for i, a := range args {
				if a != tc.want.args[i] {
					t.Errorf("ParseBlock(%q) args[%d] = %q, want %q", tc.input, i, a, tc.want.args[i])
				}
			}
		})
	}
}
