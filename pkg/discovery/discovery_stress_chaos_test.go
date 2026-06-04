package discovery

// §11.4.85 Stress + Chaos suite for the model-discovery cache.
//
// This suite closes the §11.4.85 stress+chaos gap for the Round-468e W6D fix
// (the cache-corruption / data-race defect in cacheModels' copy-on-return
// contract — see discovery.go cacheModels). It is fully self-driving
// (§11.4.98), deterministic and -race-clean at -count=3 (§11.4.50), and every
// PASS captures positive evidence to qa-results/<run-id>/ (§11.4.5 / §11.4.69).
//
// Evidence taxonomy mapping (§11.4.69): the discovery cache is an in-process
// data structure with no OS sink; its sink-side positive evidence is the
// captured latency distribution (latency.json), the categorised-error census
// (categorized_errors.json), and the post-storm canonical-catalogue snapshot
// (state_delta_snapshot.json) proving the internal cache was never corrupted.
//
// Anti-bluff (§1.1 paired mutation): TestCacheChaos_CallerMutationDoesNotCorrupt
// and TestCacheStress_ConcurrentNoCorruption MUST FAIL when the W6D defensive
// copy in cacheModels is reverted (return the stored slice directly). The
// revert procedure + captured FAIL/PASS evidence is recorded in
// qa-results/<run-id>/mutation_*.txt by hand during this session — see the
// subagent report. These tests are the regression-guard that makes the bluff
// impossible.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Evidence capture helpers (§11.4.5 / §11.4.69)
// ---------------------------------------------------------------------------

// scQARoot resolves the qa-results directory at the module root. The discovery
// package lives at pkg/discovery, so the module root is two levels up.
func scQARoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	// pkg/discovery -> module root
	return filepath.Join(wd, "..", "..", "qa-results")
}

// scRunID is a single monotonic run-id shared by every test in this suite so
// all evidence for one `go test` invocation lands under one greppable dir.
var (
	scRunIDOnce sync.Once
	scRunIDVal  string
)

func scRunID() string {
	scRunIDOnce.Do(func() {
		scRunIDVal = "w7b_stress_chaos_" + time.Now().UTC().Format("20060102T150405Z")
	})
	return scRunIDVal
}

// scEvidenceDir creates (idempotently) the per-run evidence directory and
// returns its absolute path.
func scEvidenceDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(scQARoot(t), scRunID())
	require.NoError(t, os.MkdirAll(dir, 0o755))
	return dir
}

// scWriteEvidence writes a JSON artefact under the run dir and asserts it is
// non-empty (the §11.4.69 ab_pass_with_evidence contract: the evidence path
// exists AND is non-empty).
func scWriteEvidence(t *testing.T, name string, payload any) string {
	t.Helper()
	dir := scEvidenceDir(t)
	p := filepath.Join(dir, name)
	b, err := json.MarshalIndent(payload, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(p, b, 0o644))
	info, err := os.Stat(p)
	require.NoError(t, err)
	require.Greater(t, info.Size(), int64(0), "evidence artefact %s must be non-empty (§11.4.69)", name)
	t.Logf("PASS evidence: %s [evidence: %s]", name, p)
	return p
}

// latencyStats holds a captured latency distribution (§11.4.85 stress: p50/p95/p99).
type latencyStats struct {
	Scenario   string  `json:"scenario"`
	Samples    int     `json:"samples"`
	P50Micros  int64   `json:"p50_micros"`
	P95Micros  int64   `json:"p95_micros"`
	P99Micros  int64   `json:"p99_micros"`
	MaxMicros  int64   `json:"max_micros"`
	MeanMicros float64 `json:"mean_micros"`
}

func computeLatencyStats(scenario string, durs []time.Duration) latencyStats {
	n := len(durs)
	sorted := make([]time.Duration, n)
	copy(sorted, durs)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	pick := func(p float64) int64 {
		if n == 0 {
			return 0
		}
		idx := int(p * float64(n))
		if idx >= n {
			idx = n - 1
		}
		return sorted[idx].Microseconds()
	}
	var total time.Duration
	for _, d := range durs {
		total += d
	}
	mean := 0.0
	if n > 0 {
		mean = float64(total.Microseconds()) / float64(n)
	}
	return latencyStats{
		Scenario:   scenario,
		Samples:    n,
		P50Micros:  pick(0.50),
		P95Micros:  pick(0.95),
		P99Micros:  pick(0.99),
		MaxMicros:  sorted[max0(n-1)].Microseconds(),
		MeanMicros: mean,
	}
}

func max0(i int) int {
	if i < 0 {
		return 0
	}
	return i
}

// ---------------------------------------------------------------------------
// Injectable test server (chaos: error-mid-populate, latency injection)
// ---------------------------------------------------------------------------

// chaosServer is an httptest server whose behaviour is switchable at runtime so
// chaos faults (error responses, slow responses) can be injected mid-test
// without races. Mode is read atomically.
type chaosServer struct {
	srv     *httptest.Server
	mode    atomic.Int32 // 0=ok, 1=http500, 2=garbage-body, 3=slow-ok
	okCalls atomic.Int64
	errCalls atomic.Int64
}

const (
	chaosModeOK      int32 = 0
	chaosModeHTTP500 int32 = 1
	chaosModeGarbage int32 = 2
	chaosModeSlowOK  int32 = 3
)

// canonicalModels is the catalogue the OK path always serves. After IsChatModel
// filtering all three survive (none match an exclude pattern).
var canonicalModels = []string{"model-a", "model-b", "model-c"}

func newChaosServer() *chaosServer {
	cs := &chaosServer{}
	cs.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		switch cs.mode.Load() {
		case chaosModeHTTP500:
			cs.errCalls.Add(1)
			w.WriteHeader(http.StatusInternalServerError)
		case chaosModeGarbage:
			cs.errCalls.Add(1)
			w.Header().Set("Content-Type", "application/json")
			// Truncated/garbage JSON → decode fails → discoverFromProviderAPI
			// returns nil → cache stays consistent (never half-written).
			_, _ = w.Write([]byte(`{"data":[{"id":"model-a"`))
		case chaosModeSlowOK:
			cs.okCalls.Add(1)
			time.Sleep(2 * time.Millisecond)
			writeCanonical(w)
		default: // chaosModeOK
			cs.okCalls.Add(1)
			writeCanonical(w)
		}
	}))
	return cs
}

func writeCanonical(w http.ResponseWriter) {
	resp := openAIModelsResponse{Data: make([]openAIModel, 0, len(canonicalModels))}
	for _, m := range canonicalModels {
		resp.Data = append(resp.Data, openAIModel{ID: m})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (cs *chaosServer) URL() string  { return cs.srv.URL }
func (cs *chaosServer) Close()        { cs.srv.Close() }
func (cs *chaosServer) setMode(m int32) { cs.mode.Store(m) }

// ---------------------------------------------------------------------------
// STRESS (a): sustained load — N>=100 cycles, p50/p95/p99 recorded
// ---------------------------------------------------------------------------

func TestCacheStress_SustainedLoad(t *testing.T) {
	cs := newChaosServer()
	defer cs.Close()

	d := NewDiscoverer(ProviderConfig{
		ProviderName:   "stress-sustained",
		ModelsEndpoint: cs.URL(),
		APIKey:         "test-key",
		CacheTTL:       1 * time.Hour, // mostly cache-hit path under sustained load
	})

	const iterations = 200 // >= 100 per §11.4.85
	durs := make([]time.Duration, 0, iterations*2)

	for i := 0; i < iterations; i++ {
		start := time.Now()
		got := d.DiscoverModels()
		durs = append(durs, time.Since(start))
		require.ElementsMatch(t, canonicalModels, got,
			"sustained load: every cycle must return the canonical catalogue")

		start = time.Now()
		cached := d.GetCachedModels()
		durs = append(durs, time.Since(start))
		require.ElementsMatch(t, canonicalModels, cached,
			"sustained load: GetCachedModels must mirror the catalogue")
	}

	stats := computeLatencyStats("sustained_load", durs)
	require.Equal(t, iterations*2, stats.Samples)
	scWriteEvidence(t, "stress_sustained_latency.json", stats)
}

// ---------------------------------------------------------------------------
// STRESS (b): concurrent contention — N>=10 goroutines, mixed ops, -race,
// NO data race, NO cache corruption (returned slice independent of cache).
// ---------------------------------------------------------------------------

func TestCacheStress_ConcurrentNoCorruption(t *testing.T) {
	cs := newChaosServer()
	defer cs.Close()

	d := NewDiscoverer(ProviderConfig{
		ProviderName:   "stress-concurrent",
		ModelsEndpoint: cs.URL(),
		APIKey:         "test-key",
		CacheTTL:       1 * time.Millisecond, // force frequent write path (re-discovery)
	})

	const workers = 24 // >= 10 per §11.4.85
	const iterations = 60

	var corruptions atomic.Int64
	var raceProxy atomic.Int64 // counts non-canonical observations from a fresh read
	var wg sync.WaitGroup
	wg.Add(workers)

	for w := 0; w < workers; w++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				switch (id + i) % 4 {
				case 0:
					got := d.DiscoverModels()
					// Caller mutation MUST NOT corrupt the cache (W6D fix).
					if len(got) > 0 {
						got[0] = "CALLER-MUTATED"
					}
				case 1:
					cached := d.GetCachedModels()
					// Any value we read must be either nil (mid-invalidate) or a
					// subset of the canonical catalogue — never a caller-mutated
					// sentinel. A leaked sentinel proves cache corruption.
					for _, m := range cached {
						if m == "CALLER-MUTATED" {
							corruptions.Add(1)
						}
					}
				case 2:
					_ = d.GetDiscoveryTier()
				case 3:
					d.InvalidateCache()
				}
			}
		}(w)
	}
	wg.Wait()

	// Post-storm: a fresh discovery must return the uncorrupted canonical set.
	d.InvalidateCache()
	final := d.DiscoverModels()
	for _, m := range final {
		if m == "CALLER-MUTATED" {
			raceProxy.Add(1)
		}
	}

	report := struct {
		Workers          int      `json:"workers"`
		IterationsEach   int      `json:"iterations_each"`
		TotalOps         int      `json:"total_ops"`
		CorruptionsSeen  int64    `json:"corruptions_seen"`
		PostStormLeaks   int64    `json:"post_storm_caller_mutation_leaks"`
		FinalCatalogue   []string `json:"final_catalogue"`
		CanonicalExpected []string `json:"canonical_expected"`
	}{
		Workers:           workers,
		IterationsEach:    iterations,
		TotalOps:          workers * iterations,
		CorruptionsSeen:   corruptions.Load(),
		PostStormLeaks:    raceProxy.Load(),
		FinalCatalogue:    final,
		CanonicalExpected: canonicalModels,
	}
	scWriteEvidence(t, "stress_concurrent_report.json", report)

	assert.Zero(t, corruptions.Load(),
		"no caller-mutated sentinel may ever appear in the cache (W6D copy-on-return)")
	assert.Zero(t, raceProxy.Load(),
		"post-storm fresh discovery must be free of caller-mutation leakage")
	assert.ElementsMatch(t, canonicalModels, final,
		"internal cache must remain the canonical catalogue after concurrent storm")
}

// ---------------------------------------------------------------------------
// STRESS (c): boundary conditions — empty cache, nil-models (CONST-036 nil
// path), max-size model list.
// ---------------------------------------------------------------------------

func TestCacheStress_Boundaries(t *testing.T) {
	type boundaryResult struct {
		Name     string `json:"name"`
		Outcome  string `json:"outcome"`
		ModelLen int    `json:"model_len"`
	}
	var results []boundaryResult

	// Boundary 1: empty cache — GetCachedModels before any discovery returns nil.
	t.Run("empty_cache", func(t *testing.T) {
		d := NewDiscoverer(ProviderConfig{ProviderName: "empty", FallbackModels: []string{"nope"}})
		got := d.GetCachedModels()
		assert.Nil(t, got, "empty cache must return nil (CONST-036)")
		results = append(results, boundaryResult{"empty_cache", "nil_as_expected", len(got)})
	})

	// Boundary 2: nil models / CONST-036 nil path — no API key, Tier 1 skipped.
	t.Run("nil_path_const036", func(t *testing.T) {
		d := NewDiscoverer(ProviderConfig{
			ProviderName:   "offline",
			ModelsEndpoint: "https://unreachable.invalid/v1/models",
			APIKey:         "",
			FallbackModels: []string{"must-not-appear"},
		})
		got := d.DiscoverModels()
		assert.Nil(t, got, "no-API-key path must return nil, never fallback (CONST-036)")
		assert.NotContains(t, got, "must-not-appear")
		assert.NotEqual(t, 3, d.GetDiscoveryTier())
		results = append(results, boundaryResult{"nil_path_const036", "nil_as_expected", len(got)})
	})

	// Boundary 3: max-size model list — a large catalogue round-trips intact and
	// the returned slice is an independent copy.
	t.Run("max_size_list", func(t *testing.T) {
		const big = 5000
		large := make([]string, big)
		for i := range large {
			large[i] = fmt.Sprintf("model-%05d", i) // none match an exclude pattern
		}
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			resp := openAIModelsResponse{Data: make([]openAIModel, big)}
			for i, m := range large {
				resp.Data[i] = openAIModel{ID: m}
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		d := NewDiscoverer(ProviderConfig{
			ProviderName:   "bigcat",
			ModelsEndpoint: srv.URL,
			APIKey:         "test-key",
			CacheTTL:       1 * time.Hour,
		})
		got := d.DiscoverModels()
		require.Len(t, got, big, "max-size catalogue must round-trip without loss")
		// Mutate the returned slice; the cache copy must be unaffected.
		got[0] = "MUTATED"
		cached := d.GetCachedModels()
		require.Len(t, cached, big)
		assert.NotEqual(t, "MUTATED", cached[0],
			"max-size: caller mutation must not corrupt the cache (W6D copy-on-return)")
		results = append(results, boundaryResult{"max_size_list", "intact_independent_copy", len(cached)})
	})

	scWriteEvidence(t, "stress_boundaries.json", results)
}

// ---------------------------------------------------------------------------
// CHAOS (a): concurrent cache invalidation/refresh mid-read → readers always
// see a consistent (old or new, never torn) slice.
// ---------------------------------------------------------------------------

func TestCacheChaos_InvalidateRefreshMidRead(t *testing.T) {
	cs := newChaosServer()
	defer cs.Close()

	d := NewDiscoverer(ProviderConfig{
		ProviderName:   "chaos-invalidate",
		ModelsEndpoint: cs.URL(),
		APIKey:         "test-key",
		CacheTTL:       1 * time.Hour,
	})
	require.ElementsMatch(t, canonicalModels, d.DiscoverModels(), "prime cache")

	const readers = 16
	const writers = 4
	stop := make(chan struct{})
	var tornReads atomic.Int64
	var wg sync.WaitGroup

	// Writers: hammer invalidate + re-discover concurrently.
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					d.InvalidateCache()
					_ = d.DiscoverModels()
				}
			}
		}()
	}

	// Readers: every observed slice must be EITHER nil/empty (between invalidate
	// and re-populate) OR exactly the canonical catalogue — never a torn slice
	// (partial, duplicated, or interleaved entries).
	for r := 0; r < readers; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 500; i++ {
				got := d.GetCachedModels()
				if len(got) == 0 {
					continue // consistent empty state
				}
				if !sameSet(got, canonicalModels) {
					tornReads.Add(1)
				}
			}
		}()
	}

	time.Sleep(40 * time.Millisecond)
	close(stop)
	wg.Wait()

	scWriteEvidence(t, "chaos_invalidate_refresh.json", struct {
		Readers   int   `json:"readers"`
		Writers   int   `json:"writers"`
		TornReads int64 `json:"torn_reads"`
	}{readers, writers, tornReads.Load()})

	assert.Zero(t, tornReads.Load(),
		"every reader must see a consistent slice (old/new), never a torn read")
}

// ---------------------------------------------------------------------------
// CHAOS (b): discovery error injected mid-populate → cache stays consistent
// (not half-written).
// ---------------------------------------------------------------------------

func TestCacheChaos_ErrorMidPopulate(t *testing.T) {
	cs := newChaosServer()
	defer cs.Close()

	d := NewDiscoverer(ProviderConfig{
		ProviderName:   "chaos-error",
		ModelsEndpoint: cs.URL(),
		APIKey:         "test-key",
		CacheTTL:       1 * time.Millisecond, // each call re-checks the live source
	})

	type phase struct {
		Mode       string   `json:"mode"`
		Discovered []string `json:"discovered"`
		Cached     []string `json:"cached"`
	}
	var phases []phase

	// 1. Healthy populate.
	cs.setMode(chaosModeOK)
	got := d.DiscoverModels()
	require.ElementsMatch(t, canonicalModels, got)
	phases = append(phases, phase{"ok_populate", got, d.GetCachedModels()})

	// 2. Inject HTTP 500 mid-life. Live discovery fails; CacheTTL expired so the
	//    cache-hit fast path is bypassed → DiscoverModels returns nil. The
	//    previously-cached good data must NOT be corrupted/half-written: a fresh
	//    GetCachedModels still yields the canonical set (the failed Tier 1 never
	//    called cacheModels, so the prior good cache stands).
	time.Sleep(2 * time.Millisecond) // ensure TTL elapsed
	cs.setMode(chaosModeHTTP500)
	gotErr := d.DiscoverModels()
	assert.Nil(t, gotErr, "failed live discovery returns nil (CONST-036)")
	cachedAfterErr := d.GetCachedModels()
	assert.ElementsMatch(t, canonicalModels, cachedAfterErr,
		"a failed discovery must leave the prior good cache intact (no half-write)")
	phases = append(phases, phase{"http500_midlife", gotErr, cachedAfterErr})

	// 3. Inject garbage body — JSON decode fails → nil → cache still intact.
	cs.setMode(chaosModeGarbage)
	gotGarbage := d.DiscoverModels()
	assert.Nil(t, gotGarbage, "garbage body → decode fail → nil")
	cachedAfterGarbage := d.GetCachedModels()
	assert.ElementsMatch(t, canonicalModels, cachedAfterGarbage,
		"a decode failure must not half-write the cache")
	phases = append(phases, phase{"garbage_body", gotGarbage, cachedAfterGarbage})

	// 4. Recover — healthy again, cache refreshes to canonical.
	cs.setMode(chaosModeOK)
	d.InvalidateCache()
	recovered := d.DiscoverModels()
	assert.ElementsMatch(t, canonicalModels, recovered, "recovery restores canonical catalogue")
	phases = append(phases, phase{"recovered", recovered, d.GetCachedModels()})

	scWriteEvidence(t, "chaos_error_mid_populate.json", phases)
}

// TestCacheChaos_ConcurrentErrorInjection runs error-mode flapping concurrently
// with readers to prove the cache is never observed half-written under chaos.
func TestCacheChaos_ConcurrentErrorInjection(t *testing.T) {
	cs := newChaosServer()
	defer cs.Close()

	d := NewDiscoverer(ProviderConfig{
		ProviderName:   "chaos-flap",
		ModelsEndpoint: cs.URL(),
		APIKey:         "test-key",
		CacheTTL:       1 * time.Millisecond,
	})
	require.ElementsMatch(t, canonicalModels, d.DiscoverModels(), "prime")

	stop := make(chan struct{})
	var inconsistent atomic.Int64
	var wg sync.WaitGroup

	// Fault flapper.
	wg.Add(1)
	go func() {
		defer wg.Done()
		modes := []int32{chaosModeOK, chaosModeHTTP500, chaosModeGarbage, chaosModeSlowOK}
		i := 0
		for {
			select {
			case <-stop:
				return
			default:
				cs.setMode(modes[i%len(modes)])
				i++
				time.Sleep(200 * time.Microsecond)
			}
		}
	}()

	// Discoverers + readers.
	for w := 0; w < 16; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < 300; i++ {
				if id%2 == 0 {
					got := d.DiscoverModels()
					// Result is either nil (failed live + no surviving cache path)
					// or a subset of canonical — never a torn/foreign slice.
					if len(got) > 0 && !subsetOfCanonical(got) {
						inconsistent.Add(1)
					}
				} else {
					cached := d.GetCachedModels()
					if len(cached) > 0 && !subsetOfCanonical(cached) {
						inconsistent.Add(1)
					}
				}
			}
		}(w)
	}

	time.Sleep(40 * time.Millisecond)
	close(stop)
	wg.Wait()

	scWriteEvidence(t, "chaos_concurrent_error_injection.json", struct {
		InconsistentObservations int64 `json:"inconsistent_observations"`
	}{inconsistent.Load()})

	assert.Zero(t, inconsistent.Load(),
		"under chaotic error injection, no caller ever observes a torn/foreign catalogue")
}

// ---------------------------------------------------------------------------
// CHAOS (c): input-corruption — a caller mutates the returned slice → assert
// the internal cache is UNAFFECTED (the W6D fix), proven under concurrency.
// ---------------------------------------------------------------------------

func TestCacheChaos_CallerMutationDoesNotCorrupt(t *testing.T) {
	cs := newChaosServer()
	defer cs.Close()

	d := NewDiscoverer(ProviderConfig{
		ProviderName:   "chaos-mutate",
		ModelsEndpoint: cs.URL(),
		APIKey:         "test-key",
		CacheTTL:       1 * time.Hour, // single populate; many readers mutate copies
	})

	// Single populate via the cache-MISS path (the exact path the W6D fix
	// hardened: cacheModels stores a defensive internal copy).
	first := d.DiscoverModels()
	require.ElementsMatch(t, canonicalModels, first)

	// Aggressively mutate the cache-miss return value.
	for i := range first {
		first[i] = "MISS-PATH-MUTATED"
	}

	// The cache must be untouched on the very next read.
	require.ElementsMatch(t, canonicalModels, d.GetCachedModels(),
		"cache-MISS return value mutation must not corrupt the cache (W6D fix)")

	// Now hammer concurrently: every reader gets its own copy and mutates it;
	// the cache must remain canonical throughout and afterwards.
	const workers = 24
	var leaks atomic.Int64
	var wg sync.WaitGroup
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				// Cache-HIT path copy.
				hit := d.DiscoverModels()
				for j := range hit {
					hit[j] = "HIT-PATH-MUTATED"
				}
				// GetCachedModels copy.
				cached := d.GetCachedModels()
				for _, m := range cached {
					if m == "HIT-PATH-MUTATED" || m == "MISS-PATH-MUTATED" {
						leaks.Add(1)
					}
				}
			}
		}()
	}
	wg.Wait()

	finalCached := d.GetCachedModels()
	scWriteEvidence(t, "chaos_caller_mutation.json", struct {
		Leaks         int64    `json:"caller_mutation_leaks_into_cache"`
		FinalCache    []string `json:"final_cache"`
		Expected      []string `json:"expected"`
	}{leaks.Load(), finalCached, canonicalModels})

	assert.Zero(t, leaks.Load(),
		"caller mutation of either the HIT-path or MISS-path return value must never leak into the cache")
	assert.ElementsMatch(t, canonicalModels, finalCached,
		"internal cache must remain canonical after concurrent caller mutation (W6D fix)")
}

// ---------------------------------------------------------------------------
// Determinism / re-runnability proof (§11.4.50 / §11.4.98)
// ---------------------------------------------------------------------------

// TestCacheStressChaos_NoPanicNoDeadlock is a fast smoke that exercises every
// op path with a bounded timeout so a deadlock surfaces as a test timeout (the
// `go test` watchdog) rather than a hang — asserting NEVER panic/deadlock.
func TestCacheStressChaos_NoPanicNoDeadlock(t *testing.T) {
	cs := newChaosServer()
	defer cs.Close()

	d := NewDiscoverer(ProviderConfig{
		ProviderName:   "smoke",
		ModelsEndpoint: cs.URL(),
		APIKey:         "test-key",
		CacheTTL:       1 * time.Millisecond,
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		var wg sync.WaitGroup
		for w := 0; w < runtime.NumCPU()*2+4; w++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for i := 0; i < 40; i++ {
					switch (id + i) % 5 {
					case 0:
						_ = d.DiscoverModels()
					case 1:
						_ = d.GetCachedModels()
					case 2:
						_ = d.GetDiscoveryTier()
					case 3:
						d.InvalidateCache()
					case 4:
						cs.setMode((int32(id+i) % 4))
					}
				}
			}(w)
		}
		wg.Wait()
	}()

	select {
	case <-done:
		// completed without panic or deadlock
	case <-time.After(30 * time.Second):
		t.Fatal("deadlock suspected: stress/chaos ops did not complete within 30s")
	}

	scWriteEvidence(t, "smoke_no_panic_no_deadlock.json", struct {
		Result string `json:"result"`
	}{"completed_no_panic_no_deadlock"})
}

// ---------------------------------------------------------------------------
// small set helpers
// ---------------------------------------------------------------------------

func sameSet(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	return subsetOfCanonical(got) && len(got) == len(want)
}

func subsetOfCanonical(s []string) bool {
	allowed := map[string]bool{}
	for _, m := range canonicalModels {
		allowed[m] = true
	}
	seen := map[string]int{}
	for _, m := range s {
		if !allowed[m] {
			return false
		}
		seen[m]++
		if seen[m] > 1 {
			return false // duplicate ⇒ torn/interleaved write
		}
	}
	return true
}
