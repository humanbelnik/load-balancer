package proxy

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type Proxy struct {
	target *url.URL
	proxy  *httputil.ReverseProxy
}

func New(target *url.URL) *Proxy {
	p := &Proxy{
		target: target,
		proxy:  httputil.NewSingleHostReverseProxy(target),
	}
	p.proxy.ErrorHandler = p.onError
	return p
}

func (p *Proxy) ServeAndReport(w http.ResponseWriter, r *http.Request) error {
	ri := &responseInterceptor{ResponseWriter: w}
	p.proxy.ServeHTTP(ri, r)

	if ri.failed {
		return fmt.Errorf("server %s failed", p.target)
	}
	return nil
}

func (p *Proxy) onError(w http.ResponseWriter, r *http.Request, err error) {
	msg := formatProxyError(err)
	code := classifyStatusCode(err)

	http.Error(w, msg, code)
}

func formatProxyError(err error) string {
	switch {
	case isTimeout(err):
		return "server timeout"
	case isNetError(err):
		return fmt.Sprintf("network error: %v", err)
	default:
		return fmt.Sprintf("proxy error: %v", err)
	}
}

func classifyStatusCode(err error) int {
	if isTimeout(err) {
		return http.StatusGatewayTimeout
	}
	return http.StatusBadGateway
}

func isTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func isNetError(err error) bool {
	var opErr *net.OpError
	return errors.As(err, &opErr)
}

type responseInterceptor struct {
	http.ResponseWriter
	failed bool
}

func (ri *responseInterceptor) WriteHeader(code int) {
	if code >= 500 {
		ri.failed = true
	}
	ri.ResponseWriter.WriteHeader(code)
}
