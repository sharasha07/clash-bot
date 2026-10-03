//go:build integration

package llm

import (
	"context"
	"testing"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/sharasha07/clash-bot/internal/clash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type geminiConfig struct {
	BaseURL string        `env:"GEMINI_BASE_URL,required"`
	ApiKey  string        `env:"GEMINI_API_KEY,required"`
	Model   string        `env:"GEMINI_MODEL,required"`
	Timeout time.Duration `env:"GEMINI_TIMEOUT,required"`
}

type clashConfig struct {
	BaseURL        string        `env:"CLASH_ROYALE_BASE_URL,required"`
	APIToken       string        `env:"CLASH_ROYALE_API_TOKEN,required"`
	Timeout        time.Duration `env:"CLASH_ROYALE_TIMEOUT,required"`
	MaxResultBytes int           `env:"CLASH_ROYALE_MAX_RESULT_BYTES,required"`
}

func TestLLM(t *testing.T) {
	var cfg1 geminiConfig
	require.NoError(t, env.Parse(&cfg1))

	var cfg2 clashConfig
	require.NoError(t, env.Parse(&cfg2))

	crClient := clash.NewClient(cfg2.BaseURL, cfg2.APIToken, cfg2.Timeout, cfg2.MaxResultBytes)

	client, err := NewGeminiLLM(cfg1.ApiKey, cfg1.BaseURL, cfg1.Model, cfg1.Timeout, crClient)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()

	resp, err := client.GenerateReply(ctx, "what is my tag", nil)
	require.NoError(t, err)

	assert.NotEmpty(t, resp)
}
