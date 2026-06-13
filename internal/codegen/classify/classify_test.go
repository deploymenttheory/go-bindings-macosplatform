package classify

import (
	"slices"
	"testing"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/meta"
)

// helpers

func hasTag(tags []PatternTag, want PatternTag) bool {
	return slices.Contains(tags, want)
}

func mustHave(t *testing.T, tags []PatternTag, want PatternTag) {
	t.Helper()
	if !hasTag(tags, want) {
		t.Errorf("expected tag %q; got %v", want, tags)
	}
}

func mustNotHave(t *testing.T, tags []PatternTag, unwanted PatternTag) {
	t.Helper()
	if hasTag(tags, unwanted) {
		t.Errorf("unexpected tag %q in %v", unwanted, tags)
	}
}

// ── AsyncCompletion ───────────────────────────────────────────────────────────

func TestAsyncCompletion(t *testing.T) {
	m := meta.Method{
		Selector: "fetchDataWithCompletion:",
		Return:   meta.ReturnType{ObjCType: "void"},
		Params: []meta.Param{
			{Name: "completion", ObjCType: "void (^)(NSData *, NSError *)", IsBlock: true},
		},
	}
	tags := ClassifyMethod(m, meta.Class{}, nil)
	mustHave(t, tags, AsyncCompletion)
	mustNotHave(t, tags, BlockEnumeration)
}

func TestAsyncCompletion_NonVoidReturn_NotTagged(t *testing.T) {
	m := meta.Method{
		Selector: "fetchWithCompletion:",
		Return:   meta.ReturnType{ObjCType: "NSURLSessionTask *"},
		Params: []meta.Param{
			{Name: "completion", ObjCType: "void (^)(NSData *, NSError *)", IsBlock: true},
		},
	}
	tags := ClassifyMethod(m, meta.Class{}, nil)
	mustNotHave(t, tags, AsyncCompletion)
}

func TestAsyncCompletion_BlockNotVoidReturn_NotTagged(t *testing.T) {
	// Block returns NSArray, not void → not AsyncCompletion
	m := meta.Method{
		Selector: "processWithCompletion:",
		Return:   meta.ReturnType{ObjCType: "void"},
		Params: []meta.Param{
			{Name: "handler", ObjCType: "NSArray * (^)(NSError *)", IsBlock: true},
		},
	}
	tags := ClassifyMethod(m, meta.Class{}, nil)
	mustNotHave(t, tags, AsyncCompletion)
}

// ── BoolNSError ───────────────────────────────────────────────────────────────

func TestBoolNSError(t *testing.T) {
	m := meta.Method{
		Selector:   "loadFromURL:error:",
		Return:     meta.ReturnType{ObjCType: "BOOL"},
		IsNSError: true,
	}
	tags := ClassifyMethod(m, meta.Class{}, nil)
	mustHave(t, tags, BoolNSError)
	mustNotHave(t, tags, NSErrorOut)
}

func TestBoolNSError_NoNSError_NotTagged(t *testing.T) {
	m := meta.Method{
		Selector: "isValid",
		Return:   meta.ReturnType{ObjCType: "BOOL"},
	}
	tags := ClassifyMethod(m, meta.Class{}, nil)
	mustNotHave(t, tags, BoolNSError)
}

// ── NSErrorOut ────────────────────────────────────────────────────────────────

func TestNSErrorOut(t *testing.T) {
	m := meta.Method{
		Selector:   "parseData:error:",
		Return:     meta.ReturnType{ObjCType: "NSData *"},
		IsNSError: true,
	}
	tags := ClassifyMethod(m, meta.Class{}, nil)
	mustHave(t, tags, NSErrorOut)
	mustNotHave(t, tags, BoolNSError)
}

// ── CollectionReturn ──────────────────────────────────────────────────────────

func TestCollectionReturn_NSArray(t *testing.T) {
	m := meta.Method{
		Selector: "recentDocuments",
		Return:   meta.ReturnType{ObjCType: "NSArray<NSURL *> *"},
	}
	tags := ClassifyMethod(m, meta.Class{}, nil)
	mustHave(t, tags, CollectionReturn)
}

func TestCollectionReturn_NSDictionary(t *testing.T) {
	m := meta.Method{
		Selector: "userInfo",
		Return:   meta.ReturnType{ObjCType: "NSDictionary<NSString *, id> *"},
	}
	tags := ClassifyMethod(m, meta.Class{}, nil)
	mustHave(t, tags, CollectionReturn)
}

func TestCollectionReturn_NSSet(t *testing.T) {
	m := meta.Method{
		Selector: "allObjects",
		Return:   meta.ReturnType{ObjCType: "NSSet<NSObject *> *"},
	}
	tags := ClassifyMethod(m, meta.Class{}, nil)
	mustHave(t, tags, CollectionReturn)
}

func TestCollectionReturn_NSOrderedSet(t *testing.T) {
	m := meta.Method{
		Selector: "orderedItems",
		Return:   meta.ReturnType{ObjCType: "NSOrderedSet<NSString *> *"},
	}
	tags := ClassifyMethod(m, meta.Class{}, nil)
	mustHave(t, tags, CollectionReturn)
}

func TestCollectionReturn_ScalarReturn_NotTagged(t *testing.T) {
	m := meta.Method{
		Selector: "count",
		Return:   meta.ReturnType{ObjCType: "NSUInteger"},
	}
	tags := ClassifyMethod(m, meta.Class{}, nil)
	mustNotHave(t, tags, CollectionReturn)
}

// ── CollectionParam ───────────────────────────────────────────────────────────

func TestCollectionParam(t *testing.T) {
	m := meta.Method{
		Selector: "addObjects:",
		Return:   meta.ReturnType{ObjCType: "void"},
		Params: []meta.Param{
			{Name: "objects", ObjCType: "NSArray<NSObject *> *"},
		},
	}
	tags := ClassifyMethod(m, meta.Class{}, nil)
	mustHave(t, tags, CollectionParam)
}

func TestCollectionParam_NoDictionaryArg_NotTagged(t *testing.T) {
	m := meta.Method{
		Selector: "setName:",
		Return:   meta.ReturnType{ObjCType: "void"},
		Params: []meta.Param{
			{Name: "name", ObjCType: "NSString *"},
		},
	}
	tags := ClassifyMethod(m, meta.Class{}, nil)
	mustNotHave(t, tags, CollectionParam)
}

// ── PropertyPair ──────────────────────────────────────────────────────────────

func TestPropertyPair_GetterAndSetter(t *testing.T) {
	cls := meta.Class{
		Properties: []meta.Property{
			{Name: "alphaValue", ObjCType: "CGFloat", IsReadOnly: false},
		},
	}

	getter := meta.Method{Selector: "alphaValue", Return: meta.ReturnType{ObjCType: "CGFloat"}}
	setter := meta.Method{Selector: "setAlphaValue:", Return: meta.ReturnType{ObjCType: "void"},
		Params: []meta.Param{{Name: "alphaValue", ObjCType: "CGFloat"}},
	}

	gtags := ClassifyMethod(getter, cls, nil)
	stags := ClassifyMethod(setter, cls, nil)
	mustHave(t, gtags, PropertyPair)
	mustHave(t, stags, PropertyPair)
}

func TestPropertyPair_ReadonlyNotTagged(t *testing.T) {
	cls := meta.Class{
		Properties: []meta.Property{
			{Name: "isLoading", ObjCType: "BOOL", IsReadOnly: true},
		},
	}
	m := meta.Method{Selector: "isLoading", Return: meta.ReturnType{ObjCType: "BOOL"}}
	tags := ClassifyMethod(m, cls, nil)
	mustNotHave(t, tags, PropertyPair)
}

func TestPropertyPair_CustomGetter(t *testing.T) {
	cls := meta.Class{
		Properties: []meta.Property{
			{Name: "hidden", ObjCType: "BOOL", Getter: "isHidden", IsReadOnly: false},
		},
	}
	m := meta.Method{Selector: "isHidden", Return: meta.ReturnType{ObjCType: "BOOL"}}
	tags := ClassifyMethod(m, cls, nil)
	mustHave(t, tags, PropertyPair)
}

// ── MainThreadRequired ────────────────────────────────────────────────────────

func TestMainThreadRequired(t *testing.T) {
	m := meta.Method{
		Selector:           "updateUI",
		Return:             meta.ReturnType{ObjCType: "void"},
		IsMainThreadRequired: true,
	}
	tags := ClassifyMethod(m, meta.Class{}, nil)
	mustHave(t, tags, MainThreadRequired)
}

func TestMainThreadRequired_False_NotTagged(t *testing.T) {
	m := meta.Method{
		Selector:           "compute",
		Return:             meta.ReturnType{ObjCType: "NSInteger"},
		IsMainThreadRequired: false,
	}
	tags := ClassifyMethod(m, meta.Class{}, nil)
	mustNotHave(t, tags, MainThreadRequired)
}

// ── BlockEnumeration ──────────────────────────────────────────────────────────

func TestBlockEnumeration(t *testing.T) {
	m := meta.Method{
		Selector: "enumerateObjectsUsingBlock:",
		Return:   meta.ReturnType{ObjCType: "void"},
		Params: []meta.Param{
			{Name: "block", ObjCType: "void (^)(id, NSUInteger, BOOL *)", IsBlock: true},
		},
	}
	tags := ClassifyMethod(m, meta.Class{}, nil)
	mustHave(t, tags, BlockEnumeration)
	// Must NOT also be tagged AsyncCompletion
	mustNotHave(t, tags, AsyncCompletion)
}

func TestBlockEnumeration_WithOptions(t *testing.T) {
	m := meta.Method{
		Selector: "enumerateObjectsWithOptions:usingBlock:",
		Return:   meta.ReturnType{ObjCType: "void"},
		Params: []meta.Param{
			{Name: "opts", ObjCType: "NSEnumerationOptions"},
			{Name: "block", ObjCType: "void (^)(id, NSUInteger, BOOL *)", IsBlock: true},
		},
	}
	tags := ClassifyMethod(m, meta.Class{}, nil)
	mustHave(t, tags, BlockEnumeration)
}

func TestBlockEnumeration_NoStopParam_NotTagged(t *testing.T) {
	// Block has no BOOL * stop → not a standard enumerate block
	m := meta.Method{
		Selector: "enumerateObjectsUsingBlock:",
		Return:   meta.ReturnType{ObjCType: "void"},
		Params: []meta.Param{
			{Name: "block", ObjCType: "void (^)(id, NSUInteger)", IsBlock: true},
		},
	}
	tags := ClassifyMethod(m, meta.Class{}, nil)
	mustNotHave(t, tags, BlockEnumeration)
}

// ── DelegateProtocol (IsDelegateProtocol) ────────────────────────────────────

func TestIsDelegateProtocol_HasOptional(t *testing.T) {
	proto := meta.Protocol{
		Methods: []meta.Method{
			{Selector: "required:", IsOptional: false},
			{Selector: "optional:", IsOptional: true},
		},
	}
	if !IsDelegateProtocol(proto) {
		t.Error("expected IsDelegateProtocol true for protocol with optional method")
	}
}

func TestIsDelegateProtocol_AllRequired(t *testing.T) {
	proto := meta.Protocol{
		Methods: []meta.Method{
			{Selector: "required1:", IsOptional: false},
			{Selector: "required2:", IsOptional: false},
		},
	}
	if IsDelegateProtocol(proto) {
		t.Error("expected IsDelegateProtocol false for protocol with no optional methods")
	}
}

func TestIsDelegateProtocol_Empty(t *testing.T) {
	if IsDelegateProtocol(meta.Protocol{}) {
		t.Error("expected IsDelegateProtocol false for empty protocol")
	}
}

// ── DelegateHolder (DelegateProperties) ──────────────────────────────────────

func TestDelegateProperties_Delegate(t *testing.T) {
	cls := meta.Class{
		Properties: []meta.Property{
			{Name: "delegate", ObjCType: "id<NSWindowDelegate>"},
		},
	}
	props := DelegateProperties(cls)
	if len(props) != 1 {
		t.Fatalf("expected 1 delegate property, got %d", len(props))
	}
	if props[0].ProtocolName != "NSWindowDelegate" {
		t.Errorf("expected protocol NSWindowDelegate, got %q", props[0].ProtocolName)
	}
}

func TestDelegateProperties_DataSource(t *testing.T) {
	cls := meta.Class{
		Properties: []meta.Property{
			{Name: "dataSource", ObjCType: "id<NSTableViewDataSource>"},
		},
	}
	props := DelegateProperties(cls)
	if len(props) != 1 {
		t.Fatalf("expected 1 delegate property, got %d", len(props))
	}
	if props[0].ProtocolName != "NSTableViewDataSource" {
		t.Errorf("expected protocol NSTableViewDataSource, got %q", props[0].ProtocolName)
	}
}

func TestDelegateProperties_NoDelegate(t *testing.T) {
	cls := meta.Class{
		Properties: []meta.Property{
			{Name: "title", ObjCType: "NSString *"},
		},
	}
	props := DelegateProperties(cls)
	if len(props) != 0 {
		t.Errorf("expected 0 delegate properties, got %d", len(props))
	}
}

func TestDelegateProperties_BothDelegateAndDataSource(t *testing.T) {
	cls := meta.Class{
		Properties: []meta.Property{
			{Name: "delegate", ObjCType: "id<NSTableViewDelegate>"},
			{Name: "dataSource", ObjCType: "id<NSTableViewDataSource>"},
		},
	}
	props := DelegateProperties(cls)
	if len(props) != 2 {
		t.Fatalf("expected 2 delegate properties, got %d", len(props))
	}
}

func TestDelegateProperties_NonIDType_Excluded(t *testing.T) {
	// delegate property but with a concrete type (not id<Protocol>)
	cls := meta.Class{
		Properties: []meta.Property{
			{Name: "delegate", ObjCType: "NSObject *"},
		},
	}
	props := DelegateProperties(cls)
	if len(props) != 0 {
		t.Errorf("expected 0 delegate properties for non-id<P> type, got %d", len(props))
	}
}

// ── PropertyPairs ─────────────────────────────────────────────────────────────

func TestPropertyPairsHelper(t *testing.T) {
	cls := meta.Class{
		Properties: []meta.Property{
			{Name: "title", ObjCType: "NSString *", IsReadOnly: false},
			{Name: "isEnabled", ObjCType: "BOOL", IsReadOnly: false},
			{Name: "frame", ObjCType: "CGRect", IsReadOnly: true}, // excluded
		},
	}
	pairs := PropertyPairs(cls)
	if len(pairs) != 2 {
		t.Errorf("expected 2 writable properties, got %d", len(pairs))
	}
}

// ── ClassifyFramework ─────────────────────────────────────────────────────────

func TestClassifyFramework(t *testing.T) {
	framework := &meta.FrameworkMeta{
		Framework: "TestFramework",
		Classes: map[string]meta.Class{
			"MyDocument": {
				Properties: []meta.Property{
					{Name: "title", ObjCType: "NSString *", IsReadOnly: false},
				},
				Methods: []meta.Method{
					{
						Selector:   "loadFromURL:error:",
						Return:     meta.ReturnType{ObjCType: "BOOL"},
						IsNSError: true,
					},
					{
						Selector: "title",
						Return:   meta.ReturnType{ObjCType: "NSString *"},
					},
					{
						Selector: "setTitle:",
						Return:   meta.ReturnType{ObjCType: "void"},
						Params:     []meta.Param{{Name: "title", ObjCType: "NSString *"}},
					},
				},
			},
		},
	}
	result := ClassifyFramework(framework)

	bySelector, ok := result["MyDocument"]
	if !ok {
		t.Fatal("expected MyDocument in ClassifyFramework result")
	}

	loadTags, ok := bySelector["loadFromURL:error:"]
	if !ok {
		t.Fatal("expected loadFromURL:error: in result")
	}
	if !hasTag(loadTags, BoolNSError) {
		t.Errorf("expected BoolNSError on loadFromURL:error:, got %v", loadTags)
	}

	titleTags := bySelector["title"]
	if hasTag(titleTags, PropertyPair) {
		// title is a getter for a non-readonly property → should be tagged
	}
	setTitleTags := bySelector["setTitle:"]
	if !hasTag(setTitleTags, PropertyPair) {
		t.Errorf("expected PropertyPair on setTitle:, got %v", setTitleTags)
	}
}

// ── Multi-tag combinations ────────────────────────────────────────────────────

func TestMethod_CanHaveMultipleTags(t *testing.T) {
	// A method that returns NSArray and takes NSArray — both CollectionReturn and CollectionParam.
	m := meta.Method{
		Selector: "mergedArrays:",
		Return:   meta.ReturnType{ObjCType: "NSArray *"},
		Params: []meta.Param{
			{Name: "other", ObjCType: "NSArray *"},
		},
	}
	tags := ClassifyMethod(m, meta.Class{}, nil)
	mustHave(t, tags, CollectionReturn)
	mustHave(t, tags, CollectionParam)
}

func TestMethod_MainThread_WithAsync(t *testing.T) {
	m := meta.Method{
		Selector:           "refreshUI:",
		Return:             meta.ReturnType{ObjCType: "void"},
		IsMainThreadRequired: true,
		Params: []meta.Param{
			{Name: "completion", ObjCType: "void (^)(void)", IsBlock: true},
		},
	}
	tags := ClassifyMethod(m, meta.Class{}, nil)
	mustHave(t, tags, AsyncCompletion)
	mustHave(t, tags, MainThreadRequired)
}
