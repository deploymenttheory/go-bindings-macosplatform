package docc

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestApplyPatch(t *testing.T) {
	tests := []struct {
		name string
		doc  string
		ops  []PatchOp
		want string
	}{
		{
			name: "replace nested map value",
			doc:  `{"a":{"b":"old"}}`,
			ops:  []PatchOp{{Op: "replace", Path: "/a/b", Value: "new"}},
			want: `{"a":{"b":"new"}}`,
		},
		{
			name: "replace array element",
			doc:  `{"xs":["a","b","c"]}`,
			ops:  []PatchOp{{Op: "replace", Path: "/xs/1", Value: "B"}},
			want: `{"xs":["a","B","c"]}`,
		},
		{
			name: "add map key",
			doc:  `{"a":{}}`,
			ops:  []PatchOp{{Op: "add", Path: "/a/b", Value: float64(1)}},
			want: `{"a":{"b":1}}`,
		},
		{
			name: "append to array with -",
			doc:  `{"xs":["a"]}`,
			ops:  []PatchOp{{Op: "add", Path: "/xs/-", Value: "b"}},
			want: `{"xs":["a","b"]}`,
		},
		{
			name: "remove map key",
			doc:  `{"a":1,"b":2}`,
			ops:  []PatchOp{{Op: "remove", Path: "/b"}},
			want: `{"a":1}`,
		},
		{
			name: "json pointer escapes ~1 (slash) and ~0 (tilde)",
			doc:  `{"a/b":{"c~d":"old"}}`,
			ops:  []PatchOp{{Op: "replace", Path: "/a~1b/c~0d", Value: "new"}},
			want: `{"a/b":{"c~d":"new"}}`,
		},
		{
			name: "unknown op is ignored",
			doc:  `{"a":1}`,
			ops:  []PatchOp{{Op: "test", Path: "/a", Value: float64(1)}},
			want: `{"a":1}`,
		},
		{
			name: "multiple ops applied in order",
			doc:  `{"a":1}`,
			ops: []PatchOp{
				{Op: "add", Path: "/b", Value: float64(2)},
				{Op: "replace", Path: "/a", Value: float64(9)},
			},
			want: `{"a":9,"b":2}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var doc any
			if err := json.Unmarshal([]byte(tt.doc), &doc); err != nil {
				t.Fatalf("unmarshal doc: %v", err)
			}
			got, err := ApplyPatch(doc, tt.ops)
			if err != nil {
				t.Fatalf("ApplyPatch: %v", err)
			}
			var want any
			if err := json.Unmarshal([]byte(tt.want), &want); err != nil {
				t.Fatalf("unmarshal want: %v", err)
			}
			if !reflect.DeepEqual(got, want) {
				gotJSON, _ := json.Marshal(got)
				t.Errorf("ApplyPatch = %s, want %s", gotJSON, tt.want)
			}
		})
	}
}

func TestApplyPatchErrors(t *testing.T) {
	var doc any
	_ = json.Unmarshal([]byte(`{"xs":["a"]}`), &doc)

	for _, tt := range []struct {
		name string
		op   PatchOp
	}{
		{"array index out of range", PatchOp{Op: "replace", Path: "/xs/5", Value: "x"}},
		{"non-numeric array index", PatchOp{Op: "replace", Path: "/xs/foo", Value: "x"}},
		{"pointer without leading slash", PatchOp{Op: "replace", Path: "xs", Value: "x"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ApplyPatch(doc, []PatchOp{tt.op}); err == nil {
				t.Errorf("expected error for %s, got nil", tt.name)
			}
		})
	}
}
