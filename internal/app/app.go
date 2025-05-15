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

	"github.com/humanbelnik/load-balancer/internal/balancer"
	yaml_config "github.com/humanbelnik/load-balancer/internal/config/yaml"
	"github.com/humanbelnik/load-balancer/internal/policy/rr"
	"github.com/humanbelnik/load-balancer/internal/pool/config_watcher"
	"github.com/humanbelnik/load-balancer/internal/pool/dynamic_pool"
	"github.com/humanbelnik/load-balancer/internal/server/factory"
)

func Setup(configPath, addr string) (*http.Server, error) {
	// Manually load config and setup server pool on the launch
	loader := &yaml_config.YAMLLoader{}
	urls, err := loader.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	factory := factory.New()
	p := dynamic_pool.New(factory)
	if err := p.Update(urls); err != nil {
		return nil, fmt.Errorf("update pool: %w", err)
	}

	// Watch after SIGHUPs
	watcher := config_watcher.New(loader, config_watcher.DefaultOnError)
	watcher.Watch(configPath, p)

	// Configure load balancer
	roundRobinPolicy := rr.New()
	bal := balancer.New(p, roundRobinPolicy)

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
