//go:build darwin

package idiomatic

import (
	"go/token"
	"strings"
	"unicode"
)

// commonInitialisms is the set of words that idiomatic Go writes in all capitals.
// It mirrors the Go community list (the set golang.org/x/lint enforces) and adds
// the initialisms that appear in Apple's macOS SDK class and selector names. The
// keys are the all-uppercase form; lookup upper-cases the candidate word first.
//
// Applying this set is what turns a naive PascalCase conversion such as
// "UsbControllers" (from the selector usbControllers) into the idiomatic
// "USBControllers": the bridge dispatches by selector, so renaming the exported
// Go method changes only the Go-facing name, never the Objective-C call.
var commonInitialisms = map[string]bool{
	// Go community standard (x/lint commonInitialisms).
	"ACL": true, "API": true, "ASCII": true, "CPU": true, "CSS": true,
	"DNS": true, "EOF": true, "GUID": true, "HTML": true, "HTTP": true,
	"HTTPS": true, "ID": true, "IP": true, "JSON": true, "LHS": true,
	"QPS": true, "RAM": true, "RHS": true, "RPC": true, "SLA": true,
	"SMTP": true, "SQL": true, "SSH": true, "TCP": true, "TLS": true,
	"TTL": true, "UDP": true, "UI": true, "UID": true, "UUID": true,
	"URI": true, "URL": true, "UTF8": true, "VM": true, "XML": true,
	"XMPP": true, "XSRF": true, "XSS": true,
	// Apple macOS SDK initialisms seen in Virtualization and related frameworks.
	"USB": true, "EFI": true, "NVM": true, "NVME": true, "MAC": true,
	"PCI": true, "HID": true, "OS": true, "IO": true, "GPU": true,
	"VNC": true, "DMA": true, "BSD": true, "GCD": true, "SPICE": true,
}

// splitWords breaks a MixedCaps (or mixedCaps) identifier into its component
// words by case-transition boundaries, with no regular expressions:
//
//   - a lower→upper transition starts a new word ("kernelURL" → kernel, URL);
//   - inside an upper run, an upper letter immediately followed by a lower letter
//     starts a new word, so the trailing capital belongs to the next word
//     ("URLString" → URL, String; "USBController" → USB, Controller);
//   - a digit stays attached to the word it follows.
//
// A word here is a maximal run as described; callers re-case each word.
func splitWords(name string) []string {
	runes := []rune(name)
	var words []string
	start := 0
	for i := 1; i < len(runes); i++ {
		prev, cur := runes[i-1], runes[i]
		boundary := false
		switch {
		case unicode.IsLower(prev) && unicode.IsUpper(cur):
			boundary = true
		case unicode.IsUpper(prev) && unicode.IsUpper(cur) &&
			i+1 < len(runes) && unicode.IsLower(runes[i+1]):
			boundary = true
		case unicode.IsDigit(prev) && unicode.IsUpper(cur):
			// A digit run ends a word, so the following capital starts the next
			// one ("UTF8String" → UTF8, String; "ISO8601Date" → ISO8601, Date).
			boundary = true
		}
		if boundary {
			words = append(words, string(runes[start:i]))
			start = i
		}
	}
	words = append(words, string(runes[start:]))
	return words
}

// applyInitialisms re-cases the words of an exported Go identifier so any word
// that names a known initialism is written in all capitals (USBController, not
// UsbController). Words that are not initialisms keep their original casing, so
// the function is a no-op for names with no initialism component.
func applyInitialisms(name string) string {
	if name == "" {
		return name
	}
	words := splitWords(name)
	for i, w := range words {
		if commonInitialisms[strings.ToUpper(w)] {
			words[i] = strings.ToUpper(w)
		}
	}
	return strings.Join(words, "")
}

// receiverName derives a short, type-reflecting receiver variable from a wrapper
// type name: the lower-cased leading letter of each word (LinuxBootLoader → lbl,
// VirtualMachine → vm, MACAddress → ma). Words come from both underscores and
// case/digit transitions, so an underscore-laden generated type
// (_cp_layer_renderer_capabilities → clrc) still yields a valid identifier rather
// than the blank identifier. It avoids Go keywords, the package aliases the
// generated bodies reference, and common parameter names, falling back to a
// trailing underscore so the receiver never shadows one of them.
func receiverName(goTypeName string) string {
	var sb strings.Builder
	for segment := range strings.SplitSeq(goTypeName, "_") {
		for _, w := range splitWords(segment) {
			for _, r := range w {
				if unicode.IsLetter(r) {
					sb.WriteRune(unicode.ToLower(r))
					break
				}
			}
		}
	}
	name := sb.String()
	if name == "" {
		name = "x"
	}
	// A trailing underscore keeps the receiver a valid, distinct identifier when
	// the initials would otherwise be a Go keyword (ISO8601DateFormatter → "if"),
	// shadow one of the package aliases the generated bodies reference, or collide
	// with a parameter name the generated signatures commonly use.
	if token.Lookup(name).IsKeyword() {
		return name + "_"
	}
	switch name {
	case "obj", "rt", "errkit", "objref", "purego", "objc", "ebipurego", "unsafe", "context",
		"ctx", "id", "other", "name", "key", "value", "items", "v":
		return name + "_"
	}
	return name
}

// uniqueReceiver returns the receiver variable for a method on goTypeName, with a
// trailing underscore appended as needed so it does not collide with any of the
// method's own parameter names (a wrapper type whose initials match a parameter,
// e.g. receiver "l" for Layer against a "fromLayer:" argument also named "l").
func uniqueReceiver(goTypeName string, paramNames []string) string {
	taken := make(map[string]bool, len(paramNames))
	for _, n := range paramNames {
		taken[n] = true
	}
	name := receiverName(goTypeName)
	for taken[name] {
		name += "_"
	}
	return name
}
