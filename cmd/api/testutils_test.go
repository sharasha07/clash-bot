package main

import (
	"log/slog"

	"github.com/sharasha07/clash-bot/internal/data"
	"github.com/sharasha07/clash-bot/internal/data/mocks"
)

func newMockModels() data.Models {
	return data.Models{
		Users: mocks.NewUserModel(),
	}
}

func newTestApplication() *application {
	return &application{
		logger: slog.New(slog.DiscardHandler),
		cfg:    Config{},
		models: newMockModels(),
	}
}
