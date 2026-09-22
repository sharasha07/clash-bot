package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sharasha07/clash-bot/internal/data"
)

type application struct {
	logger *slog.Logger
	cfg    Config
	models data.Models
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

	Limiter struct {
		RPS     int  `env:"LIMITER_RPS,required"`
		Burst   int  `env:"LIMITER_BURST,required"`
		Enabled bool `env:"LIMITER_ENABLED,required"`
	}

	DB struct {
		DSN         string        `env:"DB_DSN,required"`
		MinConns    int32         `env:"DB_MIN_CONNS,required"`
		MaxConns    int32         `env:"DB_MAX_CONNS,required"`
		MaxIdleTime time.Duration `env:"DB_MAX_IDLE_TIME,required"`
	}
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		logger.Error("failed to parse env vars", "err", err)
		os.Exit(1)
	}

	pool, err := connectToDB(cfg)
	if err != nil {
		logger.Error("couldn't connect to db", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	logger.Info("successfully connected to db")

	app := &application{
		logger: logger,
		cfg:    cfg,
		models: data.NewDBModels(pool),
	}

	if err := app.serve(); err != nil {
		logger.Error("server failed", "err", err)
		os.Exit(1)
	}
}

func connectToDB(cfg Config) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DB.DSN)
	if err != nil {
		return nil, err
	}

	poolConfig.MinConns = cfg.DB.MinConns
	poolConfig.MaxConns = cfg.DB.MaxConns
	poolConfig.MaxConnIdleTime = cfg.DB.MaxIdleTime

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}
