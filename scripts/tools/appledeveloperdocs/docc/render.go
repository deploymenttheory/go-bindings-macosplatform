// Package docc models Apple's DocC "render JSON" (the structured data behind
// every developer.apple.com/documentation page) and projects it into the
// Objective-C variant so symbol titles read as ObjC selectors.
package docc

import (
	"encoding/json"
	"fmt"
	"strings"
)

// RenderNode is the subset of a DocC render-JSON document we consume.
type RenderNode struct {
	AbstractInline         []Inline             `json:"abstract"`
	PrimaryContentSections []ContentSection     `json:"primaryContentSections"`
	References             map[string]Reference `json:"references"`
	Metadata               Metadata             `json:"metadata"`
	Identifier             Identifier           `json:"identifier"`
	VariantOverrides       []VariantOverride    `json:"variantOverrides"`
}

// Metadata holds the symbol's own identity.
type Metadata struct {
	Title      string `json:"title"`
	Role       string `json:"role"`
	SymbolKind string `json:"symbolKind"`
}

// Identifier is the symbol's canonical doc:// URI.
type Identifier struct {
	URL string `json:"url"`
}

// VariantOverride is one language projection expressed as a JSON-Patch.
type VariantOverride struct {
	Traits []Trait   `json:"traits"`
	Patch  []PatchOp `json:"patch"`
}

// Trait identifies which interface language a variant override applies to.
type Trait struct {
	InterfaceLanguage string `json:"interfaceLanguage"`
}

// Reference is an entry in the references map: a symbol linked from this page,
// including the page's own child members (methods, properties, enum cases).
type Reference struct {
	Type       string     `json:"type"`
	Title      string     `json:"title"`
	Identifier string     `json:"identifier"`
	Abstract   []Inline   `json:"abstract"`
	Fragments  []Fragment `json:"fragments"`
	Role       string     `json:"role"`
	Kind       string     `json:"kind"`
	URL        string     `json:"url"`
}

// Fragment is one token of a symbol's declaration. In the ObjC variant the
// first fragment of a method is "- " (instance) or "+ " (class).
type Fragment struct {
	Kind string `json:"kind"`
	Text string `json:"text"`
}

// ContentSection is a primary-content block; we read the "content" kind for the
// Discussion/Overview prose.
type ContentSection struct {
	Kind    string         `json:"kind"`
	Content []BlockElement `json:"content"`
}

// BlockElement is a block-level content node (paragraph, heading, …).
type BlockElement struct {
	Type          string   `json:"type"`
	Text          string   `json:"text"`
	InlineContent []Inline `json:"inlineContent"`
}

// Inline is an inline content node (text, codeVoice, reference, emphasis, …).
type Inline struct {
	Type          string   `json:"type"`
	Text          string   `json:"text"`
	Code          string   `json:"code"`
	Identifier    string   `json:"identifier"`
	InlineContent []Inline `json:"inlineContent"`
}

// occLanguage is DocC's interface-language code for Objective-C.
const occLanguage = "occ"

// ParseObjC decodes DocC render JSON and projects it into the Objective-C
// variant by applying the matching variantOverrides JSON-Patch. When no ObjC
// override is present (Swift-only symbol) the base document is returned as-is.
func ParseObjC(data []byte) (*RenderNode, error) {
	// First decode just the overrides so we know whether a projection is needed.
	var head struct {
		VariantOverrides []VariantOverride `json:"variantOverrides"`
	}
	if err := json.Unmarshal(data, &head); err != nil {
		return nil, fmt.Errorf("decoding variantOverrides: %w", err)
	}

	var ops []PatchOp
	for _, ov := range head.VariantOverrides {
		for _, tr := range ov.Traits {
			if tr.InterfaceLanguage == occLanguage {
				ops = ov.Patch
			}
		}
	}

	if len(ops) > 0 {
		var raw any
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("decoding render tree: %w", err)
		}
		patched, err := ApplyPatch(raw, ops)
		if err != nil {
			return nil, err
		}
		data, err = json.Marshal(patched)
		if err != nil {
			return nil, fmt.Errorf("re-encoding patched tree: %w", err)
		}
	}

	var node RenderNode
	if err := json.Unmarshal(data, &node); err != nil {
		return nil, fmt.Errorf("decoding render node: %w", err)
	}
	return &node, nil
}

// Abstract returns the symbol's one-line summary as plain text.
func (n *RenderNode) Abstract() string {
	return flattenInline(n.AbstractInline, n.References)
}

// Discussion returns the symbol's Overview/Discussion paragraphs as plain text,
// paragraphs separated by blank lines. Non-paragraph blocks are skipped.
func (n *RenderNode) Discussion() string {
	var paras []string
	for _, section := range n.PrimaryContentSections {
		if section.Kind != "content" {
			continue
		}
		for _, block := range section.Content {
			if block.Type != "paragraph" {
				continue
			}
			if text := flattenInline(block.InlineContent, n.References); text != "" {
				paras = append(paras, text)
			}
		}
	}
	return strings.Join(paras, "\n\n")
}

// IsClassMethod reports whether this reference's ObjC declaration is a class
// method ("+ " leading fragment) rather than an instance method ("- ").
func (r Reference) IsClassMethod() bool {
	if len(r.Fragments) == 0 {
		return false
	}
	return strings.TrimSpace(r.Fragments[0].Text) == "+"
}

// HasMethodFragment reports whether this reference looks like an ObjC method
// declaration (leading "- " or "+ " fragment), distinguishing methods from
// properties/constants that share the references map.
func (r Reference) HasMethodFragment() bool {
	if len(r.Fragments) == 0 {
		return false
	}
	lead := strings.TrimSpace(r.Fragments[0].Text)
	return lead == "+" || lead == "-"
}

// AbstractText returns this reference's summary as plain text.
func (r Reference) AbstractText(refs map[string]Reference) string {
	return flattenInline(r.Abstract, refs)
}

// flattenInline renders a DocC inline-content array to clean single-line plain
// text suitable for a Go // comment: references resolve to their link title,
// markup is unwrapped, and runs of whitespace collapse to single spaces.
func flattenInline(content []Inline, refs map[string]Reference) string {
	var sb strings.Builder
	writeInline(&sb, content, refs)
	return collapseSpaces(sb.String())
}

func writeInline(sb *strings.Builder, content []Inline, refs map[string]Reference) {
	for _, c := range content {
		switch c.Type {
		case "text":
			sb.WriteString(c.Text)
		case "codeVoice":
			if c.Code != "" {
				sb.WriteString(c.Code)
			} else {
				sb.WriteString(c.Text)
			}
		case "reference":
			if ref, ok := refs[c.Identifier]; ok && ref.Title != "" {
				sb.WriteString(ref.Title)
			}
		case "emphasis", "strong", "newTerm", "superscript", "subscript", "strikethrough":
			writeInline(sb, c.InlineContent, refs)
		default:
			// Unknown inline kinds: descend if they wrap content, else drop.
			writeInline(sb, c.InlineContent, refs)
		}
	}
}

func collapseSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
