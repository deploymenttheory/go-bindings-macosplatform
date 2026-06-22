//go:build darwin

package foundation_test

import (
	"github.com/deploymenttheory/go-bindings-macosplatform/opinionated/idiomatic/framework/foundation"
)

// ExampleDateComponents builds a set of date components with the fluent API. Each
// With* setter returns the receiver, so the components read as a single
// expression. Scalar properties surface as plain Go ints rather than boxed
// NSNumber objects.
//
// This example is compile-only (no Output line), so it doubles as a smoke test
// that the generated Foundation API is usable.
func ExampleDateComponents() {
	components := foundation.NewDateComponents().
		WithYear(2026).
		WithMonth(6).
		WithDay(22).
		WithHour(9).
		WithMinute(30)

	_ = components
}
