package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/caarlos0/env/v11"
)

type application struct {
	logger *slog.Logger
	cfg    Config
}

type Config struct {
	Port int `env:"PORT,required"`

	Server struct {
		ReadTimeout  time.Duration `env:"SERVER_READ_TIMEOUT,required"`
		WriteTimeout time.Duration `env:"SERVER_WRITE_TIMEOUT,required"`
		IdleTimeout  time.Duration `env:"SERVER_IDLE_TIMEOUT,required"`
	}

	CORS struct {
		TrustedOrigins []string `env:"TRUSTED_ORIGINS,required"`
	}
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		logger.Error("failed to parse env vars", "err", err)
		os.Exit(1)
	}

	app := &application{
		cfg:    cfg,
		logger: logger,
	}

	if err := app.serve(); err != nil {
		logger.Error("server failed", "err", err)
		os.Exit(1)
	}
}
