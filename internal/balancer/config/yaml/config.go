package yaml_config

import (
	"errors"
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

var (
	ErrCannotLoad = errors.New("cannot load config")
)

type YAMLLoader struct{}

func New() *YAMLLoader {
	return &YAMLLoader{}
}

type URLs struct {
	URLs []string `yaml:"servers"`
}

func (l *YAMLLoader) Load(path string) ([]string, error) {
	var cfg URLs

	err := cleanenv.ReadConfig(path, &cfg)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCannotLoad, err)
	}

	return cfg.URLs, nil
}
