package scanner

import "testing"

func TestParseRecordLayouts(t *testing.T) {
	// -fdump-record-layouts-simple shape: sizes/offsets in BITS, interleaved IRgen
	// blocks (ignored), a struct tag, a typedef name, a bitfield record (non-byte
	// offset — dropped), and an anonymous record (dropped).
	dump := `*** Dumping AST Record Layout
Type: struct Foo
Layout: <ASTRecordLayout
  Size:64
  DataSize:64
  Alignment:16
  FieldOffsets: [0, 16, 32]>
*** Dumping IRgen Record Layout
Record: RecordDecl struct Foo definition
Layout: <CGRecordLayout
  LLVMType:%struct.Foo = type { i8, i16, i8 }>
*** Dumping AST Record Layout
Type: TdAnon
Layout: <ASTRecordLayout
  Size:32
  FieldOffsets: [0, 16]>
*** Dumping AST Record Layout
Type: struct Bits
Layout: <ASTRecordLayout
  Size:8
  FieldOffsets: [0, 1]>
*** Dumping AST Record Layout
Type: struct (unnamed at foo.h:1:1)
Layout: <ASTRecordLayout
  Size:8
  FieldOffsets: [0]>
`
	got := parseRecordLayouts(dump)

	if len(got) != 2 {
		t.Fatalf("parsed %d records; want 2 (Foo, TdAnon): %v", len(got), got)
	}
	if foo := got["Foo"]; foo.Size != 8 || !intsEqualRL(foo.FieldOffsets, []int{0, 2, 4}) {
		t.Errorf("Foo = %+v; want size 8 offsets [0 2 4]", foo)
	}
	if td := got["TdAnon"]; td.Size != 4 || !intsEqualRL(td.FieldOffsets, []int{0, 2}) {
		t.Errorf("TdAnon = %+v; want size 4 offsets [0 2]", td)
	}
	if _, ok := got["Bits"]; ok {
		t.Error("Bits (bitfield, offset 1 bit) should be dropped")
	}
}

func intsEqualRL(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
