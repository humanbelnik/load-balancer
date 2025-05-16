package balancer

import (
	"log/slog"
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
	logger  *slog.Logger
}

type Option func(*Balancer)

func WithRateLimiter(rl RateLimiter) Option {
	return func(b *Balancer) {
		b.ratelim = rl
	}
}

func WithLogger(logger *slog.Logger) Option {
	return func(b *Balancer) {
		b.logger = logger
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
		// If not specified in functional options - use default
		logger: slog.Default(),
	}

	for _, opt := range opts {
		opt(b)
	}
	return b
}

func (b *Balancer) Serve(w http.ResponseWriter, r *http.Request) {
	b.logger.Info("handling request", slog.String("method", r.Method), slog.String("url", r.URL.String()), slog.String("client", r.RemoteAddr))
	if b.ratelim != nil && !b.HasTicket(w, r) {
		return
	}

	aliveServers, err := b.pool.Alive()
	if err != nil {
		b.logger.Error("no backends available", slog.Any("err", err))
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
			b.logger.Error("policy selection failed", slog.Any("err", err))
			http.Error(w, "policy error", http.StatusServiceUnavailable)
			return
		}

		err = srv.Serve(w, r)
		if err == nil {
			b.logger.Info("request served", slog.String("server", srv.URL()))
			return
		}

		b.logger.Warn("backend failed", slog.String("server", srv.URL()), slog.Any("err", err))
	}

	http.Error(w, "all backends failed", http.StatusBadGateway)
}

/*
Wrap rate limiter's work
*/
func (b *Balancer) HasTicket(w http.ResponseWriter, r *http.Request) bool {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		b.logger.Warn("malformed remote addr", slog.String("remote", r.RemoteAddr), slog.Any("err", err))
		http.Error(w, "incorrect request", http.StatusBadRequest)
		return false
	}

	if !b.ratelim.Allow(ip) {
		b.logger.Info("rate limit exceeded", slog.String("ip", ip))
		http.Error(w, "rate limit exeed", http.StatusTooManyRequests)
		return false
	}

	return true
}
