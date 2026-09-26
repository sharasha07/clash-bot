package data

import (
	"context"
	"errors"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrDuplicateUsersUsername = errors.New("unique violation for users username")
)

var AnonymousUser *User

//go:generate mockgen -source=users.go -destination=../mocks/user_repo.go -package=mocks
type UserRepository interface {
	Insert(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id int64) (User, error)
	GetByUsername(ctx context.Context, username string) (User, error)
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

func (u *User) IsAnonymous() bool {
	return u == AnonymousUser
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

func (m UserModel) GetByID(ctx context.Context, id int64) (User, error) {
	query := `
		SELECT id, username, password_hash, game_tag, profile_picture, created_at, version
		FROM users
		WHERE id = $1`

	var user User

	err := m.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.GameTag,
		&user.ProfilePicture,
		&user.CreatedAt,
		&user.Version,
	)

	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return User{}, ErrNoRecord
		default:
			return User{}, err
		}
	}

	return user, nil
}

func (m UserModel) GetByUsername(ctx context.Context, username string) (User, error) {
	query := `
		SELECT id, username, password_hash, game_tag, profile_picture, created_at, version
		FROM users
		WHERE username = $1`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var u User
	err := m.pool.QueryRow(ctx, query, username).Scan(
		&u.ID,
		&u.Username,
		&u.PasswordHash,
		&u.GameTag,
		&u.ProfilePicture,
		&u.CreatedAt,
		&u.Version,
	)

	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return User{}, ErrNoRecord
		default:
			return User{}, err
		}
	}

	return u, nil
}
