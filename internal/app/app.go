package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/humanbelnik/load-balancer/internal/balancer/balancer"
	"github.com/humanbelnik/load-balancer/internal/balancer/policy/rr"
	"github.com/humanbelnik/load-balancer/internal/balancer/pool/config_watcher"
	"github.com/humanbelnik/load-balancer/internal/balancer/pool/dynamic_pool"
	"github.com/humanbelnik/load-balancer/internal/balancer/server/factory"
	yaml_config "github.com/humanbelnik/load-balancer/internal/config/yaml"
	"github.com/humanbelnik/load-balancer/internal/ratelimiter/ratelimiter"
)

func Setup(configPath, addr string) (*http.Server, error) {
	// Manually load config and setup server pool on the launch
	loader := yaml_config.NewBalancerLoader()
	urls, err := loader.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("load balancer config: %w", err)
	}

	factory := factory.New()
	p := dynamic_pool.New(factory)

	if err := p.Update(urls); err != nil {
		return nil, fmt.Errorf("update pool: %w", err)
	}

	// Watch after SIGHUPs
	watcher := config_watcher.New(loader, config_watcher.DefaultOnError)
	watcher.Watch(configPath, p)

	// Rate limiter
	rlLoader := yaml_config.NewRateLimiterLoader()
	cfg, err := rlLoader.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("rate limiter config")
	}
	rl := ratelimiter.New(cfg)
	rl.SetClient("127.0.0.1", nil, nil)

	// Configure load balancer
	roundRobinPolicy := rr.New()
	bal := balancer.New(p, roundRobinPolicy, balancer.WithRateLimiter(rl))
	log.Println(addr)
	return &http.Server{
		Addr:    addr,
		Handler: http.HandlerFunc(bal.Serve),
	}, nil
}

/*
Performs gracefull shutdown on SIGINT/SIGTERM.
*/
func Run(srv *http.Server) {
	idleConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, syscall.SIGINT, syscall.SIGTERM)
		<-sigint

		log.Println("shutdown signal received")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("graceful shutdown failed: %v", err)
		} else {
			log.Println("shutdown complete")
		}

		close(idleConnsClosed)
	}()

	log.Printf("listening on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}

	<-idleConnsClosed
}
