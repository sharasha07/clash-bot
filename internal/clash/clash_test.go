//go:build integration

package clash

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/stretchr/testify/require"
)

const playerTag = "#U8RYGC8GU"

type apiClientConfig struct {
	BaseURL        string        `env:"CLASH_ROYALE_BASE_URL,required"`
	APIToken       string        `env:"CLASH_ROYALE_API_TOKEN,required"`
	Timeout        time.Duration `env:"CLASH_ROYALE_TIMEOUT,required"`
	MaxResultBytes int           `env:"CLASH_ROYALE_MAX_RESULT_BYTES,required"`
}

func TestAPIClient(t *testing.T) {
	var cfg apiClientConfig
	require.NoError(t, env.Parse(&cfg))
	c := NewAPIClient(cfg.BaseURL, cfg.APIToken, cfg.Timeout, cfg.MaxResultBytes)

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	t.Run("GetPlayer", func(t *testing.T) {
		got, err := c.GetPlayer(ctx, playerTag)
		require.NoError(t, err)
		require.True(t, json.Valid([]byte(got)))

		got, err = c.GetPlayer(ctx, "#unknown")
		require.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("GetPlayerBattleLog", func(t *testing.T) {
		got, err := c.GetPlayerBattleLog(ctx, playerTag, 999)
		require.NoError(t, err)
		require.True(t, json.Valid([]byte(got)))
	})

	t.Run("GetPlayersUpcomingChests", func(t *testing.T) {
		got, err := c.GetPlayersUpcomingChests(ctx, playerTag, 10)
		require.NoError(t, err)
		require.True(t, json.Valid([]byte(got)))
	})

	t.Run("ResponseTooLarge", func(t *testing.T) {
		c := NewAPIClient(cfg.BaseURL, cfg.APIToken, cfg.Timeout, 1)

		_, err := c.GetPlayer(ctx, playerTag)
		require.ErrorIs(t, err, ErrResponseTooLarge)
	})
}
