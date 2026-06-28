package mainactor

import (
	"testing"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/meta"
	rootactor "github.com/deploymenttheory/go-bindings-macosplatform/internal/mainactor"
)

func TestApply(t *testing.T) {
	fw := &meta.FrameworkMeta{Classes: map[string]meta.Class{
		"NSView": {Methods: []meta.Method{
			{Selector: "layout"},
			{Selector: "isFlipped"},
		}},
		"NSColor": {Methods: []meta.Method{
			{Selector: "highlightWithLevel:"},
			{Selector: "redColor", IsClassMethod: true},
		}},
	}}

	iso := &rootactor.Isolation{
		MainActorClasses:     []string{"NSView"},
		MainActorSelectors:   map[string][]string{"NSColor": {"highlightWithLevel:"}},
		NonisolatedSelectors: map[string][]string{"NSView": {"isFlipped"}},
	}
	Apply(iso, fw)

	if !fw.Classes["NSView"].IsMainThreadRequired {
		t.Error("NSView should be main-thread")
	}
	// nonisolated opt-out recorded on the exempted method.
	for _, m := range fw.Classes["NSView"].Methods {
		if m.Selector == "isFlipped" && !m.IsMainThreadExempt {
			t.Error("isFlipped should be exempt (nonisolated)")
		}
	}
	// Individually-isolated selector marked on an otherwise non-isolated class.
	var sawIsolated bool
	for _, m := range fw.Classes["NSColor"].Methods {
		if m.Selector == "highlightWithLevel:" && m.IsMainThreadRequired {
			sawIsolated = true
		}
	}
	if !sawIsolated {
		t.Error("NSColor.highlightWithLevel: should be main-thread")
	}
	if fw.Classes["NSColor"].IsMainThreadRequired {
		t.Error("NSColor should NOT be a whole-class main-thread type")
	}
}

func TestPropagate(t *testing.T) {
	// Cross-framework hierarchy: NSButton → NSControl → NSView (root), in one
	// framework; MKMapView → NSView in another. NSObject is unrelated.
	appkit := &meta.FrameworkMeta{Classes: map[string]meta.Class{
		"NSView":    {Super: "NSResponder", IsMainThreadRequired: true, Methods: []meta.Method{{Selector: "layout"}}},
		"NSControl": {Super: "NSView", Methods: []meta.Method{{Selector: "sizeToFit"}}},
		"NSButton":  {Super: "NSControl", Methods: []meta.Method{{Selector: "setTitle:"}, {Selector: "factory", IsClassMethod: true}, {Selector: "threadSafe", IsMainThreadExempt: true}}},
		"NSObject":  {Methods: []meta.Method{{Selector: "description"}}},
	}}
	mapkit := &meta.FrameworkMeta{Classes: map[string]meta.Class{
		"MKMapView": {Super: "NSView", Methods: []meta.Method{{Selector: "setRegion:"}}},
	}}

	Propagate([]*meta.FrameworkMeta{appkit, mapkit})

	wantMain := map[string]map[string]bool{
		"appkit": {"NSView": true, "NSControl": true, "NSButton": true, "NSObject": false},
		"mapkit": {"MKMapView": true},
	}
	for name, exp := range wantMain["appkit"] {
		if got := appkit.Classes[name].IsMainThreadRequired; got != exp {
			t.Errorf("appkit %s: IsMainThreadRequired=%v want %v", name, got, exp)
		}
	}
	if !mapkit.Classes["MKMapView"].IsMainThreadRequired {
		t.Error("MKMapView should inherit main-thread from NSView across frameworks")
	}

	// Instance methods of a propagated class are stamped; class methods and
	// nonisolated methods are not.
	button := appkit.Classes["NSButton"]
	for _, m := range button.Methods {
		switch m.Selector {
		case "setTitle:":
			if !m.IsMainThreadRequired {
				t.Error("NSButton.setTitle: should be stamped main-thread")
			}
		case "factory":
			if m.IsMainThreadRequired {
				t.Error("class method NSButton.factory should NOT be stamped")
			}
		case "threadSafe":
			if m.IsMainThreadRequired {
				t.Error("nonisolated NSButton.threadSafe should NOT be stamped")
			}
		}
	}
}
