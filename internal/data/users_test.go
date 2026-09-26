package data

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestUserInsert(t *testing.T) {
	tests := []struct {
		name     string
		username string
		password string
		err      error
	}{
		{
			name:     "nil error",
			username: "saba",
			err:      nil,
		},
		{
			name:     "username unique violation error",
			username: "luka",
			err:      ErrDuplicateUsersUsername,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := newTestPool(t)
			m := UserModel{pool: pool}

			user := User{Username: tt.username}

			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()

			err := m.Insert(ctx, &user)
			assert.Equal(t, tt.err, err)

			if err == nil {
				assert.Equal(t, int64(2), user.ID)
				assert.Equal(t, tt.username, user.Username)
				assert.Nil(t, user.GameTag)
				assert.Nil(t, user.ProfilePicture)
				assert.Equal(t, int32(1), user.Version)
			}
		})
	}
}

func TestUserGetByID(t *testing.T) {
	tests := []struct {
		name string
		id   int64
		err  error
	}{
		{"no record", 2, ErrNoRecord},
		{"success", 1, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := newTestPool(t)
			m := UserModel{pool: pool}

			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()

			_, err := m.GetByID(ctx, tt.id)
			assert.Equal(t, tt.err, err)
		})
	}
}
