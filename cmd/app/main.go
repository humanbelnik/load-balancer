package main

import (
	"flag"
	"log"

	"github.com/humanbelnik/load-balancer/internal/app"
)

func args() app.Config {
	var (
		port        = flag.String("port", "8080", "Port to listen on")
		host        = flag.String("host", "localhost", "Host to bind")
		confpath    = flag.String("config", "./config/config.yaml", "Path to backend servers config file")
		rlimit      = flag.Bool("rlimit", false, "Enable rate limiter")
		rlimitStore = flag.String("rlstore", "ratelimiter.db", "Path to rate-limiter DB")
	)
	flag.Parse()
	log.Printf("using: port=%s, host=%s, config=%s, with-rate-limiting=%v, rlimit-store=%s",
		*port, *host, *confpath, *rlimit, *rlimitStore)

	return app.Config{
		Port:        *port,
		Host:        *host,
		Confpath:    *confpath,
		Rlimit:      *rlimit,
		RlimitStore: *rlimitStore,
	}
}

func main() {
	cfg := args()
	server, err := app.Setup(cfg)
	if err != nil {
		log.Fatalf("%v", err)
	}
	app.Run(server)
}
