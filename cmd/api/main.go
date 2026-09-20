package main

import (
	"fmt"
	"log"
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Port int `env:"PORT,required"`

	Server struct {
		ReadTimeout  time.Duration `env:"SERVER_READ_TIMEOUT,required"`
		WriteTimeout time.Duration `env:"SERVER_WRITE_TIMEOUT,required"`
		IdleTimeout  time.Duration `env:"SERVER_IDLE_TIMEOUT,required"`
	}
}

type application struct {
	cfg Config
}

func main() {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		log.Fatal(err)
	}

	app := &application{
		cfg: cfg,
	}

	fmt.Println(cfg)

	if err := app.serve(); err != nil {
		log.Fatal(err)
	}
}
