package main

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/appledocs"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
	"github.com/deploymenttheory/go-bindings-macosplatform/scripts/tools/appledeveloperdocs/docc"
)

// harvestStats summarises one framework's harvest for the run log.
type harvestStats struct {
	Pages    int // pages successfully fetched
	NotFound int // symbols with no published page
	Errors   int // fetch/parse failures (non-404)
	Classes  int
	Methods  int
	Props    int
	Enums    int
	Members  int
	Structs  int
	Protos   int
}

// harvester drives DocC extraction for a single framework, building the
// Apple-documentation sidecar from the symbols the metadata says we bind.
type harvester struct {
	client *docc.Client
	deep   bool
	logf   func(format string, args ...any)
}

// extract walks framework's metadata and fetches the matching Apple docs into a
// sidecar. It fetches one page per top-level symbol (class/enum/protocol/struct)
// and harvests child member abstracts (methods, properties, enum cases) out of
// each page's references map — so a class fetch documents the whole class.
func (h *harvester) extract(framework *macosplatformmetadata.FrameworkMeta) (*appledocs.Docs, harvestStats) {
	fwSlug := strings.ToLower(framework.Framework)
	var stats harvestStats

	docs := &appledocs.Docs{
		SchemaVersion: appledocs.SchemaVersion,
		Framework:     framework.Framework,
		Source:        "developer.apple.com DocC render API",
		Classes:       map[string]appledocs.SymbolDoc{},
		Protocols:     map[string]appledocs.SymbolDoc{},
		Enums:         map[string]appledocs.EnumDoc{},
		Structs:       map[string]appledocs.SymbolDoc{},
		Methods:       map[string]map[string]appledocs.SymbolDoc{},
		Properties:    map[string]map[string]appledocs.SymbolDoc{},
	}

	// Classes (+ methods + properties harvested from references).
	for _, className := range sortedKeys(framework.Classes) {
		path := fwSlug + "/" + strings.ToLower(className)
		node, err := h.fetch(path, &stats)
		if err != nil {
			continue
		}
		docs.Classes[className] = appledocs.SymbolDoc{Doc: h.docText(node), URL: docc.PageURL(path)}
		stats.Classes++

		methods := map[string]appledocs.SymbolDoc{}
		props := map[string]appledocs.SymbolDoc{}
		for _, ref := range node.References {
			if !isDirectChild(ref.Identifier, framework.Framework, className) {
				continue
			}
			abstract := ref.AbstractText(node.References)
			if abstract == "" {
				continue
			}
			if ref.HasMethodFragment() {
				key := appledocs.MethodKey(ref.Title, ref.IsClassMethod())
				methods[key] = appledocs.SymbolDoc{Doc: abstract, URL: absURL(ref.URL)}
			} else {
				props[ref.Title] = appledocs.SymbolDoc{Doc: abstract, URL: absURL(ref.URL)}
			}
		}
		if len(methods) > 0 {
			docs.Methods[className] = methods
			stats.Methods += len(methods)
		}
		if len(props) > 0 {
			docs.Properties[className] = props
			stats.Props += len(props)
		}
	}

	// Enums (+ members).
	for _, enumName := range sortedKeys(framework.Enums) {
		path := fwSlug + "/" + strings.ToLower(enumName)
		node, err := h.fetch(path, &stats)
		if err != nil {
			continue
		}
		entry := appledocs.EnumDoc{Doc: h.docText(node), URL: docc.PageURL(path), Members: map[string]appledocs.SymbolDoc{}}
		for _, ref := range node.References {
			if !isDirectChild(ref.Identifier, framework.Framework, enumName) {
				continue
			}
			abstract := ref.AbstractText(node.References)
			if abstract == "" {
				continue
			}
			entry.Members[ref.Title] = appledocs.SymbolDoc{Doc: abstract, URL: absURL(ref.URL)}
		}
		if len(entry.Members) == 0 {
			entry.Members = nil
		}
		docs.Enums[enumName] = entry
		stats.Enums++
		stats.Members += len(entry.Members)
	}

	// Structs.
	for _, structName := range sortedKeys(framework.Structs) {
		path := fwSlug + "/" + strings.ToLower(structName)
		node, err := h.fetch(path, &stats)
		if err != nil {
			continue
		}
		docs.Structs[structName] = appledocs.SymbolDoc{Doc: h.docText(node), URL: docc.PageURL(path)}
		stats.Structs++
	}

	// Protocols (stored for forward-compatibility; emitted once Protocol.Doc lands).
	for _, protoName := range sortedKeys(framework.Protocols) {
		path := fwSlug + "/" + strings.ToLower(protoName)
		node, err := h.fetch(path, &stats)
		if err != nil {
			continue
		}
		docs.Protocols[protoName] = appledocs.SymbolDoc{Doc: h.docText(node), URL: docc.PageURL(path)}
		stats.Protos++
	}

	pruneEmpty(docs)
	return docs, stats
}

// fetch wraps the client, folding 404s into a nil node and updating stats.
func (h *harvester) fetch(path string, stats *harvestStats) (*docc.RenderNode, error) {
	node, err := h.client.FetchObjC(path)
	switch {
	case errors.Is(err, docc.ErrNotFound):
		stats.NotFound++
		return nil, err
	case err != nil:
		stats.Errors++
		h.logf("  warn: %s: %v", path, err)
		return nil, err
	}
	stats.Pages++
	return node, nil
}

// docText returns the abstract, optionally followed by the Discussion in deep mode.
func (h *harvester) docText(node *docc.RenderNode) string {
	abstract := node.Abstract()
	if !h.deep {
		return abstract
	}
	discussion := node.Discussion()
	switch {
	case abstract == "":
		return discussion
	case discussion == "":
		return abstract
	default:
		return abstract + "\n\n" + discussion
	}
}

// isDirectChild reports whether identifier names a direct member of
// framework/parent, e.g. doc://…/documentation/Foundation/NSString/length.
func isDirectChild(identifier, framework, parent string) bool {
	const marker = "/documentation/"
	i := strings.Index(identifier, marker)
	if i < 0 {
		return false
	}
	segs := strings.Split(identifier[i+len(marker):], "/")
	if len(segs) != 3 {
		return false
	}
	return strings.EqualFold(segs[0], framework) && segs[1] == parent
}

// absURL turns a site-relative DocC URL ("/documentation/…") into an absolute one.
func absURL(u string) string {
	if strings.HasPrefix(u, "/") {
		return "https://developer.apple.com" + u
	}
	return u
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// pruneEmpty drops empty top-level maps so the sidecar JSON stays minimal.
func pruneEmpty(docs *appledocs.Docs) {
	if len(docs.Classes) == 0 {
		docs.Classes = nil
	}
	if len(docs.Protocols) == 0 {
		docs.Protocols = nil
	}
	if len(docs.Enums) == 0 {
		docs.Enums = nil
	}
	if len(docs.Structs) == 0 {
		docs.Structs = nil
	}
	if len(docs.Methods) == 0 {
		docs.Methods = nil
	}
	if len(docs.Properties) == 0 {
		docs.Properties = nil
	}
}

// describeStats renders a one-line summary.
func (s harvestStats) String() string {
	return fmt.Sprintf(
		"pages=%d notfound=%d errors=%d | classes=%d methods=%d props=%d enums=%d members=%d structs=%d protocols=%d",
		s.Pages, s.NotFound, s.Errors,
		s.Classes, s.Methods, s.Props, s.Enums, s.Members, s.Structs, s.Protos,
	)
}
