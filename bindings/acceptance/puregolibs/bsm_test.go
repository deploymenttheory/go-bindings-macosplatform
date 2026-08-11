//go:build darwin

package puregolibs_test

import (
	"testing"

	bsm "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/bsm"
	endpointsecurity "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/endpointsecurity"
)

// TestBsm_AuditTokenByValue is the field-exact probe for 32-byte
// struct-by-value marshalling: audit_token_t's layout is documented — val[3]
// is the real uid, val[5] the pid, val[6] the audit session id. Each accessor
// just reads its dword, so a hand-built token must round-trip every field
// exactly; any byte shift or truncation in the ABI crossing changes the
// result.
func TestBsm_AuditTokenByValue(t *testing.T) {
	token := endpointsecurity.AuditTokenT{
		Val: [8]uint32{0xA0A0A0A0, 501, 20, 501, 20, 43210, 0x0D15EA5E, 7},
	}
	if got := bsm.Audit_token_to_pid(token); got != 43210 {
		t.Errorf("audit_token_to_pid = %d; want 43210 (val[5])", got)
	}
	if got := bsm.Audit_token_to_ruid(token); got != 501 {
		t.Errorf("audit_token_to_ruid = %d; want 501 (val[3])", got)
	}
	if got := bsm.Audit_token_to_asid(token); got != 0x0D15EA5E {
		t.Errorf("audit_token_to_asid = %#x; want 0xD15EA5E (val[6])", got)
	}
}
