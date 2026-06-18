//go:build darwin

package rules

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/deploymenttheory/go-bindings-macosplatform/examples/warden/shared"
)

func TestFindVerdicts(t *testing.T) {
	e := New("")
	e.Add(&shared.Rule{UUID: "1", Key: "/bin/curl", Path: "/bin/curl", Action: shared.RuleStateAllow})
	e.Add(&shared.Rule{UUID: "2", Key: "/bin/nc", Path: "/bin/nc", Action: shared.RuleStateBlock})
	// Endpoint-specific block overrides a process-wide allow.
	e.Add(&shared.Rule{UUID: "3", Key: "/bin/curl", Path: "/bin/curl", EndpointAddr: "1.2.3.4", Action: shared.RuleStateBlock})

	if got := e.Find("/bin/curl", "9.9.9.9", "443"); got != shared.RuleStateAllow {
		t.Errorf("curl→9.9.9.9: got %d, want allow", got)
	}
	if got := e.Find("/bin/curl", "1.2.3.4", "443"); got != shared.RuleStateBlock {
		t.Errorf("curl→1.2.3.4: endpoint rule should win; got %d, want block", got)
	}
	if got := e.Find("/bin/nc", "5.5.5.5", "53"); got != shared.RuleStateBlock {
		t.Errorf("nc: got %d, want block", got)
	}
	if got := e.Find("/bin/unknown", "5.5.5.5", "53"); got != shared.RuleStateNotFound {
		t.Errorf("unknown: got %d, want not-found", got)
	}
}

func TestDisabledAndExpired(t *testing.T) {
	e := New("")
	past := time.Now().Add(-time.Hour)
	e.Add(&shared.Rule{UUID: "1", Key: "/a", Action: shared.RuleStateAllow, IsDisabled: true})
	e.Add(&shared.Rule{UUID: "2", Key: "/b", Action: shared.RuleStateAllow, Expiration: &past})
	if got := e.Find("/a", "x", "1"); got != shared.RuleStateNotFound {
		t.Errorf("disabled rule matched: got %d", got)
	}
	if got := e.Find("/b", "x", "1"); got != shared.RuleStateNotFound {
		t.Errorf("expired rule matched: got %d", got)
	}
}

func TestToggleDelete(t *testing.T) {
	e := New("")
	e.Add(&shared.Rule{UUID: "1", Key: "/a", Action: shared.RuleStateAllow})
	if !e.Toggle("/a", "1", true) || e.Find("/a", "x", "1") != shared.RuleStateNotFound {
		t.Error("toggle-disable failed")
	}
	if !e.Toggle("/a", "1", false) || e.Find("/a", "x", "1") != shared.RuleStateAllow {
		t.Error("toggle-enable failed")
	}
	if !e.Delete("/a", "1") || len(e.All()) != 0 {
		t.Error("delete failed")
	}
}

func TestPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rules.json")
	e := New(path)
	e.Add(&shared.Rule{UUID: "1", Key: "/a", Path: "/a", Action: shared.RuleStateBlock, Creation: time.Now()})
	if err := e.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}
	e2 := New(path)
	if err := e2.Load(); err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := e2.Find("/a", "x", "1"); got != shared.RuleStateBlock {
		t.Errorf("after reload: got %d, want block", got)
	}
}
