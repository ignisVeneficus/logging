package logging

import (
	"strings"
	"sync"
	"sync/atomic"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.mau.fi/zeroconfig"
)

type LoggingConfig struct {
	zeroconfig.Config `yaml:",inline"`
	Loggers           map[string]string `yaml:"loggers"`
}

type resolvedConfig struct {
	loggers sync.Map // string -> zerolog.Level
}

var globalConfig atomic.Pointer[resolvedConfig]

func (c *resolvedConfig) resolveLevel(name string) *zerolog.Level {
	if c == nil {
		return nil
	}

	v, ok := c.loggers.Load(name)
	if ok {
		if v == nil {
			return nil
		}
		return v.(*zerolog.Level)
	}

	parts := strings.Split(name, "/")

	// backward search
	for i := len(parts) - 1; i > 0; i-- {
		key := strings.Join(parts[:i], "/")

		v, ok := c.loggers.Load(key)
		if !ok {
			continue
		}
		var lvl *zerolog.Level
		if v != nil {
			lvl = v.(*zerolog.Level)
		}

		// forward cache fill
		for j := i + 1; j <= len(parts); j++ {
			cacheKey := strings.Join(parts[:j], "/")
			c.loggers.Store(cacheKey, lvl)
		}
		return lvl
	}
	for j := 1; j <= len(parts); j++ {
		cacheKey := strings.Join(parts[:j], "/")
		c.loggers.Store(cacheKey, nil)
	}
	return nil
}

func Configure(cfg LoggingConfig) {
	resolved := &resolvedConfig{}

	for k, v := range cfg.Loggers {
		level, err := zerolog.ParseLevel(v)

		if err != nil {
			log.Logger.Error().Str("level", v).Err(err).Msg("invalid level")
			continue
		}
		resolved.loggers.Store(k, &level)
	}
	globalConfig.Store(resolved)
}

func ConfigureLogger(logger zerolog.Logger, name string) zerolog.Logger {
	cfg := globalConfig.Load()
	if cfg == nil {
		return logger
	}

	level := cfg.resolveLevel(name)
	if level == nil {
		return logger
	}
	return logger.Level(*level)
}
