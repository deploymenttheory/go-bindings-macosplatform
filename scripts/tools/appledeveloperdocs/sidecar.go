package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/appledocs"
)

// discoverMetaFiles finds every .gometa.json under metadataDir (frameworks and
// libraries trees, including sub-frameworks one level deeper), preferring the
// arm64 variant when a directory holds several arches.
func discoverMetaFiles(metadataDir string) ([]string, error) {
	var patterns []string
	for _, kind := range []string{"frameworks", "libraries"} {
		base := filepath.Join(metadataDir, kind)
		patterns = append(patterns,
			filepath.Join(base, "*", "*.gometa.json"),
			filepath.Join(base, "*", "*", "*.gometa.json"),
		)
	}

	byDir := map[string][]string{}
	for _, pat := range patterns {
		matches, err := filepath.Glob(pat)
		if err != nil {
			return nil, err
		}
		for _, m := range matches {
			dir := filepath.Dir(m)
			byDir[dir] = append(byDir[dir], m)
		}
	}

	var out []string
	for _, group := range byDir {
		chosen := group[0]
		for _, f := range group {
			if strings.Contains(filepath.Base(f), "-arm64-") {
				chosen = f
				break
			}
		}
		out = append(out, chosen)
	}
	return out, nil
}

// writeSidecar writes docs as appledocs.json next to metaPath.
func writeSidecar(metaPath string, docs *appledocs.Docs) (string, error) {
	out := filepath.Join(filepath.Dir(metaPath), appledocs.FileName)
	data, err := json.MarshalIndent(docs, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshalling sidecar: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(out, data, 0o644); err != nil {
		return "", fmt.Errorf("writing %s: %w", out, err)
	}
	return out, nil
}
