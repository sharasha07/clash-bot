package data

import (
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrUniqueViolation = errors.New("unique violation error")
)

type Models struct {
	Users UserModelInterface
}

func NewDBModels(pool *pgxpool.Pool) Models {
	return Models{
		Users: UserModel{pool: pool},
	}
}
