package yaml_config

import (
	"errors"
	"fmt"

	"github.com/humanbelnik/load-balancer/internal/ratelimiter/config"
	"github.com/ilyakaznacheev/cleanenv"
)

var (
	ErrCannotLoadRateLimiter = errors.New("cannot load rate limiter config")
)

type RateLimiterYAMLLoader struct{}

func NewRateLimiterLoader() *RateLimiterYAMLLoader {
	return &RateLimiterYAMLLoader{}
}

type RateLimiterWrapper struct {
	RateLimiter config.RateLimiterConfig `yaml:"rate_limiter"`
}

func (l *RateLimiterYAMLLoader) Load(path string) (config.RateLimiterConfig, error) {
	var cfg RateLimiterWrapper

	err := cleanenv.ReadConfig(path, &cfg)
	if err != nil {
		return config.RateLimiterConfig{}, fmt.Errorf("%w: %w", ErrCannotLoadRateLimiter, err)
	}

	return cfg.RateLimiter, nil
}
