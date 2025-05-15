package dynamic_pool

import (
	"errors"
	"fmt"
	"sync"

	"github.com/humanbelnik/load-balancer/internal/server/server"
)

var (
	ErrDuplicateURL   = errors.New("server with such url already present")
	ErrNoServers      = errors.New("no servers")
	ErrUnableToUpdate = errors.New("unable to update")
)

type Factory interface {
	Create(url string) (server.Server, error)
}

/*
Dynamic since it expand/shrink it's size based on the current configuration.
*/
type Dynamic struct {
	// Since build-in map is not thread-safe.
	m sync.RWMutex

	// For faster pool update.
	urls map[string]struct{}

	// Gonna use URLs as keys, 'http://localhost:9001' as an example.
	servers map[string]server.Server

	// Used to create new Server instances on Update call.
	serverFactory Factory
}

func New(factory Factory) *Dynamic {
	return &Dynamic{
		servers:       make(map[string]server.Server),
		urls:          make(map[string]struct{}),
		serverFactory: factory,
	}
}

func (p *Dynamic) Add(s server.Server) error {
	p.m.Lock()
	defer p.m.Unlock()

	url := s.URL()
	if _, exists := p.servers[url]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicateURL, url)
	}
	p.servers[url] = s
	p.urls[url] = struct{}{}
	return nil
}

func (p *Dynamic) Remove(url string) error {
	p.m.Lock()
	defer p.m.Unlock()

	delete(p.servers, url)
	delete(p.urls, url)

	return nil
}

func (p *Dynamic) Get(url string) (server.Server, bool) {
	p.m.RLock()
	defer p.m.RUnlock()

	s, ok := p.servers[url]
	return s, ok
}

func (p *Dynamic) All() ([]server.Server, error) {
	p.m.RLock()
	defer p.m.RUnlock()

	result := make([]server.Server, 0, len(p.servers))
	for _, srv := range p.servers {
		result = append(result, srv)
	}
	if len(result) == 0 {
		return nil, ErrNoServers
	}
	return result, nil
}

func (p *Dynamic) Alive() ([]server.Server, error) {
	p.m.RLock()
	defer p.m.RUnlock()

	result := make([]server.Server, 0)
	for _, s := range p.servers {
		if s.IsAlive() {
			result = append(result, s)
		}
	}

	if len(result) == 0 {
		return nil, ErrNoServers
	}
	return result, nil
}

func (p *Dynamic) Update(urls []string) error {
	p.m.Lock()
	defer p.m.Unlock()

	for _, url := range urls {
		if _, exists := p.urls[url]; exists {
			continue
		}

		new, err := p.serverFactory.Create(url)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrUnableToUpdate, err)
		}
		if err = p.Add(new); err != nil {
			return err
		}
	}

	/*
		Delete servers that are not present in new configuration.
	*/
	urlsSet := make(map[string]struct{}, len(urls))
	for _, url := range urls {
		urlsSet[url] = struct{}{}
	}
	for url := range p.urls {
		if _, exists := urlsSet[url]; exists {
			continue
		}
		if err := p.Remove(url); err != nil {
			return err
		}
	}

	return nil
}
