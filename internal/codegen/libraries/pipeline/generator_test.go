package pipeline

import (
	"sort"
	"testing"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
)

// ─── detectImportCycle ──────────────────────────────────────────────────────────

func TestFindImportCycleNil(t *testing.T) {
	// Acyclic graph: A→B, B→C — no cycle.
	adj := map[string][]string{
		"A": {"B"},
		"B": {"C"},
		"C": {},
	}
	if got := detectImportCycle(adj); got != nil {
		t.Errorf("expected nil for acyclic graph, got %v", got)
	}
}

func TestFindImportCycleSelfLoop(t *testing.T) {
	// A→A is a self-loop.
	adj := map[string][]string{
		"A": {"A"},
	}
	cycle := detectImportCycle(adj)
	if cycle == nil {
		t.Fatal("expected a cycle for self-loop A→A")
	}
	found := false
	for _, n := range cycle {
		if n == "A" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'A' in cycle %v", cycle)
	}
}

func TestFindImportCycleSimple(t *testing.T) {
	// A→B→A — simple two-node cycle.
	adj := map[string][]string{
		"A": {"B"},
		"B": {"A"},
	}
	cycle := detectImportCycle(adj)
	if cycle == nil {
		t.Fatal("expected a cycle for A→B→A")
	}
	has := func(n string) bool {
		for _, x := range cycle {
			if x == n {
				return true
			}
		}
		return false
	}
	if !has("A") || !has("B") {
		t.Errorf("expected both A and B in cycle %v", cycle)
	}
}

func TestFindImportCycleTriangle(t *testing.T) {
	// A→B→C→A — three-node cycle.
	adj := map[string][]string{
		"A": {"B"},
		"B": {"C"},
		"C": {"A"},
	}
	cycle := detectImportCycle(adj)
	if cycle == nil {
		t.Fatal("expected a cycle for A→B→C→A")
	}
	if len(cycle) < 3 {
		t.Errorf("expected cycle of length >= 3, got %v", cycle)
	}
}

func TestFindImportCycleDisconnected(t *testing.T) {
	// Two disconnected chains, no cycles.
	adj := map[string][]string{
		"A": {"B"},
		"B": {},
		"C": {"D"},
		"D": {},
	}
	if got := detectImportCycle(adj); got != nil {
		t.Errorf("expected nil for disconnected acyclic graph, got %v", got)
	}
}

func TestFindImportCycleOneCycle(t *testing.T) {
	// Larger graph with one cycle: X→Y→Z→Y.
	adj := map[string][]string{
		"X": {"Y"},
		"Y": {"Z"},
		"Z": {"Y"},
		"W": {"X"},
	}
	cycle := detectImportCycle(adj)
	if cycle == nil {
		t.Fatal("expected a cycle in graph with Z→Y back-edge")
	}
	has := func(n string) bool {
		for _, x := range cycle {
			if x == n {
				return true
			}
		}
		return false
	}
	if !has("Y") || !has("Z") {
		t.Errorf("expected Y and Z in cycle %v", cycle)
	}
}

// ─── resolveBlockedImports ────────────────────────────────────────────────────

func makeReg(frameworks []*macosplatformmetadata.FrameworkMeta, owner map[string]string) *Registry {
	kc := make(map[string]bool)
	for cls := range owner {
		kc[cls] = true
	}
	return &Registry{
		Frameworks:   frameworks,
		ClassNameIndex: kc,
		OwnerIndex: owner,
		GenericClasses: map[string]bool{},
		ClassIndex:     map[string]macosplatformmetadata.Class{},
	}
}

func TestComputeBlockedImportsEmpty(t *testing.T) {
	reg := makeReg(nil, map[string]string{})
	blocked := resolveBlockedImports(reg)
	if len(blocked) != 0 {
		t.Errorf("expected empty blocked map for empty registry, got %v", blocked)
	}
}

func TestComputeBlockedImportsNoCycle(t *testing.T) {
	// A references B, B does not reference A → no cycle, nothing blocked.
	fmA := &macosplatformmetadata.FrameworkMeta{
		Framework: "FrameworkA",
		Classes: map[string]macosplatformmetadata.Class{
			"AClass": {
				Methods: []macosplatformmetadata.Method{{
					Selector: "getB",
					Return:   macosplatformmetadata.ReturnType{ObjCType: "BClass *"},
				}},
			},
		},
	}
	fmB := &macosplatformmetadata.FrameworkMeta{
		Framework: "FrameworkB",
		Classes: map[string]macosplatformmetadata.Class{
			"BClass": {},
		},
	}
	reg := makeReg(
		[]*macosplatformmetadata.FrameworkMeta{fmA, fmB},
		map[string]string{"BClass": "FrameworkB"},
	)
	blocked := resolveBlockedImports(reg)
	if len(blocked) != 0 {
		t.Errorf("expected no blocked imports for acyclic graph, got %v", blocked)
	}
}

func TestComputeBlockedImportsSingleCycle(t *testing.T) {
	// A references a class owned by B; B references a class owned by A.
	// One of these edges must be blocked.
	fmA := &macosplatformmetadata.FrameworkMeta{
		Framework: "FrameworkA",
		Classes: map[string]macosplatformmetadata.Class{
			"AClass": {
				Methods: []macosplatformmetadata.Method{{
					Selector: "getB",
					Return:   macosplatformmetadata.ReturnType{ObjCType: "BClass *"},
				}},
			},
		},
	}
	fmB := &macosplatformmetadata.FrameworkMeta{
		Framework: "FrameworkB",
		Classes: map[string]macosplatformmetadata.Class{
			"BClass": {
				Methods: []macosplatformmetadata.Method{{
					Selector: "getA",
					Return:   macosplatformmetadata.ReturnType{ObjCType: "AClass *"},
				}},
			},
		},
	}
	reg := makeReg(
		[]*macosplatformmetadata.FrameworkMeta{fmA, fmB},
		map[string]string{
			"AClass": "FrameworkA",
			"BClass": "FrameworkB",
		},
	)
	blocked := resolveBlockedImports(reg)

	// After breaking the cycle exactly one directed edge should be blocked.
	total := 0
	for _, targets := range blocked {
		total += len(targets)
	}
	if total == 0 {
		t.Errorf("expected at least one blocked import for cyclic graph, got %v", blocked)
	}
}

func TestComputeBlockedImportsMinWeight(t *testing.T) {
	// FrameworkA references BClass twice; FrameworkB references AClass once.
	// The lighter edge (B→A, weight 1) should be blocked, not the heavier (A→B, weight 2).
	fmA := &macosplatformmetadata.FrameworkMeta{
		Framework: "FrameworkA",
		Classes: map[string]macosplatformmetadata.Class{
			"AClass": {
				Methods: []macosplatformmetadata.Method{
					{Selector: "getB1", Return: macosplatformmetadata.ReturnType{ObjCType: "BClass *"}},
					{Selector: "getB2", Return: macosplatformmetadata.ReturnType{ObjCType: "BClass *"}},
				},
			},
		},
	}
	fmB := &macosplatformmetadata.FrameworkMeta{
		Framework: "FrameworkB",
		Classes: map[string]macosplatformmetadata.Class{
			"BClass": {
				Methods: []macosplatformmetadata.Method{{
					Selector: "getA",
					Return:   macosplatformmetadata.ReturnType{ObjCType: "AClass *"},
				}},
			},
		},
	}
	reg := makeReg(
		[]*macosplatformmetadata.FrameworkMeta{fmA, fmB},
		map[string]string{
			"AClass": "FrameworkA",
			"BClass": "FrameworkB",
		},
	)
	blocked := resolveBlockedImports(reg)

	// The minimum-weight edge is B→A (weight 1), so FrameworkB→FrameworkA is blocked.
	if !blocked["FrameworkB"]["FrameworkA"] {
		t.Errorf("expected FrameworkB→FrameworkA to be blocked (lighter edge), got %v", blocked)
	}
	if blocked["FrameworkA"]["FrameworkB"] {
		t.Errorf("expected FrameworkA→FrameworkB NOT to be blocked (heavier edge), got %v", blocked)
	}
}

func TestComputeBlockedImportsForeignExtensions(t *testing.T) {
	// ForeignExtensions entries add weight to the would-import graph.
	// FrameworkA extends class BClass (owned by FrameworkB) via ForeignExtensions.
	// FrameworkB also references AClass (owned by FrameworkA) in its methods.
	// One edge must be blocked.
	fmA := &macosplatformmetadata.FrameworkMeta{
		Framework: "FrameworkA",
		Classes:   map[string]macosplatformmetadata.Class{"AClass": {}},
		ForeignExtensions: map[string][]macosplatformmetadata.Method{
			"BClass": {voidInstanceMethod("ext")},
		},
	}
	fmB := &macosplatformmetadata.FrameworkMeta{
		Framework: "FrameworkB",
		Classes: map[string]macosplatformmetadata.Class{
			"BClass": {
				Methods: []macosplatformmetadata.Method{{
					Selector: "getA",
					Return:   macosplatformmetadata.ReturnType{ObjCType: "AClass *"},
				}},
			},
		},
	}
	reg := makeReg(
		[]*macosplatformmetadata.FrameworkMeta{fmA, fmB},
		map[string]string{
			"AClass": "FrameworkA",
			"BClass": "FrameworkB",
		},
	)
	blocked := resolveBlockedImports(reg)
	total := 0
	for _, targets := range blocked {
		total += len(targets)
	}
	if total == 0 {
		t.Errorf("expected at least one blocked import with ForeignExtensions cycle, got %v", blocked)
	}
}

func TestComputeBlockedImportsEnumCycle(t *testing.T) {
	// FrameworkA references an enum type EnumB owned by FrameworkB (4 times).
	// FrameworkB has 6 protocol-Implements edges pointing to a protocol owned by FrameworkA.
	// This mirrors the real Foundation→Security (SSLProtocol) cycle.
	//
	// Expected outcome: FrameworkA→FrameworkB is blocked (it's the lighter method-weight edge,
	// weight 4 vs 6), leaving FrameworkB able to embed FrameworkA's protocol.
	// (Protocol-only edges have lower priority to break than method-weight edges when
	// the method-weight edge is actually lighter, but here we verify the enum is counted
	// so the cycle is detected at all.)
	fmA := &macosplatformmetadata.FrameworkMeta{
		Framework: "FrameworkA",
		Classes: map[string]macosplatformmetadata.Class{
			"AClass": {
				Methods: []macosplatformmetadata.Method{
					{Selector: "m1", Return: macosplatformmetadata.ReturnType{ObjCType: "EnumB"}},
					{Selector: "m2", Return: macosplatformmetadata.ReturnType{ObjCType: "EnumB"}},
					{Selector: "m3", Return: macosplatformmetadata.ReturnType{ObjCType: "EnumB"}},
					{Selector: "m4", Return: macosplatformmetadata.ReturnType{ObjCType: "EnumB"}},
				},
			},
		},
	}
	fmB := &macosplatformmetadata.FrameworkMeta{
		Framework: "FrameworkB",
		Protocols: map[string]macosplatformmetadata.Protocol{
			"PB1": {InheritedProtocols: []string{"ProtoA"}},
			"PB2": {InheritedProtocols: []string{"ProtoA"}},
			"PB3": {InheritedProtocols: []string{"ProtoA"}},
			"PB4": {InheritedProtocols: []string{"ProtoA"}},
			"PB5": {InheritedProtocols: []string{"ProtoA"}},
			"PB6": {InheritedProtocols: []string{"ProtoA"}},
		},
	}
	reg := &Registry{
		Frameworks:     []*macosplatformmetadata.FrameworkMeta{fmA, fmB},
		ClassNameIndex:   map[string]bool{"AClass": true},
		OwnerIndex: map[string]string{"AClass": "FrameworkA"},
		GenericClasses: map[string]bool{},
		ClassIndex:     map[string]macosplatformmetadata.Class{},
		EnumIndex:     map[string]string{"EnumB": "FrameworkB"},
		ProtocolIndex: map[string]string{"ProtoA": "FrameworkA"},
	}
	blocked := resolveBlockedImports(reg)

	// The cycle must be detected and exactly one direction blocked.
	aBlocksB := blocked["FrameworkA"]["FrameworkB"]
	bBlocksA := blocked["FrameworkB"]["FrameworkA"]
	if !aBlocksB && !bBlocksA {
		t.Fatalf("no edge was blocked; cycle was not detected (blocked=%v)", blocked)
	}
	if aBlocksB && bBlocksA {
		t.Fatalf("both edges blocked; over-suppression (blocked=%v)", blocked)
	}
}

func TestComputeBlockedImportsProtocolEmbedPriority(t *testing.T) {
	// FrameworkA→FrameworkB: 5 method-type references (high cost to suppress).
	// FrameworkB→FrameworkA: 3 protocol-Implements references only (low cost to suppress).
	// Even though FrameworkB→FrameworkA has fewer total references, since it's
	// protocol-embed-only it should be preferred for blocking over the method-type edge.
	fmA := &macosplatformmetadata.FrameworkMeta{
		Framework: "FrameworkA",
		Classes: map[string]macosplatformmetadata.Class{
			"AClass": {
				Methods: []macosplatformmetadata.Method{
					{Selector: "m1", Return: macosplatformmetadata.ReturnType{ObjCType: "BClass *"}},
					{Selector: "m2", Return: macosplatformmetadata.ReturnType{ObjCType: "BClass *"}},
					{Selector: "m3", Return: macosplatformmetadata.ReturnType{ObjCType: "BClass *"}},
					{Selector: "m4", Return: macosplatformmetadata.ReturnType{ObjCType: "BClass *"}},
					{Selector: "m5", Return: macosplatformmetadata.ReturnType{ObjCType: "BClass *"}},
				},
			},
		},
	}
	fmB := &macosplatformmetadata.FrameworkMeta{
		Framework: "FrameworkB",
		Classes:   map[string]macosplatformmetadata.Class{"BClass": {}},
		Protocols: map[string]macosplatformmetadata.Protocol{
			"PB1": {InheritedProtocols: []string{"ProtoA"}},
			"PB2": {InheritedProtocols: []string{"ProtoA"}},
			"PB3": {InheritedProtocols: []string{"ProtoA"}},
		},
	}
	reg := &Registry{
		Frameworks:     []*macosplatformmetadata.FrameworkMeta{fmA, fmB},
		ClassNameIndex:   map[string]bool{"AClass": true, "BClass": true},
		OwnerIndex: map[string]string{"AClass": "FrameworkA", "BClass": "FrameworkB"},
		GenericClasses: map[string]bool{},
		ClassIndex:     map[string]macosplatformmetadata.Class{},
		EnumIndex:     map[string]string{},
		ProtocolIndex: map[string]string{"ProtoA": "FrameworkA"},
	}
	blocked := resolveBlockedImports(reg)

	// FrameworkB→FrameworkA is protocol-embed-only, so it should be blocked first
	// even though FrameworkA→FrameworkB has more total references.
	if !blocked["FrameworkB"]["FrameworkA"] {
		t.Errorf("expected FrameworkB→FrameworkA to be blocked (protocol-embed-only edge), got %v", blocked)
	}
	if blocked["FrameworkA"]["FrameworkB"] {
		t.Errorf("expected FrameworkA→FrameworkB NOT to be blocked (method-type edge), got %v", blocked)
	}
}

// voidInstanceMethod mirrors the helper in the emit test for convenience here.
func voidInstanceMethod(selector string) macosplatformmetadata.Method {
	return macosplatformmetadata.Method{
		Selector:      selector,
		IsClassMethod: false,
		Return:        macosplatformmetadata.ReturnType{ObjCType: "void"},
	}
}

// ─── sortFrameworksByDependency ───────────────────────────────────────────────────────

func TestTopoSortNoDeps(t *testing.T) {
	// Three independent frameworks — all should appear in the output.
	fmA := &macosplatformmetadata.FrameworkMeta{Framework: "Alpha", Classes: map[string]macosplatformmetadata.Class{"AClass": {}}}
	fmB := &macosplatformmetadata.FrameworkMeta{Framework: "Beta", Classes: map[string]macosplatformmetadata.Class{"BClass": {}}}
	fmC := &macosplatformmetadata.FrameworkMeta{Framework: "Gamma", Classes: map[string]macosplatformmetadata.Class{"CClass": {}}}
	reg := &Registry{
		Frameworks:     []*macosplatformmetadata.FrameworkMeta{fmA, fmB, fmC},
		OwnerIndex: map[string]string{"AClass": "Alpha", "BClass": "Beta", "CClass": "Gamma"},
		ClassNameIndex:   map[string]bool{"AClass": true, "BClass": true, "CClass": true},
		GenericClasses: map[string]bool{},
		ClassIndex:     map[string]macosplatformmetadata.Class{},
	}
	sorted := sortFrameworksByDependency(reg)
	if len(sorted) != 3 {
		t.Fatalf("expected 3 frameworks, got %d", len(sorted))
	}
	names := make([]string, len(sorted))
	for i, framework := range sorted {
		names[i] = framework.Framework
	}
	sort.Strings(names)
	want := []string{"Alpha", "Beta", "Gamma"}
	for i, w := range want {
		if names[i] != w {
			t.Errorf("missing framework %q in sorted output %v", w, names)
		}
	}
}

func TestTopoSortLinearChain(t *testing.T) {
	// AClass inherits BClass; B should come before A.
	fmA := &macosplatformmetadata.FrameworkMeta{
		Framework: "FrameworkA",
		Classes: map[string]macosplatformmetadata.Class{
			"AClass": {Super: "BClass"},
		},
	}
	fmB := &macosplatformmetadata.FrameworkMeta{
		Framework: "FrameworkB",
		Classes:   map[string]macosplatformmetadata.Class{"BClass": {}},
	}
	reg := &Registry{
		Frameworks:     []*macosplatformmetadata.FrameworkMeta{fmA, fmB},
		OwnerIndex: map[string]string{"AClass": "FrameworkA", "BClass": "FrameworkB"},
		ClassNameIndex:   map[string]bool{"AClass": true, "BClass": true},
		GenericClasses: map[string]bool{},
		ClassIndex:     map[string]macosplatformmetadata.Class{},
	}
	sorted := sortFrameworksByDependency(reg)
	if len(sorted) != 2 {
		t.Fatalf("expected 2 frameworks, got %d", len(sorted))
	}

	pos := make(map[string]int)
	for i, framework := range sorted {
		pos[framework.Framework] = i
	}
	if pos["FrameworkB"] >= pos["FrameworkA"] {
		t.Errorf("FrameworkB should come before FrameworkA; got order %v", sorted)
	}
}

func TestTopoSortDiamond(t *testing.T) {
	// Diamond: CClass extends AClass and BClass;
	// FrameworkC depends on both FrameworkA and FrameworkB → C must be last.
	fmA := &macosplatformmetadata.FrameworkMeta{
		Framework: "FrameworkA",
		Classes:   map[string]macosplatformmetadata.Class{"AClass": {}},
	}
	fmB := &macosplatformmetadata.FrameworkMeta{
		Framework: "FrameworkB",
		Classes:   map[string]macosplatformmetadata.Class{"BClass": {}},
	}
	fmC := &macosplatformmetadata.FrameworkMeta{
		Framework: "FrameworkC",
		Classes: map[string]macosplatformmetadata.Class{
			// Super records only one parent; use two classes to create two deps.
			"CClass":  {Super: "AClass"},
			"CClass2": {Super: "BClass"},
		},
	}
	reg := &Registry{
		Frameworks: []*macosplatformmetadata.FrameworkMeta{fmA, fmB, fmC},
		OwnerIndex: map[string]string{
			"AClass":  "FrameworkA",
			"BClass":  "FrameworkB",
			"CClass":  "FrameworkC",
			"CClass2": "FrameworkC",
		},
		ClassNameIndex:   map[string]bool{"AClass": true, "BClass": true, "CClass": true, "CClass2": true},
		GenericClasses: map[string]bool{},
		ClassIndex:     map[string]macosplatformmetadata.Class{},
	}
	sorted := sortFrameworksByDependency(reg)
	if len(sorted) != 3 {
		t.Fatalf("expected 3 frameworks, got %d", len(sorted))
	}
	pos := make(map[string]int)
	for i, framework := range sorted {
		pos[framework.Framework] = i
	}
	if pos["FrameworkC"] <= pos["FrameworkA"] || pos["FrameworkC"] <= pos["FrameworkB"] {
		t.Errorf("FrameworkC must come after both A and B; positions: %v", pos)
	}
}

func TestTopoSortIncludesAll(t *testing.T) {
	// Every framework in the input must appear exactly once in the output.
	frameworks := make([]*macosplatformmetadata.FrameworkMeta, 0, 5)
	owner := map[string]string{}
	kc := map[string]bool{}
	for _, name := range []string{"Alpha", "Beta", "Gamma", "Delta", "Epsilon"} {
		cls := name + "Root"
		frameworks = append(frameworks, &macosplatformmetadata.FrameworkMeta{
			Framework: name,
			Classes:   map[string]macosplatformmetadata.Class{cls: {}},
		})
		owner[cls] = name
		kc[cls] = true
	}
	reg := &Registry{
		Frameworks:     frameworks,
		OwnerIndex: owner,
		ClassNameIndex:   kc,
		GenericClasses: map[string]bool{},
		ClassIndex:     map[string]macosplatformmetadata.Class{},
	}
	sorted := sortFrameworksByDependency(reg)
	if len(sorted) != len(frameworks) {
		t.Fatalf("expected %d frameworks in output, got %d", len(frameworks), len(sorted))
	}
	seen := map[string]int{}
	for _, framework := range sorted {
		seen[framework.Framework]++
	}
	for _, framework := range frameworks {
		if seen[framework.Framework] != 1 {
			t.Errorf("framework %q appears %d times in sorted output", framework.Framework, seen[framework.Framework])
		}
	}
}
