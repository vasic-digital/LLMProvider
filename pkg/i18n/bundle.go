// bundle.go provides a YAML-backed Translator implementation for
// LLMProvider's CONST-046 i18n seam (round-338 §11.4).
//
// CONST-051(B): the loader is project-not-aware. It accepts an
// io.Reader or a []byte so the consuming binary owns the file path
// — the package never reaches into a parent project's tree.
package i18n

import (
	"context"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

// BundleTranslator resolves message IDs against a flat
// messageID -> template map loaded from a YAML bundle. Placeholder
// interpolation uses Go's text/template-free %!(name) substitution:
// occurrences of "{{name}}" in the template are replaced with the
// corresponding templateData value.
type BundleTranslator struct {
	messages map[string]string
}

// NewBundleTranslatorFromBytes parses a YAML message bundle into a
// BundleTranslator. Round-341 merge-first reconciliation (CONST-061):
// the bundle reconciles two LLMProvider i18n lineages that adopted
// different YAML dialects — round-338's flat form (messageID:
// "template") and round-337's go-i18n nested form (messageID:\n
// other: "template"). This loader accepts BOTH so neither lineage's
// migrated entries are lost in the union.
func NewBundleTranslatorFromBytes(data []byte) (*BundleTranslator, error) {
	// First parse into a generic map so each value can be either a
	// scalar string (flat dialect) or a nested map carrying an
	// "other" key (go-i18n dialect).
	var generic map[string]any
	if err := yaml.Unmarshal(data, &generic); err != nil {
		return nil, fmt.Errorf("i18n: parse bundle: %w", err)
	}
	raw := make(map[string]string, len(generic))
	for id, val := range generic {
		switch v := val.(type) {
		case string:
			raw[id] = v
		case map[string]any:
			// go-i18n plural form: take the "other" entry.
			if other, ok := v["other"].(string); ok {
				raw[id] = other
				continue
			}
			return nil, fmt.Errorf("i18n: message id %q: nested entry missing string \"other\" key", id)
		case nil:
			raw[id] = ""
		default:
			return nil, fmt.Errorf("i18n: message id %q: unsupported value type %T", id, val)
		}
	}
	return &BundleTranslator{messages: raw}, nil
}

// NewBundleTranslatorFromReader reads a flat YAML map from r and
// parses it into a BundleTranslator.
func NewBundleTranslatorFromReader(r io.Reader) (*BundleTranslator, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("i18n: read bundle: %w", err)
	}
	return NewBundleTranslatorFromBytes(data)
}

// T resolves messageID against the loaded bundle. An unknown
// messageID returns an error so the caller can fall back to the
// loud NoopTranslator echo via Tr.
func (b *BundleTranslator) T(_ context.Context, messageID string, templateData map[string]any) (string, error) {
	tmpl, ok := b.messages[messageID]
	if !ok {
		return "", fmt.Errorf("i18n: unknown message id %q", messageID)
	}
	return interpolate(tmpl, templateData), nil
}

// interpolate replaces "{{key}}" occurrences in tmpl with the
// stringified templateData values. Keys absent from templateData
// are left untouched so a missing placeholder is visible.
func interpolate(tmpl string, templateData map[string]any) string {
	if len(templateData) == 0 {
		return tmpl
	}
	out := tmpl
	for k, v := range templateData {
		out = replaceAll(out, "{{"+k+"}}", fmt.Sprintf("%v", v))
	}
	return out
}

// replaceAll is a small dependency-free strings.ReplaceAll.
func replaceAll(s, old, new string) string {
	if old == "" {
		return s
	}
	var buf []byte
	for {
		idx := indexOf(s, old)
		if idx < 0 {
			buf = append(buf, s...)
			break
		}
		buf = append(buf, s[:idx]...)
		buf = append(buf, new...)
		s = s[idx+len(old):]
	}
	return string(buf)
}

// indexOf returns the first index of sub in s, or -1.
func indexOf(s, sub string) int {
	n := len(sub)
	if n == 0 {
		return 0
	}
	for i := 0; i+n <= len(s); i++ {
		if s[i:i+n] == sub {
			return i
		}
	}
	return -1
}
