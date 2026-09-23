package main

import (
	"log/slog"
	"testing"

	"github.com/sharasha07/clash-bot/internal/data"
	"github.com/sharasha07/clash-bot/internal/mocks"

	"go.uber.org/mock/gomock"
)

func newTestApplication(t *testing.T) *application {
	return &application{
		logger:   slog.New(slog.DiscardHandler),
		cfg:      Config{},
		validate: newValidate(),
		models: data.Models{
			Users: mocks.NewMockUserModelInterface(gomock.NewController(t)),
		},
	}
}
