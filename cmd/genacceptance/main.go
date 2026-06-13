//go:build darwin

package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const modulePrefix = "github.com/deploymenttheory/go-bindings-macosplatform"

func main() {
	n := flag.Int("n", 100, "number of functions to sample")
	seedHex := flag.String("seed", "", "hex seed for the RNG (empty or 0 = use current time)")
	useGit := flag.Bool("git", false, "derive seed from git HEAD commit SHA (overrides --seed)")
	metaDir := flag.String("metadata-dir", "./metadata/frameworks", "directory containing .gometa.json files")
	out := flag.String("out", "./acceptance/generated_acceptance_test.go", "output test file path")
	flag.Parse()

	seed := resolveSeed(*seedHex, *useGit)

	// Phase 1: load candidates from metadata.
	candidates, err := loadCandidates([]string{*metaDir}, modulePrefix)
	if err != nil {
		log.Fatalf("load: %v", err)
	}
	fmt.Fprintf(os.Stderr, "loaded %d auto-testable candidates\n", len(candidates))

	// Phase 2: stratified sample.
	rng := rand.New(rand.NewSource(seed))
	sample := stratifiedSample(candidates, *n, rng)
	fmt.Fprintf(os.Stderr, "sampled %d functions (seed=0x%x)\n", len(sample), seed)

	// Phase 3: emit.
	f, err := os.Create(*out)
	if err != nil {
		log.Fatalf("create output: %v", err)
	}
	defer f.Close()

	if err := emitTestFile(f, sample, seed, modulePrefix); err != nil {
		log.Fatalf("emit: %v", err)
	}
	fmt.Fprintf(os.Stderr, "wrote %s\n", *out)
}

// resolveSeed converts the CLI flags into an int64 seed for the RNG.
// Priority: --git > --seed > time.
func resolveSeed(seedHex string, useGit bool) int64 {
	if useGit {
		sha, err := gitHeadSHA()
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not read git HEAD SHA (%v); falling back to time\n", err)
		} else {
			seed, err := hexToSeed(sha[:16])
			if err == nil {
				return seed
			}
		}
	}
	if seedHex != "" && seedHex != "0" {
		seed, err := hexToSeed(seedHex)
		if err == nil {
			return seed
		}
		// Fall through: maybe it's a decimal integer from older invocations.
		if n, err := strconv.ParseInt(seedHex, 10, 64); err == nil {
			return n
		}
	}
	return time.Now().UnixNano()
}

// gitHeadSHA returns the full 40-char hex SHA of the current HEAD commit.
func gitHeadSHA() (string, error) {
	out, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		return "", err
	}
	sha := strings.TrimSpace(string(out))
	if len(sha) < 16 {
		return "", fmt.Errorf("unexpected SHA length: %q", sha)
	}
	return sha, nil
}

// hexToSeed converts the first 8 bytes of a hex string to an int64 seed.
func hexToSeed(h string) (int64, error) {
	if len(h) > 16 {
		h = h[:16]
	}
	// Pad to even length.
	if len(h)%2 != 0 {
		h = "0" + h
	}
	b, err := hex.DecodeString(h)
	if err != nil {
		return 0, err
	}
	var v int64
	for _, byt := range b {
		v = (v << 8) | int64(byt)
	}
	return v, nil
}
