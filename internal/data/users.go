package data

import (
	"context"
	"errors"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrDuplicateUsersUsername = errors.New("unique violation for users username")
)

//go:generate mockgen -source=users.go -destination=../mocks/user_repo.go -package=mocks
type UserRepository interface {
	Insert(ctx context.Context, user *User) error
}

type User struct {
	ID             int64     `json:"id"`
	Username       string    `json:"username"`
	PasswordHash   string    `json:"-"`
	GameTag        *string   `json:"game_tag"`
	ProfilePicture *string   `json:"profile_picture"`
	CreatedAt      time.Time `json:"created_at"`
	Version        int32     `json:"-"`
}

func (u *User) SetPassword(plain string) error {
	hash, err := argon2id.CreateHash(plain, argon2id.DefaultParams)
	if err != nil {
		return err
	}

	u.PasswordHash = hash

	return nil
}

type UserModel struct {
	pool *pgxpool.Pool
}

func (m UserModel) Insert(ctx context.Context, user *User) error {
	query := `
		INSERT INTO users (username, password_hash, game_tag, profile_picture)
		VALUES ($1, $2, $3, $4)
		RETURNING id, username, password_hash, game_tag, profile_picture, created_at, version`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	args := []any{
		user.Username,
		user.PasswordHash,
		user.GameTag,
		user.ProfilePicture,
	}

	err := m.pool.QueryRow(ctx, query, args...).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.GameTag,
		&user.ProfilePicture,
		&user.CreatedAt,
		&user.Version,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		switch {
		case errors.As(err, &pgErr) && pgErr.Code == "23505":
			if ucErr, ok := uniqueConstraintErrors[pgErr.ConstraintName]; ok {
				return ucErr
			}
			return err
		default:
			return err
		}
	}

	return nil
}
