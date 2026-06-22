//go:build darwin

package appkit_test

import (
	"github.com/deploymenttheory/go-bindings-macosplatform/opinionated/idiomatic/framework/appkit"
)

// ExampleStackView builds a stack view with the fluent API. Each With* setter
// returns the receiver, so layout configuration reads as a single expression;
// scalar and boolean properties surface as plain Go float64 and bool.
//
// This example is compile-only (no Output line), so it doubles as a smoke test
// that the generated AppKit API is usable. AppKit objects must be created and
// used on the main thread, which is why this example does not run.
func ExampleStackView() {
	stack := appkit.NewStackView().
		WithSpacing(8).
		WithDetachesHiddenViews(true).
		WithHasEqualSpacing(true)

	_ = stack
}
