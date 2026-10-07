//go:build integration

package llm

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/redis/go-redis/v9"
	"github.com/sharasha07/clash-bot/internal/clash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type geminiClientConfig struct {
	BaseURL string        `env:"GEMINI_BASE_URL,required"`
	ApiKey  string        `env:"GEMINI_API_KEY,required"`
	Model   string        `env:"GEMINI_MODEL,required"`
	Timeout time.Duration `env:"GEMINI_TIMEOUT,required"`
}

type clashAPIClientConfig struct {
	BaseURL        string        `env:"CLASH_ROYALE_BASE_URL,required"`
	APIToken       string        `env:"CLASH_ROYALE_API_TOKEN,required"`
	Timeout        time.Duration `env:"CLASH_ROYALE_TIMEOUT,required"`
	RedisURL       string        `env:"REDIS_URL,required"`
	MaxResultBytes int           `env:"CLASH_ROYALE_MAX_RESULT_BYTES,required"`
}

func TestGeminiClient(t *testing.T) {
	var cfg1 geminiClientConfig
	require.NoError(t, env.Parse(&cfg1))

	var cfg2 clashAPIClientConfig
	require.NoError(t, env.Parse(&cfg2))

	opt, err := redis.ParseURL(cfg2.RedisURL)
	require.NoError(t, err)

	redisClient := redis.NewClient(opt)
	crClient := clash.NewAPIClient(cfg2.BaseURL, cfg2.APIToken, cfg2.Timeout, redisClient, slog.New(slog.DiscardHandler), cfg2.MaxResultBytes)

	client, err := NewGemini(cfg1.ApiKey, cfg1.BaseURL, cfg1.Model, cfg1.Timeout, crClient)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()

	resp, err := client.GenerateReply(ctx, "what is my tag", nil)
	require.NoError(t, err)

	assert.NotEmpty(t, resp)
}
