//go:build darwin

package puregolibs_test

import (
	"os"
	"path/filepath"
	"testing"

	xar "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/xar"
)

// TestXar_CreateAndReopen writes a real xar archive to disk, closes it,
// reopens it read-only, and checks the negative path — string params, handle
// returns, and int32 rc handling live.
func TestXar_CreateAndReopen(t *testing.T) {
	const (
		xarRead  = 0
		xarWrite = 1
	)
	path := filepath.Join(t.TempDir(), "t.xar")

	w := xar.Xar_open(path, xarWrite)
	if w == nil {
		t.Fatal("xar_open(WRITE) returned nil")
	}
	if rc := xar.Xar_close(w); rc != 0 {
		t.Fatalf("xar_close(write) rc = %d", rc)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("archive was not created: %v", err)
	}

	r := xar.Xar_open(path, xarRead)
	if r == nil {
		t.Fatal("xar_open(READ) on the just-written archive returned nil")
	}
	if rc := xar.Xar_close(r); rc != 0 {
		t.Errorf("xar_close(read) rc = %d", rc)
	}

	if bad := xar.Xar_open(filepath.Join(t.TempDir(), "missing", "no.xar"), xarRead); bad != nil {
		xar.Xar_close(bad)
		t.Error("xar_open on a nonexistent path returned a handle")
	}
}
