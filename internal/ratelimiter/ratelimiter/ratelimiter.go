package ratelimiter

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/humanbelnik/load-balancer/internal/ratelimiter/config"
)

var (
	ErrAddIP    = errors.New("unable to add ip")
	ErrRemoveIP = errors.New("unable to remove ip")
)

/*
Token bucket implementation.
*/

type tokenBucket struct {
	capacity   int
	tokens     int
	refEvery   time.Duration
	lastRefill time.Time
	mu         sync.Mutex
}

type Storage interface {
	Add(IP string, capacity int, refillEvery time.Duration) error
	Delete(IP string) error
	LoadAll() ([]ClientConfig, error)
}

type ClientConfig struct {
	IP          string
	Capacity    int
	RefillEvery time.Duration
}

type Limiter struct {
	cfg config.RateLimiterConfig

	clients map[string]*tokenBucket
	mu      sync.RWMutex
	ticker  *time.Ticker
	quit    chan struct{}
	store   Storage
}

func New(cfg config.RateLimiterConfig, store Storage) *Limiter {
	rl := &Limiter{
		cfg:     cfg,
		clients: make(map[string]*tokenBucket),
		ticker:  time.NewTicker(time.Duration(cfg.DefaultRefillRate)),
		quit:    make(chan struct{}),
		store:   store,
	}

	go rl.refillLoop()
	return rl
}

func (rl *Limiter) refillLoop() {
	for {
		select {
		case <-rl.ticker.C:
			rl.mu.RLock()
			for _, bucket := range rl.clients {
				bucket.mu.Lock()
				elapsed := time.Since(bucket.lastRefill)
				newTokens := int(elapsed / bucket.refEvery)
				if newTokens > 0 {
					bucket.tokens += newTokens
					if bucket.tokens > bucket.capacity {
						bucket.tokens = bucket.capacity
					}
					bucket.lastRefill = time.Now()
				}
				bucket.mu.Unlock()
			}
			rl.mu.RUnlock()
		case <-rl.quit:
			return
		}
	}
}

/*
Use defaults for capacity && refill rate if not specified explicitly in parameters.
*/
func (rl *Limiter) SetClient(ip string, capacity *int, rate *time.Duration) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	var (
		cap int
		ref time.Duration
	)

	if capacity == nil {
		cap = rl.cfg.DefaultCapacity
	} else {
		cap = *capacity
	}
	if rate == nil {
		ref = rl.cfg.DefaultRefillRate
	} else {
		ref = *rate
	}

	if err := rl.store.Add(ip, cap, ref); err != nil {
		return fmt.Errorf("%w: %w", ErrAddIP, err)
	}

	rl.clients[ip] = &tokenBucket{
		capacity:   cap,
		tokens:     cap,
		refEvery:   ref,
		lastRefill: time.Now(),
	}
	return nil
}

func (rl *Limiter) RemoveClient(ip string) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if err := rl.store.Delete(ip); err != nil {
		return fmt.Errorf("%w: %w", ErrRemoveIP, err)
	}

	delete(rl.clients, ip)
	return nil
}

func (rl *Limiter) Allow(ip string) bool {
	rl.mu.RLock()
	bucket, ok := rl.clients[ip]
	rl.mu.RUnlock()
	if !ok {
		return false
	}

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	if bucket.tokens > 0 {
		bucket.tokens--
		return true
	}
	return false
}
