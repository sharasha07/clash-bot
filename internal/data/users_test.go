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

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := m.Insert(ctx, &user)
			assert.Equal(t, tt.err, err)
		})
	}
}
