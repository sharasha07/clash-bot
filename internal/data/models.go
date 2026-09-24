package data

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

var uniqueConstraintErrors = map[string]error{
	"users_username_unique": ErrDuplicateUsersUsername,
}

type Models struct {
	Users UserRepository
}

func NewModels(pool *pgxpool.Pool) Models {
	return Models{
		Users: UserModel{pool: pool},
	}
}
