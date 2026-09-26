package data

import (
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNoRecord     = errors.New("no record found")
	ErrEditConflict = errors.New("edit conflict")
)

type Models struct {
	Users  UserRepository
	Tokens TokenRepository
}

func NewModels(pool *pgxpool.Pool) Models {
	return Models{
		Users:  UserModel{pool: pool},
		Tokens: TokenModel{pool: pool},
	}
}
