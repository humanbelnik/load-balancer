package api_ratelimiter

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/humanbelnik/load-balancer/internal/ratelimiter/ratelimiter"
)

type API struct {
	Limiter *ratelimiter.Limiter
	logger  *slog.Logger
}

type Option func(*API)

func WithLogger(logger *slog.Logger) Option {
	return func(a *API) {
		a.logger = logger
	}
}

func New(limiter *ratelimiter.Limiter, opts ...Option) *API {
	api := &API{
		Limiter: limiter,
		logger:  slog.Default(),
	}
	for _, opt := range opts {
		opt(api)
	}
	return api
}

type AddRequest struct {
	IP          string `json:"ip"`
	Capacity    int    `json:"capacity"`
	RefillEvery string `json:"refill_every"` // e.g. "2s"
}

type DeleteRequest struct {
	IP string `json:"ip"`
}

func (a *API) AddClient(w http.ResponseWriter, r *http.Request) {
	var req AddRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		a.logger.Warn("invalid JSON on AddClient", slog.Any("err", err))
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if req.IP == "" || req.Capacity <= 0 || req.RefillEvery == "" {
		a.logger.Warn("invalid fields on AddClient", slog.Any("request", req))
		http.Error(w, "missing or invalid fields", http.StatusBadRequest)
		return
	}

	dur, err := time.ParseDuration(req.RefillEvery)
	if err != nil {
		a.logger.Warn("invalid duration on AddClient", slog.String("raw", req.RefillEvery))
		http.Error(w, "invalid duration", http.StatusBadRequest)
		return
	}

	if err := a.Limiter.SetClient(req.IP, &req.Capacity, &dur); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		a.logger.Error("SetClient failed", slog.Any("err", err))
		return
	}
	a.logger.Info("client added", slog.String("ip", req.IP), slog.Int("capacity", req.Capacity), slog.Duration("refill", dur))
	w.WriteHeader(http.StatusCreated)
}

func (a *API) DeleteClient(w http.ResponseWriter, r *http.Request) {
	var req DeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		a.logger.Warn("invalid JSON on DeleteClient", slog.Any("err", err))
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if req.IP == "" {
		a.logger.Warn("missing IP on DeleteClient")
		http.Error(w, "missing IP", http.StatusBadRequest)
		return
	}

	if err := a.Limiter.RemoveClient(req.IP); err != nil {
		a.logger.Warn("DeleteClient failed", slog.String("ip", req.IP), slog.Any("err", err))
		http.Error(w, "client not found", http.StatusNotFound)
		return
	}
	a.logger.Info("client deleted", slog.String("ip", req.IP))
	w.WriteHeader(http.StatusOK)
}
