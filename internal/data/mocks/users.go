package mocks

import (
	"context"
	"time"

	"github.com/sharasha07/clash-bot/internal/data"
)

type UserModel struct {
	users  map[int]data.User
	nextID int
}

func NewUserModel() *UserModel {
	return &UserModel{
		users: map[int]data.User{
			1: {
				ID:       1,
				Username: "shaba",
				Version:  4,
			},
		},
		nextID: 2,
	}
}

func (m *UserModel) Insert(ctx context.Context, user *data.User) error {
	for _, u := range m.users {
		if u.Username == user.Username {
			return data.ErrUniqueViolation
		}
	}

	user.ID = int64(m.nextID)
	user.CreatedAt = time.Now()
	user.Version = 1

	m.users[m.nextID] = *user
	m.nextID++

	return nil
}
