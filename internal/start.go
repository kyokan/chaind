package internal

import (
	"github.com/kyokan/chaind/pkg/config"
		"github.com/kyokan/chaind/internal/proxy"
	"os"
	"os/signal"
	"syscall"
	"github.com/kyokan/chaind/pkg/log"
	"github.com/kyokan/chaind/internal/audit"
	"github.com/kyokan/chaind/internal/cache"
	"github.com/inconshreveable/log15"
	)

func Start(cfg *config.Config) error {
	if err := config.ValidateConfig(cfg); err != nil {
	    return err
	}

	logger := log.NewLog("")
	lvl, err := log15.LvlFromString(cfg.LogLevel)
	if err != nil {
		logger.Warn("invalid log level, falling back to INFO", "level", cfg.LogLevel)
		lvl = log15.LvlInfo
	}
	log.SetLevel(lvl)

	sw := proxy.NewBackendSwitch(cfg.Backends)
	if err := sw.Start(); err != nil {
		return err
	}

	cacher := cache.NewRedisCacher(cfg.RedisConfig)
	if err := cacher.Start(); err != nil {
		return err
	}

	auditor, err := audit.NewLogAuditor(cfg.LogAuditorConfig)
	if err != nil {
		return err
	}

	fHelper := proxy.NewBlockHeightWatcher(sw)
	if err := fHelper.Start(); err != nil {
		return err
	}

	prox := proxy.NewProxy(sw, auditor, cacher, fHelper, cfg)
	if err := prox.Start(); err != nil {
		return err
	}

	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		logger.Info("interrupted, shutting down")
		if err := sw.Stop(); err != nil {
			logger.Error("failed to stop backend switch", "err", err)
		}
		if err := cacher.Stop(); err != nil {
			logger.Error("failed to stop cacher", "err", err)
		}
		if err := fHelper.Stop(); err != nil {
			logger.Error("failed to stop finalization helper", "err", err)
		}
		if err := prox.Stop(); err != nil {
			logger.Error("failed to stop proxy", "err", err)
		}
		done <- true
	}()

	<-done
	logger.Info("goodbye")
	return nil
}
