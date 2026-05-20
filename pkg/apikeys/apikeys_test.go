package apikeys

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestScan_ReadsApiKeyPrefixedEnvVars verifies the scanner reads exactly the
// ApiKey_<Provider> env vars exported by ~/api_keys.sh and ignores
// everything else. This is positive runtime evidence: we set known env vars,
// then assert the snapshot contains them and excludes other keys.
func TestScan_ReadsApiKeyPrefixedEnvVars(t *testing.T) {
	// Use real env vars rather than a mock; per CONST-035, unit tests that
	// mock the System Under Test are bluff tests. We set + unset around the
	// call so we exercise the real os.Environ() path.
	t.Setenv("ApiKey_TestProvider", "secret-value-123")
	t.Setenv("ApiKey_AnotherProvider", "key-xyz")
	t.Setenv("UNRELATED_VAR", "should-not-appear")
	t.Setenv("ApiKey_Empty", "") // empty value must be excluded — represents an unset key

	snap := Scan()

	assert.True(t, snap.Has("TestProvider"), "TestProvider key should be detected")
	assert.True(t, snap.Has("AnotherProvider"), "AnotherProvider key should be detected")
	assert.False(t, snap.Has("UNRELATED_VAR"), "unrelated env var must not appear")
	assert.False(t, snap.Has("Empty"), "empty-value key must not appear (operator deliberately unset it)")
	assert.False(t, snap.Has("nonexistent"), "missing key must return false from Has")

	assert.Equal(t, "secret-value-123", snap.Get("TestProvider"))
	assert.Equal(t, "key-xyz", snap.Get("AnotherProvider"))
	assert.Equal(t, "", snap.Get("nonexistent"))
}

// TestScan_ProvidersSorted verifies Snapshot.Providers() returns a stable
// sorted list. Critical for deterministic UI rendering of "credentials
// available for: …" in tooling that consumes this package.
func TestScan_ProvidersSorted(t *testing.T) {
	t.Setenv("ApiKey_Zulu", "z")
	t.Setenv("ApiKey_Alpha", "a")
	t.Setenv("ApiKey_Mike", "m")

	snap := Scan()
	providers := snap.Providers()

	// Filter to just the ones we set (the host might have many others).
	var ours []string
	for _, p := range providers {
		if p == "Zulu" || p == "Alpha" || p == "Mike" {
			ours = append(ours, p)
		}
	}
	require := []string{"Alpha", "Mike", "Zulu"}
	assert.Equal(t, require, ours, "Providers() must return sorted output")
	assert.True(t, sort.StringsAreSorted(providers), "full snapshot must be sorted")
}

// TestScan_OperatorApiKeysShFormat documents the exact format the operator's
// ~/api_keys.sh uses (ApiKey_HuggingFace, ApiKey_Nvidia, …) and asserts the
// scanner reads them correctly. This is the anchor test that locks the
// convention so a future drift (e.g., someone renaming Prefix) is caught
// immediately.
func TestScan_OperatorApiKeysShFormat(t *testing.T) {
	// Sample of names from ~/api_keys.sh (see the operator's home dir):
	// ApiKey_HuggingFace, ApiKey_Nvidia, ApiKey_Mistral_AiStudio, …
	t.Setenv("ApiKey_HuggingFace", "hf_xxx")
	t.Setenv("ApiKey_Mistral_AiStudio", "ms_yyy")
	t.Setenv("ApiKey_Cloudflare_Workers_AI", "cf_zzz")

	snap := Scan()

	// All three must round-trip through Snapshot.Get(), proving the
	// scanner handles names with underscores (Mistral_AiStudio,
	// Cloudflare_Workers_AI) — historically a class-of-bug area.
	assert.Equal(t, "hf_xxx", snap.Get("HuggingFace"))
	assert.Equal(t, "ms_yyy", snap.Get("Mistral_AiStudio"))
	assert.Equal(t, "cf_zzz", snap.Get("Cloudflare_Workers_AI"))
}

// TestPrefixConstant locks the canonical prefix string. If anyone changes
// Prefix away from "ApiKey_", this test fails and the change is forced to
// be explicit (the value is shared with the operator's ~/api_keys.sh and
// with LLMsVerifier's challenge scripts).
func TestPrefixConstant(t *testing.T) {
	assert.Equal(t, "ApiKey_", Prefix,
		"Prefix is the canonical convention used by ~/api_keys.sh and LLMsVerifier — do not change without coordinating both")
}
