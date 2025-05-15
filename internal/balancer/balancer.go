package balancer

import (
	"fmt"
	"net/http"

	"github.com/humanbelnik/load-balancer/internal/server/server"
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

type Balancer struct {
	pool   Pool
	policy Policy
}

type Option func(*Balancer)

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
	aliveServers, err := b.pool.Alive()
	if err != nil {
		http.Error(w, "no backends available", http.StatusServiceUnavailable)
		return
	}

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
