package factory

import "github.com/humanbelnik/load-balancer/internal/balancer/server/server"

type Factory struct{}

func New() *Factory {
	return &Factory{}
}

func (f *Factory) Create(url string) (server.Server, error) {
	return server.New(url)
}
