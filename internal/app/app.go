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
	api_ratelimiter "github.com/humanbelnik/load-balancer/internal/ratelimiter/api/http"
	"github.com/humanbelnik/load-balancer/internal/ratelimiter/ratelimiter"
	sqlite_storage "github.com/humanbelnik/load-balancer/internal/ratelimiter/storage/sqlite"
)

type Config struct {
	Port        string
	Host        string
	Confpath    string
	Rlimit      bool
	RlimitStore string
}

func setupBalancer(appCfg Config, mux *http.ServeMux) ([]balancer.Option, error) {
	opts := []balancer.Option{}
	if appCfg.Rlimit {
		store, err := sqlite_storage.New(appCfg.RlimitStore)
		if err != nil {
			return nil, fmt.Errorf("rate limiter storage")
		}
		rlLoader := yaml_config.NewRateLimiterLoader()
		cfg, err := rlLoader.Load(appCfg.Confpath)
		if err != nil {
			return nil, fmt.Errorf("rate limiter config")
		}
		rl := ratelimiter.New(cfg, store)
		//rl.SetClient("127.0.0.1", nil, nil)
		opts = append(opts, balancer.WithRateLimiter(rl))
		api := api_ratelimiter.API{Limiter: rl}
		mux.HandleFunc("/clients", func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPost:
				api.AddClient(w, r)
			case http.MethodDelete:
				api.DeleteClient(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		})
	}
	return opts, nil
}

func Setup(appCfg Config) (*http.Server, error) {
	// Manually load config and setup server pool on the launch
	loader := yaml_config.NewBalancerLoader()
	urls, err := loader.Load(appCfg.Confpath)
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
	watcher.Watch(appCfg.Confpath, p)

	// Configure load balancer
	mux := http.NewServeMux()
	balancerOpts, err := setupBalancer(appCfg, mux)
	if err != nil {
		return nil, fmt.Errorf("setting up balancer options: %w", err)
	}

	roundRobinPolicy := rr.New()
	bal := balancer.New(p, roundRobinPolicy, balancerOpts...)
	addr := appCfg.Host + ":" + appCfg.Port

	// Configure API
	mux.HandleFunc("/", bal.Serve)
	http.HandleFunc("/", bal.Serve)
	return &http.Server{
		Addr:    addr,
		Handler: mux,
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
