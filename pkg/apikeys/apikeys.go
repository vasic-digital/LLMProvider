// Package apikeys is the central authority within LLMProvider for resolving
// per-provider API credentials. It reads env-vars in the ApiKey_<Provider>
// convention exported by the operator's ~/api_keys.sh — the same convention
// LLMsVerifier uses (see its challenges/scripts/run_comprehensive_challenge.sh,
// where every provider's `api_key: "${ApiKey_<Provider>}"` placeholder is
// interpolated from the same env). This keeps Yole, LLMProvider, and
// LLMsVerifier on a single source of credential truth and prevents the
// "tests pass because hardcoded keys, product broken because real keys not
// wired" class of bluff.
//
// Usage:
//
//	scan := apikeys.Scan()
//	if scan.Has("HuggingFace") {
//	    key := scan.Get("HuggingFace")
//	    // use key …
//	}
//
// Per CONST-036, this package is the SOLE place inside LLMProvider that
// reads provider-credential env vars. Other code MUST go through Scan() or
// Get() rather than calling os.Getenv directly, so future audit changes
// (key rotation, alternative credential stores) need touch only one file.
package apikeys

import (
	"os"
	"sort"
	"strings"
)

// Prefix is the canonical credential-env-var prefix from ~/api_keys.sh.
const Prefix = "ApiKey_"

// Scan returns a snapshot of every ApiKey_<Provider> env var currently set
// on the process. Returned providers are stripped of the Prefix; values are
// the raw key string. The map is safe to mutate by the caller.
func Scan() Snapshot {
	out := make(Snapshot)
	for _, kv := range os.Environ() {
		idx := strings.IndexByte(kv, '=')
		if idx < 0 {
			continue
		}
		name := kv[:idx]
		val := kv[idx+1:]
		if !strings.HasPrefix(name, Prefix) {
			continue
		}
		provider := strings.TrimPrefix(name, Prefix)
		if provider == "" || val == "" {
			continue
		}
		out[provider] = val
	}
	return out
}

// Snapshot is the result of Scan(): provider-name (without Prefix) → key.
type Snapshot map[string]string

// Has reports whether a credential is set for the given provider name. The
// provider name MUST match the suffix used in ~/api_keys.sh (e.g.,
// "HuggingFace", "Nvidia", "Mistral_AiStudio"). Comparison is case-sensitive
// to match the env-var convention exactly.
func (s Snapshot) Has(provider string) bool {
	v, ok := s[provider]
	return ok && v != ""
}

// Get returns the credential value for the given provider, or "" if not set.
func (s Snapshot) Get(provider string) string {
	return s[provider]
}

// Providers returns the sorted list of provider names with credentials set.
// The slice is freshly allocated and safe to mutate.
func (s Snapshot) Providers() []string {
	out := make([]string, 0, len(s))
	for k := range s {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
