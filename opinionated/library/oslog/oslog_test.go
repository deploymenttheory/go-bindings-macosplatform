//go:build darwin

package oslog

import (
	"fmt"
	"testing"
	"time"
)

// TestLoggerEmits writes one message per level; pair with
//
//	log show --last 2m --predicate 'subsystem == "com.deploymenttheory.orin.test"'
//
// to confirm the messages reached unified logging.
func TestLoggerEmits(t *testing.T) {
	logger := NewLogger("com.deploymenttheory.orin.test", "unit")

	marker := fmt.Sprintf("oslog-test-%d", time.Now().UnixNano())
	logger.Default(marker + " default")
	logger.Info(marker + " info")
	logger.Debug(marker + " debug")
	logger.Error(marker + " error")
	logger.Fault(marker + " fault")

	t.Log(marker)
}
