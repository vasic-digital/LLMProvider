package circuit

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"digital.vasic.llmprovider/pkg/models"
)

// failingProvider is a mock that can be configured to fail
type failingProvider struct {
	shouldFail bool
	mu         sync.Mutex
}

func (p *failingProvider) Complete(_ context.Context, _ *models.LLMRequest) (*models.LLMResponse, error) {
	p.mu.Lock()
	fail := p.shouldFail
	p.mu.Unlock()

	if fail {
		return nil, errors.New("provider error")
	}
	return &models.LLMResponse{Content: "success"}, nil
}

func (p *failingProvider) CompleteStream(_ context.Context, _ *models.LLMRequest) (<-chan *models.LLMResponse, error) {
	ch := make(chan *models.LLMResponse)
	go func() {
		defer close(ch)
		p.mu.Lock()
		fail := p.shouldFail
		p.mu.Unlock()

		if !fail {
			ch <- &models.LLMResponse{Content: "stream chunk"}
		}
	}()
	return ch, nil
}

func (p *failingProvider) HealthCheck() error            { return nil }
func (p *failingProvider) GetCapabilities() *models.ProviderCapabilities {
	return &models.ProviderCapabilities{}
}
func (p *failingProvider) ValidateConfig(_ map[string]interface{}) (bool, []string) {
	return true, nil
}

func (p *failingProvider) SetShouldFail(fail bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.shouldFail = fail
}

func TestDefaultCircuitBreakerConfig(t *testing.T) {
	config := DefaultCircuitBreakerConfig()

	assert.Equal(t, 5, config.FailureThreshold)
	assert.Equal(t, 2, config.SuccessThreshold)
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.Equal(t, 3, config.HalfOpenMaxRequests)
}

func TestCircuitBreaker_StartsInClosedState(t *testing.T) {
	provider := &failingProvider{}
	cb := NewDefaultCircuitBreaker("test", provider)

	assert.Equal(t, CircuitClosed, cb.GetState())
	assert.True(t, cb.IsClosed())
	assert.False(t, cb.IsOpen())
	assert.False(t, cb.IsHalfOpen())
}

func TestCircuitBreaker_Complete_Success(t *testing.T) {
	provider := &failingProvider{}
	cb := NewDefaultCircuitBreaker("test", provider)

	req := &models.LLMRequest{ID: "test"}
	resp, err := cb.Complete(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "success", resp.Content)

	stats := cb.GetStats()
	assert.Equal(t, int64(1), stats.TotalRequests)
	assert.Equal(t, int64(1), stats.TotalSuccesses)
	assert.Equal(t, int64(0), stats.TotalFailures)
}

func TestCircuitBreaker_OpensAfterFailures(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold:    3,
		SuccessThreshold:    2,
		Timeout:             1 * time.Minute,
		HalfOpenMaxRequests: 2,
	}
	provider := &failingProvider{shouldFail: true}
	cb := NewCircuitBreaker("test", provider, config)

	req := &models.LLMRequest{ID: "test"}
	for i := 0; i < 3; i++ {
		_, err := cb.Complete(context.Background(), req)
		assert.Error(t, err)
	}

	assert.Equal(t, CircuitOpen, cb.GetState())
	assert.True(t, cb.IsOpen())
}

func TestCircuitBreaker_RejectsWhenOpen(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold:    2,
		Timeout:             1 * time.Minute,
		HalfOpenMaxRequests: 1,
	}
	provider := &failingProvider{shouldFail: true}
	cb := NewCircuitBreaker("test", provider, config)

	req := &models.LLMRequest{ID: "test"}
	_, _ = cb.Complete(context.Background(), req)
	_, _ = cb.Complete(context.Background(), req)

	assert.True(t, cb.IsOpen())

	_, err := cb.Complete(context.Background(), req)
	assert.Equal(t, ErrCircuitOpen, err)
}

func TestCircuitBreaker_TransitionsToHalfOpen(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold:    2,
		SuccessThreshold:    3,
		Timeout:             100 * time.Millisecond,
		HalfOpenMaxRequests: 5,
	}
	provider := &failingProvider{shouldFail: true}
	cb := NewCircuitBreaker("test", provider, config)

	req := &models.LLMRequest{ID: "test"}
	_, _ = cb.Complete(context.Background(), req)
	_, _ = cb.Complete(context.Background(), req)
	assert.True(t, cb.IsOpen())

	time.Sleep(150 * time.Millisecond)
	provider.SetShouldFail(false)

	_, err := cb.Complete(context.Background(), req)
	assert.NoError(t, err)
	assert.True(t, cb.IsHalfOpen(), "Circuit should be half-open after first success")
}

func TestCircuitBreaker_ClosesAfterSuccessesInHalfOpen(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold:    2,
		SuccessThreshold:    2,
		Timeout:             100 * time.Millisecond,
		HalfOpenMaxRequests: 5,
	}
	provider := &failingProvider{shouldFail: true}
	cb := NewCircuitBreaker("test", provider, config)

	req := &models.LLMRequest{ID: "test"}
	_, _ = cb.Complete(context.Background(), req)
	_, _ = cb.Complete(context.Background(), req)
	assert.True(t, cb.IsOpen())

	time.Sleep(150 * time.Millisecond)
	provider.SetShouldFail(false)

	_, _ = cb.Complete(context.Background(), req)
	_, _ = cb.Complete(context.Background(), req)

	assert.True(t, cb.IsClosed())
}

func TestCircuitBreaker_Reset(t *testing.T) {
	config := CircuitBreakerConfig{FailureThreshold: 2}
	provider := &failingProvider{shouldFail: true}
	cb := NewCircuitBreaker("test", provider, config)

	req := &models.LLMRequest{ID: "test"}
	_, _ = cb.Complete(context.Background(), req)
	_, _ = cb.Complete(context.Background(), req)
	assert.True(t, cb.IsOpen())

	cb.Reset()
	assert.True(t, cb.IsClosed())

	stats := cb.GetStats()
	assert.Equal(t, 0, stats.ConsecutiveFailures)
}

func TestCircuitBreaker_Stats(t *testing.T) {
	provider := &failingProvider{}
	cb := NewDefaultCircuitBreaker("test-provider", provider)

	req := &models.LLMRequest{ID: "test"}
	_, _ = cb.Complete(context.Background(), req)
	_, _ = cb.Complete(context.Background(), req)
	provider.SetShouldFail(true)
	_, _ = cb.Complete(context.Background(), req)

	stats := cb.GetStats()
	assert.Equal(t, "test-provider", stats.ProviderID)
	assert.Equal(t, int64(3), stats.TotalRequests)
	assert.Equal(t, int64(2), stats.TotalSuccesses)
	assert.Equal(t, int64(1), stats.TotalFailures)
}

func TestCircuitBreaker_Listener(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 2,
		Timeout:          100 * time.Millisecond,
	}
	provider := &failingProvider{shouldFail: true}
	cb := NewCircuitBreaker("test", provider, config)

	stateChanges := make([]CircuitState, 0)
	var mu sync.Mutex

	cb.AddListener(func(providerID string, oldState, newState CircuitState) {
		mu.Lock()
		stateChanges = append(stateChanges, newState)
		mu.Unlock()
	})

	req := &models.LLMRequest{ID: "test"}
	_, _ = cb.Complete(context.Background(), req)
	_, _ = cb.Complete(context.Background(), req)

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	assert.Contains(t, stateChanges, CircuitOpen)
	mu.Unlock()
}

func TestCircuitBreakerManager_Register(t *testing.T) {
	mgr := NewDefaultCircuitBreakerManager()
	provider := &failingProvider{}

	cb := mgr.Register("test", provider)
	assert.NotNil(t, cb)

	retrieved, exists := mgr.Get("test")
	assert.True(t, exists)
	assert.Equal(t, cb, retrieved)
}

func TestCircuitBreakerManager_Unregister(t *testing.T) {
	mgr := NewDefaultCircuitBreakerManager()
	provider := &failingProvider{}

	mgr.Register("test", provider)
	mgr.Unregister("test")

	_, exists := mgr.Get("test")
	assert.False(t, exists)
}

func TestCircuitBreakerManager_GetAllStats(t *testing.T) {
	mgr := NewDefaultCircuitBreakerManager()
	mgr.Register("provider1", &failingProvider{})
	mgr.Register("provider2", &failingProvider{})

	stats := mgr.GetAllStats()
	assert.Len(t, stats, 2)
	assert.Contains(t, stats, "provider1")
	assert.Contains(t, stats, "provider2")
}

func TestCircuitBreakerManager_GetAvailableProviders(t *testing.T) {
	config := CircuitBreakerConfig{FailureThreshold: 2}
	mgr := NewCircuitBreakerManager(config)

	mgr.Register("healthy", &failingProvider{})
	cb := mgr.Register("unhealthy", &failingProvider{shouldFail: true})

	req := &models.LLMRequest{ID: "test"}
	_, _ = cb.Complete(context.Background(), req)
	_, _ = cb.Complete(context.Background(), req)

	available := mgr.GetAvailableProviders()
	assert.Contains(t, available, "healthy")
	assert.NotContains(t, available, "unhealthy")
}

func TestCircuitBreakerManager_ResetAll(t *testing.T) {
	config := CircuitBreakerConfig{FailureThreshold: 2}
	mgr := NewCircuitBreakerManager(config)

	cb1 := mgr.Register("p1", &failingProvider{shouldFail: true})
	cb2 := mgr.Register("p2", &failingProvider{shouldFail: true})

	req := &models.LLMRequest{ID: "test"}
	_, _ = cb1.Complete(context.Background(), req)
	_, _ = cb1.Complete(context.Background(), req)
	_, _ = cb2.Complete(context.Background(), req)
	_, _ = cb2.Complete(context.Background(), req)

	assert.True(t, cb1.IsOpen())
	assert.True(t, cb2.IsOpen())

	mgr.ResetAll()

	assert.True(t, cb1.IsClosed())
	assert.True(t, cb2.IsClosed())
}

func TestCircuitBreaker_CompleteStream_Success(t *testing.T) {
	provider := &failingProvider{}
	cb := NewDefaultCircuitBreaker("test", provider)

	req := &models.LLMRequest{ID: "test"}
	ch, err := cb.CompleteStream(context.Background(), req)

	assert.NoError(t, err)
	for range ch {
	}

	time.Sleep(50 * time.Millisecond)

	stats := cb.GetStats()
	assert.Equal(t, int64(1), stats.TotalSuccesses)
}

func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold:    10,
		SuccessThreshold:    5,
		Timeout:             100 * time.Millisecond,
		HalfOpenMaxRequests: 5,
	}
	provider := &failingProvider{}
	cb := NewCircuitBreaker("test", provider, config)

	req := &models.LLMRequest{ID: "test"}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = cb.Complete(context.Background(), req)
			_ = cb.GetStats()
			_ = cb.GetState()
		}()
	}

	wg.Wait()

	stats := cb.GetStats()
	assert.Equal(t, int64(100), stats.TotalRequests)
}

// logrusWarnHook captures logrus Warn-level entries for test assertions.
type logrusWarnHook struct {
	mu      sync.Mutex
	entries []string
}

func (h *logrusWarnHook) Levels() []logrus.Level {
	return []logrus.Level{logrus.WarnLevel}
}

func (h *logrusWarnHook) Fire(entry *logrus.Entry) error {
	h.mu.Lock()
	h.entries = append(h.entries, entry.Message)
	h.mu.Unlock()
	return nil
}

func (h *logrusWarnHook) messages() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	cp := make([]string, len(h.entries))
	copy(cp, h.entries)
	return cp
}

func TestCircuitBreaker_ListenerNotifyTimeout_TransitionTo(t *testing.T) {
	orig := ListenerNotifyTimeoutNs.Load()
	ListenerNotifyTimeoutNs.Store(int64(50 * time.Millisecond))
	defer ListenerNotifyTimeoutNs.Store(orig)

	hook := &logrusWarnHook{}
	logrus.AddHook(hook)
	defer logrus.StandardLogger().ReplaceHooks(logrus.LevelHooks{})

	config := CircuitBreakerConfig{
		FailureThreshold:    1,
		SuccessThreshold:    1,
		Timeout:             500 * time.Millisecond,
		HalfOpenMaxRequests: 1,
	}
	provider := &failingProvider{shouldFail: true}
	cb := NewCircuitBreaker("timeout-test", provider, config)

	blockCh := make(chan struct{})
	cb.AddListener(func(providerID string, oldState, newState CircuitState) {
		<-blockCh
	})

	req := &models.LLMRequest{ID: "r1"}
	_, _ = cb.Complete(context.Background(), req)

	time.Sleep(200 * time.Millisecond)
	close(blockCh)

	msgs := hook.messages()
	found := false
	for _, m := range msgs {
		if strings.Contains(m, "timed out") {
			found = true
			break
		}
	}
	assert.True(t, found, "expected a 'timed out' warn log, got: %v", msgs)
}
