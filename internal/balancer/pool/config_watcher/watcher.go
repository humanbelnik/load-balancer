package config_watcher

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

type Loader interface {
	Load(path string) ([]string, error)
}

type PoolUpdater interface {
	Update(urls []string) error
}

/*
Watcher binds to a SIGHUP OS signal.
If signal occurs, load config and update pool.
*/
type ConfigWatcher struct {
	loader  Loader
	onError func(error)
}

func New(loader Loader, onError func(error)) *ConfigWatcher {
	return &ConfigWatcher{
		loader:  loader,
		onError: onError,
	}
}

func (w *ConfigWatcher) Watch(path string, updater PoolUpdater) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGHUP)

	go func() {
		for range signals {
			urls, err := w.loader.Load(path)
			if err != nil {
				w.onError(fmt.Errorf("failed to load config: %w", err))
				continue
			}

			if err := updater.Update(urls); err != nil {
				w.onError(fmt.Errorf("failed to update server pool: %w", err))
				continue
			}

			log.Println("pool updated")
		}
	}()
}

func DefaultOnError(err error) {
	log.Println(err)
}
