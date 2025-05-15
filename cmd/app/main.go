package main

import (
	"flag"
	"log"

	"github.com/humanbelnik/load-balancer/internal/app"
)

type config struct {
	port     string
	host     string
	confpath string
}

func args() *config {
	var (
		port     = flag.String("port", "8080", "Port to listen on")
		host     = flag.String("host", "localhost", "Host to bind")
		confpath = flag.String("config", "./config/config.yaml", "Path to backend servers config file")
	)
	flag.Parse()
	log.Printf("using: port=%s, host=%s, config=%s", *port, *host, *confpath)

	return &config{
		port:     *port,
		host:     *host,
		confpath: *confpath,
	}
}

func main() {
	cfg := args()
	appAddr := cfg.host + ":" + cfg.port
	server, err := app.Setup(cfg.confpath, appAddr)
	if err != nil {
		log.Fatalf("%v", err)
	}
	app.Run(server)
}
