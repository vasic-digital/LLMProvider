// main_test.go — paired-mutation guard for the round-292 Challenge
// runner (CONST-050(A) §1.1). The runner ships with a mutation hook
// (LLMPROVIDER_MUTATE_RUNNER=1) that inverts invariant 3 polarity.
// If a future edit silently removes the mutation logic, the runner
// would exit 0 under mutation and the paired Challenge wrapper would
// no longer guard anything. These tests pin both branches.
package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestRun_NormalMode_AllPass(t *testing.T) {
	// runner reads fixtures from challenges/fixtures by default;
	// override to absolute path so `go test` from the runner's
	// own directory still finds them.
	if err := os.Setenv("LLMPROVIDER_FIXTURES_DIR",
		"../fixtures"); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	t.Cleanup(func() { _ = os.Unsetenv("LLMPROVIDER_FIXTURES_DIR") })
	_ = os.Unsetenv("LLMPROVIDER_MUTATE_RUNNER")

	var buf bytes.Buffer
	code := run(&buf)
	if code != 0 {
		t.Fatalf("run() = %d, want 0; output:\n%s",
			code, buf.String())
	}
	if !strings.Contains(buf.String(), "FAIL=0") {
		t.Fatalf("expected FAIL=0 in summary; got:\n%s",
			buf.String())
	}
	if !strings.Contains(buf.String(),
		"circuit.opens_after_failures.en") {
		t.Fatalf("expected en circuit invariant; got:\n%s",
			buf.String())
	}
}

func TestRun_MutationDetected(t *testing.T) {
	if err := os.Setenv("LLMPROVIDER_FIXTURES_DIR",
		"../fixtures"); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	if err := os.Setenv("LLMPROVIDER_MUTATE_RUNNER", "1"); err != nil {
		t.Fatalf("setenv mutate: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Unsetenv("LLMPROVIDER_FIXTURES_DIR")
		_ = os.Unsetenv("LLMPROVIDER_MUTATE_RUNNER")
	})

	var buf bytes.Buffer
	code := run(&buf)
	if code == 0 {
		t.Fatalf("mutation undetected: run() exited 0;"+
			" output:\n%s", buf.String())
	}
	// Must contain at least one [MUTATED] FAIL line — proves the
	// mutation hook actually engaged.
	if !strings.Contains(buf.String(), "[MUTATED]") {
		t.Fatalf("expected [MUTATED] marker in output; got:\n%s",
			buf.String())
	}
}

func TestParseFixture_HandlesAllSevenKeys(t *testing.T) {
	src := `locale: en
provider_id: "p1"
prompt: "do work"
expect_circuit_state_initial: closed
expect_failure_threshold: 3
expect_health_status_initial: unknown
expect_retry_initial_delay_ms: 100
# comment line ignored
unknown_key: foo
`
	f := parseFixture(src)
	if f.locale != "en" {
		t.Errorf("locale: got %q", f.locale)
	}
	if f.providerID != "p1" {
		t.Errorf("providerID: got %q", f.providerID)
	}
	if f.prompt != "do work" {
		t.Errorf("prompt: got %q", f.prompt)
	}
	if f.expectFailureThreshold != 3 {
		t.Errorf("expectFailureThreshold: got %d",
			f.expectFailureThreshold)
	}
	if f.expectRetryInitialDelayMS != 100 {
		t.Errorf("expectRetryInitialDelayMS: got %d",
			f.expectRetryInitialDelayMS)
	}
	if f.expectCircuitStateInitial != "closed" {
		t.Errorf("expectCircuitStateInitial: got %q",
			f.expectCircuitStateInitial)
	}
	if f.expectHealthStatusInitial != "unknown" {
		t.Errorf("expectHealthStatusInitial: got %q",
			f.expectHealthStatusInitial)
	}
}
