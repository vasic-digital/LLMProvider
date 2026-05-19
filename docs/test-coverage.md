# LLMProvider Test Coverage Ledger (round-276)

This ledger maps the surface-area symbols of LLMProvider to the
specific tests + Challenges that exercise them with positive
runtime evidence (CONST-035 / Article XI §11.9, CONST-050(B)).

> Verbatim 2026-05-19 operator mandate (preserved per
> CONST-049 §11.4.17):
> "all existing tests and Challenges do work in anti-bluff manner
> - they MUST confirm that all tested codebase really works as
> expected! We had been in position that all tests do execute with
> success and all Challenges as well, but in reality the most of
> the features does not work and can't be used! This MUST NOT be
> the case and execution of tests and Challenges MUST guarantee
> the quality, the completition and full usability by end users
> of the product!"

## Anti-bluff guarantees

Every row below identifies BOTH a symbol AND the failing-line a
mutation would produce. Mutation pairing per §1.1 / CONST-050(A).

## Ledger — pkg/circuit

| Symbol                                | Test                                              | Challenge invariant                                |
|---------------------------------------|---------------------------------------------------|----------------------------------------------------|
| `NewCircuitBreaker()`                 | `pkg/circuit/circuit_breaker_test.go`             | runner construction path (5 locales)               |
| `NewDefaultCircuitBreaker()`          | `pkg/circuit/circuit_breaker_test.go`             | (covered via NewCircuitBreaker)                    |
| `DefaultCircuitBreakerConfig()`       | `pkg/circuit/circuit_breaker_test.go`             | runner uses defaults overridden per fixture        |
| `CircuitBreaker.GetState()`           | `pkg/circuit/circuit_breaker_test.go`             | `circuit.initial_state.<locale>`                   |
| `CircuitBreaker.IsClosed()`/`IsOpen()`| `pkg/circuit/circuit_breaker_test.go`             | `circuit.is_closed.<locale>`                       |
| `CircuitBreaker.Complete()`           | `pkg/circuit/circuit_breaker_test.go`             | `circuit.opens_after_failures.<locale>`            |
| `CircuitBreaker.ErrCircuitOpen`       | `pkg/circuit/circuit_breaker_test.go`             | runner asserts `errors.Is(err, ErrCircuitOpen)`    |
| `CircuitBreaker.AddListener()`        | `pkg/circuit/circuit_breaker_test.go`             | (covered by unit suite)                            |
| `CircuitBreakerManager.Register()`    | `pkg/circuit/circuit_breaker_test.go`             | (covered by unit suite)                            |

## Ledger — pkg/health

| Symbol                                       | Test                                            | Challenge invariant                       |
|----------------------------------------------|-------------------------------------------------|-------------------------------------------|
| `NewHealthMonitor()` / `Default*`            | `pkg/health/health_monitor_test.go`             | runner constructs one shared monitor      |
| `HealthMonitor.RegisterProvider()`           | `pkg/health/health_monitor_test.go`             | `health.initial_status.<locale>` (x5)     |
| `HealthMonitor.GetHealth()`                  | `pkg/health/health_monitor_test.go`             | runner asserts Status==Unknown initially  |
| `HealthMonitor.RecordFailure()`              | `pkg/health/health_monitor_test.go`             | `health.transitions_after_failures`       |
| `HealthMonitor.AddListener()`                | `pkg/health/health_monitor_test.go`             | runner counts listener flips              |
| `HealthStatusUnknown`/`Unhealthy` constants  | `pkg/health/health_monitor_test.go`             | runner asserts status string literals     |

## Ledger — pkg/retry

| Symbol                          | Test                                  | Challenge invariant                             |
|---------------------------------|---------------------------------------|-------------------------------------------------|
| `IsRetryableStatusCode()`       | `pkg/retry/retry_test.go`             | `retry.is_retryable_status_code`                |
| `CalculateBackoff()`            | `pkg/retry/retry_test.go`             | `retry.calculate_backoff_bounds`                |
| `DefaultRetryConfig()`          | `pkg/retry/retry_test.go`             | runner builds custom config to bound MaxDelay   |
| `ExecuteWithRetry()`            | `pkg/retry/retry_test.go`             | (covered by unit + HTTP suites)                 |
| `RetryConfig` struct fields     | `pkg/retry/retry_test.go`             | runner validates Initial/Max/Multiplier bounds  |

## Ledger — pkg/provider + pkg/models

| Symbol                          | Test                                       | Challenge invariant                     |
|---------------------------------|--------------------------------------------|-----------------------------------------|
| `provider.LLMProvider` iface    | (every provider package)                   | runner satisfies it via `controllableProvider` |
| `models.LLMRequest`             | `pkg/models/types_test.go`                 | runner builds real requests per-fixture |
| `models.LLMResponse`            | `pkg/models/types_test.go`                 | runner threads through breaker          |
| `models.ProviderCapabilities`   | `pkg/models/types_test.go`                 | runner asserts non-nil from controllable|

## Challenges (challenges/ + challenges/scripts/)

| Challenge                                   | Purpose                                   | Mutation-paired? |
|---------------------------------------------|-------------------------------------------|------------------|
| `llmprovider_describe_challenge.sh`         | runner invariants (23 PASS)               | **YES** (round-276) |
| `chaos_failure_injection_challenge.sh`      | runtime chaos resilience                  | env-gated        |
| `ddos_health_flood_challenge.sh`            | DDoS health-flood resilience              | env-gated        |
| `scaling_horizontal_challenge.sh`           | horizontal scale check                    | env-gated        |
| `stress_sustained_load_challenge.sh`        | stress floor                              | env-gated        |
| `ui_terminal_interaction_challenge.sh`      | TUI smoke                                 | env-gated        |
| `ux_end_to_end_flow_challenge.sh`           | UX end-to-end                             | env-gated        |
| `host_no_auto_suspend_challenge.sh`         | CONST-033 host-power ban                  | n                |
| `no_suspend_calls_challenge.sh`             | CONST-033 anti-suspend scan               | n                |

## How to run

```bash
# Race-detector full unit + runner pass:
GOMAXPROCS=2 go test -count=1 -race -p 1 ./...

# Anti-bluff describe Challenge (23 invariants):
./challenges/llmprovider_describe_challenge.sh normal   # exit 0
./challenges/llmprovider_describe_challenge.sh mutate   # exit 99

# Standalone runner (useful for forensics):
go run ./challenges/runner/
LLMPROVIDER_MUTATE_RUNNER=1 go run ./challenges/runner/
```

## Cascade

CONST-047 + CONST-051(A): this ledger is the LLMProvider-side
realisation of the cross-org "equal codebase" mandate. The same
ledger pattern lives in sibling submodules (LeakHub round-266,
MCP_Module round-267, Models round-268, Ouroborous round-269,
Planning round-270, conversation round-271, DebateOrchestrator
round-272, HelixSpecifier round-273) and is bumped from the
meta-repo on every round close-out.
