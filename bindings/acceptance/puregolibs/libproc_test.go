//go:build darwin

package puregolibs_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unsafe"

	libproc "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/libproc"
)

// TestLibproc_PidPathAndListPids resolves this very process's executable path
// from its pid and finds its pid in the all-pids listing — buffer out-params
// live against the kernel.
func TestLibproc_PidPathAndListPids(t *testing.T) {
	pid := int32(os.Getpid())

	buf := make([]byte, 4096)
	n := libproc.Proc_pidpath(pid, unsafe.Pointer(&buf[0]), uint32(len(buf)))
	if n <= 0 {
		t.Fatalf("proc_pidpath(%d) = %d", pid, n)
	}
	path := string(buf[:n])
	if !strings.Contains(path, "/") {
		t.Errorf("proc_pidpath returned %q; want an absolute path", path)
	}
	// proc_pidpath returns the resolved path (/private/var/...) where
	// os.Executable may report the /var symlink form — compare post-resolution.
	if self, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(self); err == nil && path != resolved {
			t.Errorf("proc_pidpath = %q; want %q", path, resolved)
		}
	}

	const procAllPids = 1
	pids := make([]int32, 4096)
	got := libproc.Proc_listpids(procAllPids, 0, unsafe.Pointer(&pids[0]), int32(len(pids)*4))
	if got <= 0 {
		t.Fatalf("proc_listpids = %d", got)
	}
	found := false
	for _, p := range pids[:got/4] {
		if p == pid {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("own pid %d not in proc_listpids result (%d pids)", pid, got/4)
	}
}
