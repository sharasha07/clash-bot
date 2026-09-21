package data

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/alexedwards/argon2id"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sharasha07/clash-bot/internal/validator"
)

type UserModelInterface interface {
	Insert(ctx context.Context, user *User) error
}

type User struct {
	ID             int64     `json:"id"`
	Username       string    `json:"username"`
	Password       password  `json:"-"`
	GameTag        *string   `json:"game_tag"`
	ProfilePicture *string   `json:"profile_picture"`
	CreatedAt      time.Time `json:"created_at"`
	Version        int32     `json:"-"`
}

func (u *User) Validate(v *validator.Validator) {
	v.Check(u.Username != "", "username", "must be provided")
	v.Check(utf8.RuneCountInString(u.Username) <= 10, "username", "must not be more than 10 characters")

	v.Check(u.Password.plain != "", "password", "must be provided")
	v.Check(utf8.RuneCountInString(u.Password.plain) >= 8, "password", "must be at least 8 characters")
	v.Check(utf8.RuneCountInString(u.Password.plain) <= 30, "password", "must not be more than 30 characters")

	if u.Password.hash == "" {
		panic("missing password hash for the user")
	}

	if u.GameTag != nil {
		v.Check(*u.GameTag != "", "game_tag", "must not be empty")
		v.Check(strings.HasPrefix(*u.GameTag, "#"), "game_tag", "must start with #")
	}
}

type password struct {
	plain string
	hash  string
}

func (p *password) Set(plain string) error {
	hash, err := argon2id.CreateHash(plain, argon2id.DefaultParams)
	if err != nil {
		return err
	}

	p.plain = plain
	p.hash = hash

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
		user.Password.hash,
		user.GameTag,
		user.ProfilePicture,
	}

	err := m.pool.QueryRow(ctx, query, args...).Scan(
		&user.ID,
		&user.Username,
		&user.Password.hash,
		&user.GameTag,
		&user.ProfilePicture,
		&user.CreatedAt,
		&user.Version,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		switch {
		case errors.As(err, &pgErr) && pgErr.Code == "23505":
			return ErrUniqueViolation
		default:
			return err
		}
	}

	return nil
}
