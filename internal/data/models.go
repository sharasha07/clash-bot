package data

import "github.com/jackc/pgx/v5/pgxpool"

type Models struct {
}

func NewDBModels(pool *pgxpool.Pool) Models {
	return Models{}
}
