package config

import "time"

type RateLimiterConfig struct {
	DefaultCapacity   int           `yaml:"default_capacity"`
	DefaultRefillRate time.Duration `yaml:"default_refill_rate"`
}
