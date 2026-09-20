package main

import "log/slog"

func newTestApplication() *application {
	return &application{
		logger: slog.New(slog.DiscardHandler),
		cfg:    Config{},
	}
}
