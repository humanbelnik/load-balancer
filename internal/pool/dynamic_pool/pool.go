package dynamic_pool

import (
	"errors"
	"fmt"
	"log"
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

func (p *Dynamic) add(s server.Server) error {
	url := s.URL()
	if _, exists := p.servers[url]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicateURL, url)
	}
	p.servers[url] = s
	p.urls[url] = struct{}{}
	return nil
}

func (p *Dynamic) remove(url string) error {
	delete(p.servers, url)
	delete(p.urls, url)

	return nil
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
	log.Println("update", urls)
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

		if err = p.add(new); err != nil {
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
		if err := p.remove(url); err != nil {
			return err
		}
	}

	return nil
}
