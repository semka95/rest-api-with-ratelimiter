package limiter

import (
	"context"
	"sync"
	"time"

	"github.com/sethvargo/go-limiter"
	"github.com/sethvargo/go-limiter/memorystore"
)

// RequestLimiter represents request limiter store
type RequestLimiter struct {
	mu            *sync.RWMutex
	isTimedOut    map[string]bool
	limiterStore  limiter.Store
	subnetTimeout time.Duration
}

// NewRequestLimiter creates request limiter store
func NewRequestLimiter(tokens uint64, interval, requestCooldown time.Duration) (RequestLimiter, error) {
	limiterStore, err := memorystore.New(&memorystore.Config{
		Tokens:   tokens,
		Interval: interval,
	})
	if err != nil {
		return RequestLimiter{}, err
	}

	return RequestLimiter{
		&sync.RWMutex{},
		make(map[string]bool),
		limiterStore,
		requestCooldown,
	}, nil
}

// CooldownSubnet prohibits all requests for given subnet
func (l *RequestLimiter) CooldownSubnet(ip string) {
	l.mu.Lock()
	l.isTimedOut[ip] = true
	l.mu.Unlock()
	go l.allowAfterTimeout(ip)
}

// allowAfterTimeout permits all requests for given subnet after timeout
func (l *RequestLimiter) allowAfterTimeout(ip string) {
	time.Sleep(l.subnetTimeout)
	l.mu.Lock()
	l.isTimedOut[ip] = false
	l.mu.Unlock()
}

// IsTimedOut checks if subnet is timed out
func (l *RequestLimiter) IsTimedOut(ip string) bool {
	l.mu.RLock()
	c, ok := l.isTimedOut[ip]
	l.mu.RUnlock()
	return ok && c
}

// Take takes token from limiter store for the given subnet
func (l *RequestLimiter) Take(ctx context.Context, ip string) (remaining uint64, ok bool, err error) {
	_, remaining, _, ok, err = l.limiterStore.Take(ctx, ip)
	return
}

// Close closes limite store
func (l *RequestLimiter) Close(ctx context.Context) error {
	return l.limiterStore.Close(ctx)
}

// func (l *RequestLimiter) Reset(ip string) {
// 	// l.limiterStore.Set(ctx context.Context, key string, tokens uint64, interval time.Duration)
// 	l.limiterStore.Get(ctx, ip)
// }
