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

// NewBundleTranslatorFromBytes parses a flat YAML map (messageID:
// template) into a BundleTranslator.
func NewBundleTranslatorFromBytes(data []byte) (*BundleTranslator, error) {
	var raw map[string]string
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("i18n: parse bundle: %w", err)
	}
	if raw == nil {
		raw = map[string]string{}
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
