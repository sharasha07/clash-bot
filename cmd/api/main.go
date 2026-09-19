package main

import (
	"fmt"
	"log"

	"github.com/caarlos0/env"
)

type Config struct {
	Port int `env:"PORT,required"`
}

func main() {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		log.Fatal(err)
	}

	fmt.Println(cfg.Port)
}
