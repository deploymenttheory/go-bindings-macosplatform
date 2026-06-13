//go:build darwin

package acceptance_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"regexp"
	"sync"
	"testing"
	"time"
)

// subprocessEnv is set to "1" when the test is running as an isolated subprocess.
// runIsolated checks this to avoid infinite recursion.
const subprocessEnv = "GENACCEPT_SUBPROCESS"

// attestEnv names the file path to append JSONL attestation records to.
const attestEnv = "GENACCEPT_ATTEST"

var (
	attestMu   sync.Mutex
	attestFile *os.File
)

func init() {
	if path := os.Getenv(attestEnv); path != "" {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err == nil {
			attestFile = f
		}
	}
}

// runIsolated re-executes the current test as a subprocess so that a SIGSEGV or
// ObjC abort in the generated bridge cannot crash the entire test run. The
// subprocess inherits TestMain's main-loop pump, so RunOnMainThread works.
//
// When GENACCEPT_SUBPROCESS=1 is already set (we are the subprocess) fn is
// called directly to avoid infinite recursion.
func runIsolated(t *testing.T, id string, fn func(*testing.T)) {
	t.Helper()
	if os.Getenv(subprocessEnv) == "1" {
		fn(t)
		return
	}

	start := time.Now()
	cmd := exec.Command(
		os.Args[0],
		"-test.run=^"+regexp.QuoteMeta(t.Name())+"$",
		"-test.v",
		"-test.timeout=10s",
	)
	cmd.Env = append(os.Environ(), subprocessEnv+"=1")
	out, err := cmd.CombinedOutput()
	ms := time.Since(start).Milliseconds()

	if err != nil {
		t.Errorf("subprocess failed: %v\noutput:\n%s", err, out)
		writeAttest(id, "fail", err.Error(), ms)
		return
	}
	writeAttest(id, "pass", "", ms)
}

// writeAttest appends one JSONL record to the attestation file (if configured).
func writeAttest(id, status, errMsg string, ms int64) {
	if attestFile == nil {
		return
	}
	rec := map[string]any{
		"id":     id,
		"status": status,
		"ms":     ms,
		"run_at": time.Now().UTC().Format(time.RFC3339),
	}
	if errMsg != "" {
		rec["error"] = errMsg
	}
	b, _ := json.Marshal(rec)
	attestMu.Lock()
	_, _ = attestFile.Write(append(b, '\n'))
	attestMu.Unlock()
}
