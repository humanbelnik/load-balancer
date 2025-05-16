package balancer

import (
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/humanbelnik/load-balancer/internal/balancer/server/server"
)

/*
Pool is responsible for giving alive servers for the future routing based
on policy decision.
*/
type Pool interface {
	Alive() ([]server.Server, error)
}

/*
Policy instance is responsible for choosing a server to route request to.
Uses Round-Robin or any other scheduling alorithm.
*/
type Policy interface {
	Select(servers []server.Server) (server.Server, error)
}

type RateLimiter interface {
	Allow(ip string) bool
}

type Balancer struct {
	pool    Pool
	policy  Policy
	ratelim RateLimiter
}

type Option func(*Balancer)

func WithRateLimiter(rl RateLimiter) Option {
	return func(b *Balancer) {
		b.ratelim = rl
	}
}

/*
Explicitly defining required parameters.
Other stuff (eg. Rate limiter) is optional.
*/
func New(pool Pool, policy Policy, opts ...Option) *Balancer {
	b := &Balancer{
		pool:   pool,
		policy: policy,
	}

	for _, opt := range opts {
		opt(b)
	}
	return b
}

func (b *Balancer) Serve(w http.ResponseWriter, r *http.Request) {
	if b.ratelim != nil && !b.HasTicket(w, r) {
		return
	}

	log.Println("serve!")
	aliveServers, err := b.pool.Alive()
	if err != nil {
		http.Error(w, "no backends available", http.StatusServiceUnavailable)
		return
	}

	/*
		Try in loop.
		If choosen server gave 5xx (his problem) - retry with the next.
	*/
	for range aliveServers {
		srv, err := b.policy.Select(aliveServers)
		if err != nil {
			http.Error(w, "policy error", http.StatusServiceUnavailable)
			return
		}

		err = srv.Serve(w, r)
		if err == nil {
			return
		}

		fmt.Printf("[WARN] server %s failed: %v — trying next\n", srv.URL(), err)
	}

	http.Error(w, "all backends failed", http.StatusBadGateway)
}

/*
Wrap rate limiter's work
*/
func (b *Balancer) HasTicket(w http.ResponseWriter, r *http.Request) bool {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		http.Error(w, "incorrect request", http.StatusBadRequest)
		return false
	}

	if !b.ratelim.Allow(ip) {
		http.Error(w, "rate limit exeed", http.StatusTooManyRequests)
		return false
	}

	return true
}
