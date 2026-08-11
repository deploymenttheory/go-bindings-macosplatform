//go:build darwin

package puregolibs_test

import (
	"bytes"
	"testing"

	compression "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/compression"
)

// TestCompression_RoundTrip compresses a buffer with zlib and decompresses it
// back — proving pointer, length, and enum marshalling end to end: the output
// is only byte-identical if every parameter crossed the ABI correctly.
func TestCompression_RoundTrip(t *testing.T) {
	src := bytes.Repeat([]byte("the quick brown fox jumps over the lazy dog. "), 100)
	dst := make([]byte, len(src)+4096)

	encoded := compression.Compression_encode_buffer(
		&dst[0], uint64(len(dst)),
		&src[0], uint64(len(src)),
		nil, compression.COMPRESSION_ZLIB,
	)
	if encoded == 0 || encoded >= uint64(len(src)) {
		t.Fatalf("encode: %d bytes from %d input; want 0 < n < input (compressible data)", encoded, len(src))
	}

	back := make([]byte, len(src)+16)
	decoded := compression.Compression_decode_buffer(
		&back[0], uint64(len(back)),
		&dst[0], encoded,
		nil, compression.COMPRESSION_ZLIB,
	)
	if decoded != uint64(len(src)) {
		t.Fatalf("decode: %d bytes; want %d", decoded, len(src))
	}
	if !bytes.Equal(back[:decoded], src) {
		t.Fatal("round-trip corrupted the data")
	}
}

// TestCompression_ScratchSizes checks the enum-by-value marshalling on a
// simple scalar-returning call for every documented algorithm.
func TestCompression_ScratchSizes(t *testing.T) {
	for _, alg := range []compression.CompressionAlgorithm{
		compression.COMPRESSION_LZ4, compression.COMPRESSION_ZLIB,
	} {
		if compression.Compression_encode_scratch_buffer_size(alg) == 0 {
			t.Errorf("scratch buffer size for %v = 0; want > 0", alg)
		}
	}
}
