package main

import (
	"log/slog"
	"testing"

	"github.com/sharasha07/clash-bot/internal/data"
	"github.com/sharasha07/clash-bot/internal/mocks"
	"go.uber.org/mock/gomock"
)

func newTestApplication(t *testing.T) *application {
	ctrl := gomock.NewController(t)

	return &application{
		logger: slog.New(slog.DiscardHandler),
		cfg:    Config{},
		models: data.Models{
			Users:  mocks.NewMockUserRepository(ctrl),
			Tokens: mocks.NewMockTokenRepository(ctrl),
			Chats:  mocks.NewMockChatRepository(ctrl),
		},
	}
}
