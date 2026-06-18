package docc

import (
	"fmt"
	"strconv"
	"strings"
)

// PatchOp is a single RFC-6902 JSON-Patch operation as it appears in a DocC
// variantOverrides entry. Only the operations DocC actually emits (add,
// replace, remove) are supported.
type PatchOp struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value any    `json:"value,omitempty"`
}

// ApplyPatch applies an RFC-6902 patch to a decoded JSON document (a
// map[string]any / []any tree) in place where possible, returning the new root.
// DocC uses this mechanism in variantOverrides to project the base (Swift)
// render JSON into the Objective-C view, which is where ObjC selectors live.
func ApplyPatch(doc any, ops []PatchOp) (any, error) {
	for i, op := range ops {
		var err error
		doc, err = applyOne(doc, op)
		if err != nil {
			return nil, fmt.Errorf("patch op %d (%s %s): %w", i, op.Op, op.Path, err)
		}
	}
	return doc, nil
}

func applyOne(doc any, op PatchOp) (any, error) {
	tokens, err := parsePointer(op.Path)
	if err != nil {
		return nil, err
	}
	switch op.Op {
	case "add", "replace":
		return setAt(doc, tokens, op.Value)
	case "remove":
		return removeAt(doc, tokens)
	default:
		// Unknown ops (move/copy/test) are not emitted by DocC; ignore them so a
		// future schema addition does not hard-fail the harvest.
		return doc, nil
	}
}

// parsePointer splits a JSON Pointer ("/a/b/0") into unescaped tokens.
func parsePointer(pointer string) ([]string, error) {
	if pointer == "" {
		return nil, nil
	}
	if !strings.HasPrefix(pointer, "/") {
		return nil, fmt.Errorf("invalid JSON pointer %q", pointer)
	}
	parts := strings.Split(pointer[1:], "/")
	for i, p := range parts {
		// Order matters: ~1 → "/" then ~0 → "~".
		p = strings.ReplaceAll(p, "~1", "/")
		p = strings.ReplaceAll(p, "~0", "~")
		parts[i] = p
	}
	return parts, nil
}

func setAt(doc any, tokens []string, value any) (any, error) {
	if len(tokens) == 0 {
		return value, nil
	}
	tok := tokens[0]
	last := len(tokens) == 1

	switch node := doc.(type) {
	case map[string]any:
		if last {
			node[tok] = value
			return node, nil
		}
		child, err := setAt(node[tok], tokens[1:], value)
		if err != nil {
			return nil, err
		}
		node[tok] = child
		return node, nil
	case []any:
		if tok == "-" {
			if !last {
				return nil, fmt.Errorf("cannot descend through append token %q", tok)
			}
			return append(node, value), nil
		}
		idx, err := strconv.Atoi(tok)
		if err != nil || idx < 0 || idx >= len(node) {
			return nil, fmt.Errorf("array index %q out of range (len %d)", tok, len(node))
		}
		if last {
			node[idx] = value
			return node, nil
		}
		child, err := setAt(node[idx], tokens[1:], value)
		if err != nil {
			return nil, err
		}
		node[idx] = child
		return node, nil
	default:
		return nil, fmt.Errorf("cannot index %T with %q", doc, tok)
	}
}

func removeAt(doc any, tokens []string) (any, error) {
	if len(tokens) == 0 {
		return nil, nil
	}
	tok := tokens[0]
	last := len(tokens) == 1

	switch node := doc.(type) {
	case map[string]any:
		if last {
			delete(node, tok)
			return node, nil
		}
		child, err := removeAt(node[tok], tokens[1:])
		if err != nil {
			return nil, err
		}
		node[tok] = child
		return node, nil
	case []any:
		idx, err := strconv.Atoi(tok)
		if err != nil || idx < 0 || idx >= len(node) {
			return nil, fmt.Errorf("array index %q out of range (len %d)", tok, len(node))
		}
		if last {
			return append(node[:idx], node[idx+1:]...), nil
		}
		child, err := removeAt(node[idx], tokens[1:])
		if err != nil {
			return nil, err
		}
		node[idx] = child
		return node, nil
	default:
		return nil, fmt.Errorf("cannot index %T with %q", doc, tok)
	}
}
