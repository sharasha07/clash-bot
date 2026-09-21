package data

import (
	"context"
	"testing"
	"time"

	"github.com/sharasha07/clash-bot/internal/assert"
)

func TestInsert(t *testing.T) {
	tests := []struct {
		name     string
		username string
		err      error
	}{
		{
			name:     "Success",
			username: "luka",
			err:      nil,
		},
		{
			name:     "Username Unique Violation",
			username: "shaba",
			err:      ErrUniqueViolation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := newTestPool(t)

			m := UserModel{pool: pool}

			user := User{Username: tt.username}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := m.Insert(ctx, &user)
			assert.Equal(t, tt.err, err)
		})
	}
}
