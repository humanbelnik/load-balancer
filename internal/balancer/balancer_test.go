package balancer

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/humanbelnik/load-balancer/internal/mocks"
	"github.com/humanbelnik/load-balancer/internal/server/server"
)

func makeMockServer(t *testing.T, code int, err error) *mocks.Server {
	t.Helper()
	s := new(mocks.Server)
	s.On("URL").Return("http://mock")
	s.On("IsAlive").Return(true)
	s.On("Serve", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		w := args.Get(0).(http.ResponseWriter)
		if code != 0 {
			w.WriteHeader(code)
		}
	}).Return(err)
	return s
}

func TestBalancer_Success(t *testing.T) {
	s := makeMockServer(t, http.StatusOK, nil)

	pool := new(mocks.Pool)
	pool.On("Alive").Return([]server.Server{s}, nil)

	policy := new(mocks.Policy)
	policy.On("Select", mock.Anything).Return(s, nil)

	b := New(pool, policy)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	b.Serve(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
}

func TestBalancer_ClientError_4xx(t *testing.T) {
	s := makeMockServer(t, http.StatusBadRequest, nil)

	pool := new(mocks.Pool)
	pool.On("Alive").Return([]server.Server{s}, nil)

	policy := new(mocks.Policy)
	policy.On("Select", mock.Anything).Return(s, nil)

	b := New(pool, policy)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	b.Serve(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	s.AssertNotCalled(t, "SetAlive", false)
}

func TestBalancer_ServerError_5xx(t *testing.T) {
	s := new(mocks.Server)
	s.On("URL").Return("http://mock")
	s.On("IsAlive").Return(true)
	s.On("Serve", mock.Anything, mock.Anything).Return(errors.New("internal"))
	s.On("SetAlive", false).Once()

	pool := new(mocks.Pool)
	pool.On("Alive").Return([]server.Server{s}, nil)

	policy := new(mocks.Policy)
	policy.On("Select", mock.Anything).Return(s, nil)

	b := New(pool, policy)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	b.Serve(rr, req)

	require.Equal(t, http.StatusBadGateway, rr.Code)
}

func TestBalancer_PoolError(t *testing.T) {
	pool := new(mocks.Pool)
	pool.On("Alive").Return(nil, errors.New("pool failure"))

	policy := new(mocks.Policy)

	b := New(pool, policy)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	b.Serve(rr, req)

	require.Equal(t, http.StatusServiceUnavailable, rr.Code)
}

func TestBalancer_PolicyError(t *testing.T) {
	s := makeMockServer(t, http.StatusOK, nil)

	pool := new(mocks.Pool)
	pool.On("Alive").Return([]server.Server{s}, nil)

	policy := new(mocks.Policy)
	policy.On("Select", mock.Anything).Return(nil, errors.New("policy fail"))

	b := New(pool, policy)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	b.Serve(rr, req)

	require.Equal(t, http.StatusServiceUnavailable, rr.Code)
}
