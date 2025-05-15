package rr

import (
	"errors"
	"sync"

	"github.com/humanbelnik/load-balancer/internal/balancer/server/server"
)

var (
	ErrNoServers = errors.New("no servers")
)

type RoundRobinPolicy struct {
	m    sync.Mutex
	next int
}

func New() *RoundRobinPolicy {
	return &RoundRobinPolicy{}
}

func (p *RoundRobinPolicy) Select(servers []server.Server) (server.Server, error) {
	p.m.Lock()
	defer p.m.Unlock()

	if len(servers) == 0 {
		return nil, ErrNoServers
	}

	s := servers[p.next%len(servers)]
	p.next++
	return s, nil
}
