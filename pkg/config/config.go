package config

import (
	"github.com/gnames/gnsys"
	"github.com/rs/zerolog/log"
)

type Config struct {
	CacheDir      string
	PgHost        string
	PgUser        string
	PgPass        string
	PgDb          string
	ForceDownload bool
}

type Option func(*Config)

func OptCacheDir(s string) Option {
	return func(cfg *Config) {
		s, err := gnsys.ConvertTilda(s)
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}
		cfg.CacheDir = s
	}
}

func OptPgHost(s string) Option {
	return func(cfg *Config) {
		cfg.PgHost = s
	}
}

func OptPgUser(s string) Option {
	return func(cfg *Config) {
		cfg.PgUser = s
	}
}

func OptPgPass(s string) Option {
	return func(cfg *Config) {
		cfg.PgPass = s
	}
}

func OptPgDb(s string) Option {
	return func(cfg *Config) {
		cfg.PgDb = s
	}
}

func OptForceDownload(b bool) Option {
	return func(cfg *Config) {
		cfg.ForceDownload = b
	}
}

func New(opts ...Option) Config {
	cacheDir, _ := gnsys.ConvertTilda("~/.cache/gndict")
	res := Config{
		CacheDir: cacheDir,
		PgHost:   "0.0.0.0",
		PgUser:   "postgres",
		PgPass:   "postgres",
		PgDb:     "gnames",
	}
	for _, opt := range opts {
		opt(&res)
	}
	return res
}
