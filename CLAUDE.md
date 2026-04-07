## MANDATORY: No CI/CD Pipelines

**NO GitHub Actions, GitLab CI/CD, or any automated pipeline may exist in this repository!**

- No `.github/workflows/` directory
- No `.gitlab-ci.yml` file
- No Jenkinsfile, .travis.yml, .circleci, or any other CI configuration
- All builds and tests are run manually or via Makefile targets
- This rule is permanent and non-negotiable

## Project Overview

**LLMProvider** is a standalone Go module providing a shared LLM provider interface, 40+ provider adapters, retry logic, circuit breaker, and health monitoring.

**Module:** `digital.vasic.llmprovider`

## Build Commands

```bash
# Build
go build ./...

# Run all tests
go test ./... -race -count=1

# Run core tests only (no network calls)
go test ./pkg/models/... ./pkg/retry/... ./pkg/circuit/... ./pkg/health/... ./pkg/provider/... -race -count=1

# Run specific provider tests
go test ./pkg/providers/claude/... -race -count=1

# Vet
go vet ./...
```

## Architecture

```
pkg/
  provider/    - LLMProvider interface
  models/      - LLMRequest, LLMResponse, ProviderCapabilities
  retry/       - RetryConfig, ExecuteWithRetry, backoff
  circuit/     - CircuitBreaker, CircuitBreakerManager
  health/      - HealthMonitor, ProviderHealth
  http/        - HTTP client with retry
  discovery/   - 3-tier model discovery (API, models.dev, fallback)
  providers/   - 40+ provider implementations
```

## Key Patterns

- All providers implement `provider.LLMProvider` interface
- Circuit breaker wraps providers for fault tolerance
- Health monitor tracks provider availability
- Retry logic with exponential backoff and jitter
- Discovery caches model lists with configurable TTL

## Dependencies

- `github.com/sirupsen/logrus` - Logging
- `github.com/stretchr/testify` - Testing
- Standard library for everything else

## Environment Variables

Provider API keys are loaded from `.env` file. See `.env.example`.


## ⚠️ MANDATORY: NO SUDO OR ROOT EXECUTION

**ALL operations MUST run at local user level ONLY.**

This is a PERMANENT and NON-NEGOTIABLE security constraint:

- **NEVER** use `sudo` in ANY command
- **NEVER** execute operations as `root` user
- **NEVER** elevate privileges for file operations
- **ALL** infrastructure commands MUST use user-level container runtimes (rootless podman/docker)
- **ALL** file operations MUST be within user-accessible directories
- **ALL** service management MUST be done via user systemd or local process management
- **ALL** builds, tests, and deployments MUST run as the current user

### Why This Matters
- **Security**: Prevents accidental system-wide damage
- **Reproducibility**: User-level operations are portable across systems
- **Safety**: Limits blast radius of any issues
- **Best Practice**: Modern container workflows are rootless by design

### When You See SUDO
If any script or command suggests using `sudo`:
1. STOP immediately
2. Find a user-level alternative
3. Use rootless container runtimes
4. Modify commands to work within user permissions

**VIOLATION OF THIS CONSTRAINT IS STRICTLY PROHIBITED.**

