//go:build darwin

package config

import (
	"testing"

	"github.com/deploymenttheory/go-bindings-macosplatform/examples/lulu/rules"
	"github.com/deploymenttheory/go-bindings-macosplatform/examples/lulu/shared"
)

const yamlDoc = `
version: "1"
defaultAction: block
rules:
  - name: curl
    path: /usr/bin/curl
    action: allow
  - name: nc
    path: /usr/bin/nc
    action: block
    endpoints:
      - host: 1.2.3.4
        port: "443"
`

const jsonDoc = `{
  "version": "1",
  "defaultAction": "block",
  "rules": [
    {"name": "curl", "path": "/usr/bin/curl", "action": "allow"},
    {"name": "nc", "path": "/usr/bin/nc", "action": "block",
     "endpoints": [{"host": "1.2.3.4", "port": "443"}]}
  ]
}`

func TestParseJSONAndYAMLAgree(t *testing.T) {
	y, err := Parse([]byte(yamlDoc), FormatYAML)
	if err != nil {
		t.Fatalf("yaml: %v", err)
	}
	j, err := Parse([]byte(jsonDoc), FormatJSON)
	if err != nil {
		t.Fatalf("json: %v", err)
	}
	yd, jd := y.Desired(), j.Desired()
	if len(yd) != len(jd) || len(yd) != 2 {
		t.Fatalf("desired counts differ: yaml=%d json=%d", len(yd), len(jd))
	}
	// Content-addressed UUIDs must match across formats for the same logical rule.
	ym := map[string]bool{}
	for _, r := range yd {
		ym[r.UUID] = true
	}
	for _, r := range jd {
		if !ym[r.UUID] {
			t.Errorf("json rule %s (%s) has no YAML counterpart", r.Path, r.UUID)
		}
	}
}

func TestValidate(t *testing.T) {
	if _, err := Parse([]byte(`{"rules":[{"path":"/x","action":"nope"}]}`), FormatJSON); err == nil {
		t.Error("expected invalid-action error")
	}
	if _, err := Parse([]byte(`{"rules":[{"action":"allow"}]}`), FormatJSON); err == nil {
		t.Error("expected missing-path error")
	}
}

func TestApplyIdempotent(t *testing.T) {
	cfg, _ := Parse([]byte(yamlDoc), FormatYAML)
	eng := rules.New("")

	added, deleted := Apply(cfg, eng)
	if added != 2 || deleted != 0 {
		t.Fatalf("first apply: +%d -%d, want +2 -0", added, deleted)
	}
	added, deleted = Apply(cfg, eng)
	if added != 0 || deleted != 0 {
		t.Fatalf("second apply (idempotent): +%d -%d, want +0 -0", added, deleted)
	}

	// Editing the config prunes the removed managed rule and adds the new one.
	cfg2, _ := Parse([]byte(`{"rules":[{"path":"/usr/bin/curl","action":"block"}]}`), FormatJSON)
	added, deleted = Apply(cfg2, eng)
	if added != 1 || deleted != 2 {
		t.Fatalf("edited apply: +%d -%d, want +1 -2", added, deleted)
	}
	if eng.Find("/usr/bin/curl", "x", "1") != shared.RuleStateBlock {
		t.Error("curl should now be blocked after re-apply")
	}
}

func TestApplyPrunesOnlyManaged(t *testing.T) {
	eng := rules.New("")
	// A hand-added (unmanaged) rule.
	eng.Add(&shared.Rule{UUID: "hand-1", Key: "/usr/bin/ssh", Path: "/usr/bin/ssh", Action: shared.RuleStateAllow})

	cfg, _ := Parse([]byte(`{"rules":[{"path":"/usr/bin/curl","action":"allow"}]}`), FormatJSON)
	Apply(cfg, eng)
	// Re-applying a config that omits ssh must NOT remove the hand-added rule.
	Apply(cfg, eng)
	if eng.Find("/usr/bin/ssh", "x", "1") != shared.RuleStateAllow {
		t.Error("hand-added rule was pruned by apply (should be preserved)")
	}
}

func TestExampleFilesLoadAndAgree(t *testing.T) {
	y, err := Load("firewall.example.yaml")
	if err != nil {
		t.Fatalf("load yaml example: %v", err)
	}
	j, err := Load("firewall.example.json")
	if err != nil {
		t.Fatalf("load json example: %v", err)
	}
	yd, jd := y.Desired(), j.Desired()
	if len(yd) == 0 || len(yd) != len(jd) {
		t.Fatalf("example docs disagree: yaml=%d json=%d desired rules", len(yd), len(jd))
	}
	jm := map[string]bool{}
	for _, r := range jd {
		jm[r.UUID] = true
	}
	for _, r := range yd {
		if !jm[r.UUID] {
			t.Errorf("yaml example rule %s (%s) absent from json example", r.Path, r.UUID)
		}
	}
}

func TestFromRulesRoundTrip(t *testing.T) {
	cfg, _ := Parse([]byte(yamlDoc), FormatYAML)
	desired := cfg.Desired()
	rebuilt := FromRules(desired)
	if len(rebuilt.Rules) != 2 {
		t.Fatalf("FromRules produced %d specs, want 2", len(rebuilt.Rules))
	}
	// Round-tripping the desired set must reproduce the same content-addressed UUIDs.
	got := map[string]bool{}
	for _, r := range rebuilt.Desired() {
		got[r.UUID] = true
	}
	for _, r := range desired {
		if !got[r.UUID] {
			t.Errorf("rule %s (%s) lost in FromRules round-trip", r.Path, r.UUID)
		}
	}
}
