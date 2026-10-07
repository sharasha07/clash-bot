//go:build integration

package clash

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

const playerTag = "#U8RYGC8GU"

type apiClientConfig struct {
	BaseURL        string        `env:"CLASH_ROYALE_BASE_URL,required"`
	APIToken       string        `env:"CLASH_ROYALE_API_TOKEN,required"`
	Timeout        time.Duration `env:"CLASH_ROYALE_TIMEOUT,required"`
	RedisURL       string        `env:"REDIS_URL,required"`
	MaxResultBytes int           `env:"CLASH_ROYALE_MAX_RESULT_BYTES,required"`
}

func TestAPIClient(t *testing.T) {
	var cfg apiClientConfig
	require.NoError(t, env.Parse(&cfg))

	opt, err := redis.ParseURL(cfg.RedisURL)
	require.NoError(t, err)

	redisClient := redis.NewClient(opt)
	c := NewAPIClient(cfg.BaseURL, cfg.APIToken, cfg.Timeout, redisClient, slog.New(slog.DiscardHandler), cfg.MaxResultBytes)

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
}
