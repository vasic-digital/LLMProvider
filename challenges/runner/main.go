// Command runner is the LLMProvider round-292 Challenge runner
// (vasic-digital twin; mirrors the HelixDevelopment twin enriched
// in round-276).
//
// It exercises the real circuit.CircuitBreaker, real
// health.HealthMonitor, and real retry.RetryConfig surfaces across
// five locale fixtures. Every PASS line is backed by a runtime
// invariant — never a metadata-only check
// (CONST-035 / Article XI §11.9, CONST-050(A) paired mutation
// per §1.1).
//
// Anti-bluff invariants enforced:
//
//  1. CircuitBreaker initial state is Closed (default state
//     contract).
//  2. CircuitBreaker.IsClosed() == true on construction;
//     IsOpen() == false.
//  3. After N=FailureThreshold consecutive failures, the breaker
//     transitions Closed -> Open and short-circuits subsequent
//     Complete() calls with ErrCircuitOpen (real fault-tolerance
//     behaviour, not a stubbed exit code).
//  4. HealthMonitor.GetHealth() reports Unknown for a freshly
//     registered provider (no false "healthy" before first
//     probe).
//  5. HealthMonitor.RecordFailure() flips the cached health
//     status from Unknown towards Unhealthy after
//     UnhealthyThreshold hits, and the listener fires with
//     old/new states.
//  6. RetryConfig.IsRetryableStatusCode covers the documented
//     HTTP status set (429/500/502/503/504) and rejects
//     200/400/404.
//  7. RetryConfig.CalculateBackoff respects InitialDelay,
//     Multiplier and MaxDelay (no infinite backoff growth).
//
// Mutation hook: when env LLMPROVIDER_MUTATE_RUNNER=1 is set, the
// runner inverts invariant (3) (treats a breaker that REMAINED
// CLOSED after 5 forced failures as PASS instead of FAIL).
// Paired Challenge wrapper asserts the runner exits non-zero
// under mutation, which the wrapper rewrites to exit 99
// (paired-mutation success).
//
// Verbatim 2026-05-19 operator mandate (preserved per
// CONST-049 §11.4.17):
//
//	"all existing tests and Challenges do work in anti-bluff
//	manner - they MUST confirm that all tested codebase really
//	works as expected! We had been in position that all tests
//	do execute with success and all Challenges as well, but
//	in reality the most of the features does not work and
//	can't be used! This MUST NOT be the case and execution
//	of tests and Challenges MUST guarantee the quality, the
//	completition and full usability by end users of the
//	product!"
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"digital.vasic.llmprovider/pkg/circuit"
	"digital.vasic.llmprovider/pkg/health"
	"digital.vasic.llmprovider/pkg/models"
	"digital.vasic.llmprovider/pkg/retry"
)

// fixture is a 7-field projection of challenges/fixtures/<locale>.yaml.
// Minimal in-process parse keeps the runner free of new yaml deps
// (CONST-051(B): no transitive deps creeping into a reusable
// submodule).
type fixture struct {
	locale                    string
	providerID                string
	prompt                    string
	expectCircuitStateInitial string
	expectFailureThreshold    int
	expectHealthStatusInitial string
	expectRetryInitialDelayMS int
}

// controllableProvider is the smallest concrete LLMProvider that
// satisfies the package interface — used here ONLY to drive the
// real circuit.CircuitBreaker through its failure-budget state
// machine. Production providers (Ollama, OpenAI, …) live in
// pkg/providers/*; this stub exists exclusively to flip-flop the
// breaker and prove the breaker REALLY observes the failures it
// claims to count. The flag is atomic so concurrent breaker
// probes never race.
type controllableProvider struct {
	id        string
	failCount atomic.Int32 // increments per Complete call
	shouldErr atomic.Bool
}

func (p *controllableProvider) Complete(
	ctx context.Context, req *models.LLMRequest,
) (*models.LLMResponse, error) {
	p.failCount.Add(1)
	if p.shouldErr.Load() {
		return nil, errors.New("controllable provider: induced failure")
	}
	return &models.LLMResponse{
		ID: "ok", RequestID: req.ID, ProviderID: p.id,
		Content: "ok",
	}, nil
}

func (p *controllableProvider) CompleteStream(
	ctx context.Context, req *models.LLMRequest,
) (<-chan *models.LLMResponse, error) {
	ch := make(chan *models.LLMResponse, 1)
	close(ch)
	return ch, nil
}

func (p *controllableProvider) HealthCheck() error {
	if p.shouldErr.Load() {
		return errors.New("controllable provider: unhealthy")
	}
	return nil
}

func (p *controllableProvider) GetCapabilities() *models.ProviderCapabilities {
	return &models.ProviderCapabilities{
		SupportedModels: []string{p.id + "-model"},
	}
}

func (p *controllableProvider) ValidateConfig(
	_ map[string]interface{},
) (bool, []string) {
	return true, nil
}

func main() {
	if code := run(os.Stdout); code != 0 {
		os.Exit(code)
	}
}

func run(out io.Writer) int {
	fmt.Fprintln(out, "=== LLMProvider Challenge Runner (round-292) ===")

	fixDir := os.Getenv("LLMPROVIDER_FIXTURES_DIR")
	if fixDir == "" {
		fixDir = filepath.Join("challenges", "fixtures")
	}

	fixtures, err := loadFixtures(fixDir)
	if err != nil {
		fmt.Fprintf(out, "FAIL: load fixtures from %s: %v\n",
			fixDir, err)
		return 1
	}
	if len(fixtures) < 5 {
		fmt.Fprintf(out, "FAIL: expected >=5 fixtures, got %d\n",
			len(fixtures))
		return 1
	}
	fmt.Fprintf(out, "[setup] loaded %d locale fixtures from %s\n",
		len(fixtures), fixDir)

	mutate := os.Getenv("LLMPROVIDER_MUTATE_RUNNER") == "1"
	if mutate {
		fmt.Fprintln(out, "[setup] MUTATION MODE: invariant 3 polarity"+
			" inverted (PASS when breaker stays CLOSED after failures)")
	}

	pass, fail := 0, 0
	step := func(name string, ok bool, detail string) {
		if ok {
			pass++
			fmt.Fprintf(out, "  PASS  %-52s  %s\n", name, detail)
			return
		}
		fail++
		fmt.Fprintf(out, "  FAIL  %-52s  %s\n", name, detail)
	}

	// Run invariants 1-3 once per fixture against a fresh
	// breaker (real circuit.CircuitBreaker, not a stub).
	for _, f := range fixtures {
		p := &controllableProvider{id: f.providerID}
		cfg := circuit.DefaultCircuitBreakerConfig()
		cfg.FailureThreshold = f.expectFailureThreshold
		cb := circuit.NewCircuitBreaker(f.providerID, p, cfg)

		// Invariant 1: initial state.
		initialState := string(cb.GetState())
		step("circuit.initial_state."+f.locale,
			initialState == f.expectCircuitStateInitial,
			fmt.Sprintf("want=%s got=%s",
				f.expectCircuitStateInitial, initialState))

		// Invariant 2: IsClosed/IsOpen helpers agree with state.
		step("circuit.is_closed."+f.locale,
			cb.IsClosed() && !cb.IsOpen(),
			fmt.Sprintf("IsClosed=%v IsOpen=%v",
				cb.IsClosed(), cb.IsOpen()))

		// Invariant 3: force FailureThreshold failures →
		// state must transition to Open. Use real .Complete()
		// path so the breaker observes failures the same way
		// production traffic would.
		p.shouldErr.Store(true)
		ctx, cancel := context.WithTimeout(
			context.Background(), 5*time.Second)
		for i := 0; i < f.expectFailureThreshold; i++ {
			_, _ = cb.Complete(ctx, &models.LLMRequest{
				ID:     f.providerID + "-fail",
				Prompt: f.prompt})
		}
		cancel()
		// Drive one more call after threshold; should be
		// short-circuited with ErrCircuitOpen.
		_, errOpen := cb.Complete(context.Background(),
			&models.LLMRequest{
				ID: f.providerID + "-after", Prompt: f.prompt})
		isOpen := cb.IsOpen()
		shortCircuited := errors.Is(errOpen, circuit.ErrCircuitOpen)

		// Combined invariant for invariant-3: both must hold.
		invariantOK := isOpen && shortCircuited
		if mutate {
			// Mutation flips polarity: PASS when breaker is
			// NOT open (proves the runner really observes).
			step("circuit.opens_after_failures."+f.locale+"[MUTATED]",
				!invariantOK,
				fmt.Sprintf("mutation-inverted: isOpen=%v"+
					" shortCircuited=%v", isOpen,
					shortCircuited))
		} else {
			step("circuit.opens_after_failures."+f.locale,
				invariantOK,
				fmt.Sprintf("isOpen=%v shortCircuited=%v"+
					" err=%v", isOpen, shortCircuited,
					errOpen))
		}
	}

	// Invariants 4-5: HealthMonitor lifecycle. One monitor
	// shared across all fixtures — proves multi-provider
	// registration works.
	hmCfg := health.DefaultHealthMonitorConfig()
	hmCfg.UnhealthyThreshold = 2
	hmCfg.Enabled = false // disable background loop in runner
	hm := health.NewHealthMonitor(hmCfg)
	statusFlips := atomic.Int32{}
	hm.AddListener(func(_ string, _, _ health.HealthStatus) {
		statusFlips.Add(1)
	})

	for _, f := range fixtures {
		p := &controllableProvider{id: f.providerID}
		hm.RegisterProvider(f.providerID, p)
		got, ok := hm.GetHealth(f.providerID)
		step("health.initial_status."+f.locale,
			ok && string(got.Status) == f.expectHealthStatusInitial,
			fmt.Sprintf("ok=%v status=%s want=%s",
				ok, got.Status, f.expectHealthStatusInitial))
	}

	// Invariant 5: drive RecordFailure past threshold for first
	// fixture and confirm the listener observed a transition.
	first := fixtures[0]
	for i := 0; i < hmCfg.UnhealthyThreshold+1; i++ {
		hm.RecordFailure(first.providerID,
			errors.New("induced"))
	}
	// allow listener goroutine to run
	time.Sleep(50 * time.Millisecond)
	postGot, _ := hm.GetHealth(first.providerID)
	step("health.transitions_after_failures",
		postGot != nil &&
			string(postGot.Status) == string(health.HealthStatusUnhealthy) &&
			statusFlips.Load() > 0,
		fmt.Sprintf("status=%s flips=%d",
			postGot.Status, statusFlips.Load()))

	// Invariant 6: retry.IsRetryableStatusCode covers documented set.
	wantRetryable := []int{
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout,
	}
	wantNonRetryable := []int{
		http.StatusOK,
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusNotFound,
	}
	allRetryable := true
	for _, c := range wantRetryable {
		if !retry.IsRetryableStatusCode(c) {
			allRetryable = false
			break
		}
	}
	noneRetryable := true
	for _, c := range wantNonRetryable {
		if retry.IsRetryableStatusCode(c) {
			noneRetryable = false
			break
		}
	}
	step("retry.is_retryable_status_code",
		allRetryable && noneRetryable,
		fmt.Sprintf("retryable_all_ok=%v non_retryable_all_ok=%v",
			allRetryable, noneRetryable))

	// Invariant 7: backoff bounds.
	rCfg := retry.RetryConfig{
		MaxRetries:   5,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
		JitterFactor: 0.0,
	}
	b0 := retry.CalculateBackoff(0, rCfg)
	b3 := retry.CalculateBackoff(3, rCfg)
	bBig := retry.CalculateBackoff(20, rCfg)
	step("retry.calculate_backoff_bounds",
		b0 == rCfg.InitialDelay &&
			b3 > b0 &&
			bBig <= rCfg.MaxDelay,
		fmt.Sprintf("b0=%s b3=%s bBig=%s max=%s",
			b0, b3, bBig, rCfg.MaxDelay))

	fmt.Fprintf(out, "\n=== Summary: PASS=%d FAIL=%d ===\n",
		pass, fail)
	if fail > 0 {
		return 1
	}
	return 0
}

// loadFixtures parses every *.yaml in dir using a tiny line-based
// parser. We only support the 7 keys our fixtures use; anything
// else is ignored. Keeping the parser in-runner avoids pulling
// yaml.v3 into the runtime path of a submodule that other projects
// reuse (CONST-051(B)).
func loadFixtures(dir string) ([]fixture, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []fixture
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", e.Name(), err)
		}
		f := parseFixture(string(data))
		if f.locale == "" {
			return nil, fmt.Errorf(
				"%s: missing locale key", e.Name())
		}
		out = append(out, f)
	}
	return out, nil
}

func parseFixture(text string) fixture {
	f := fixture{}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		colon := strings.Index(line, ":")
		if colon < 0 {
			continue
		}
		k := strings.TrimSpace(line[:colon])
		v := strings.TrimSpace(line[colon+1:])
		v = strings.Trim(v, "\"'")
		switch k {
		case "locale":
			f.locale = v
		case "provider_id":
			f.providerID = v
		case "prompt":
			f.prompt = v
		case "expect_circuit_state_initial":
			f.expectCircuitStateInitial = v
		case "expect_failure_threshold":
			if n, err := strconv.Atoi(v); err == nil {
				f.expectFailureThreshold = n
			}
		case "expect_health_status_initial":
			f.expectHealthStatusInitial = v
		case "expect_retry_initial_delay_ms":
			if n, err := strconv.Atoi(v); err == nil {
				f.expectRetryInitialDelayMS = n
			}
		}
	}
	return f
}
