package server

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"github.com/humanbelnik/load-balancer/internal/server/proxy"
)

var ErrBrokenURL = errors.New("broken URL")

/*
Describes API of a server abstraction.
*/
type Server interface {
	Serve(w http.ResponseWriter, r *http.Request) error
	SetAlive(alive bool)
	IsAlive() bool
	URL() string
}

type ServerInst struct {
	url   *url.URL
	proxy *proxy.Proxy
	mu    sync.RWMutex
	alive bool
}

func New(rawURL string) (*ServerInst, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrBrokenURL, err)
	}
	return &ServerInst{
		url:   parsed,
		proxy: proxy.New(parsed),
		alive: true,
	}, nil
}

func (s *ServerInst) URL() string {
	return s.url.String()
}

func (s *ServerInst) IsAlive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.alive
}

func (s *ServerInst) SetAlive(alive bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.alive = alive
}

func (s *ServerInst) Serve(w http.ResponseWriter, r *http.Request) error {
	err := s.proxy.ServeAndReport(w, r)
	/*
		err != nil
		on 5xx HTTP errors.
	*/
	if err != nil {
		s.SetAlive(false)
		return err
	}
	return nil
}
