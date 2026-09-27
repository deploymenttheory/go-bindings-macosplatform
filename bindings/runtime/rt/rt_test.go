//go:build darwin

package rt

import "testing"

// The rt test binary loads no framework at launch, like a consumer's process.
func TestFileURLRoundTripWithoutFoundationLoaded(t *testing.T) {
	const path = "/private/tmp/rt file url"
	if got := URLString(FileURL(path)); got != path {
		t.Fatalf("URLString(FileURL(%q)) = %q", path, got)
	}
}
