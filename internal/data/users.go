package data

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/alexedwards/argon2id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sharasha07/clash-bot/internal/validator"
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
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id int64) error
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
	u.ValidateProfile(v)
	u.Password.Validate(v)
}

func (u *User) ValidateProfile(v *validator.Validator) {
	v.Check(strings.TrimSpace(u.Username) != "", "username", "must not be empty")
	v.Check(utf8.RuneCountInString(u.Username) <= 15, "username", "must be a maximum of 15 characters")

	if u.GameTag != nil {
		v.Check(*u.GameTag != "", "game_tag", "must not be empty")
		v.Check(strings.HasPrefix(*u.GameTag, "#"), "game_tag", "must start with #")
	}
}

func (u *User) IsAnonymous() bool {
	return u == AnonymousUser
}

type password struct {
	Plain string
	Hash  string
}

func (p *password) Validate(v *validator.Validator) {
	v.Check(p.Plain != "", "password", "must not be empty")
	v.Check(utf8.RuneCountInString(p.Plain) > 8, "password", "must be more than 8 characters")
	v.Check(utf8.RuneCountInString(p.Plain) <= 40, "password", "must be a maximum of 40 characters")
}

func (p *password) SetHash(plain string) error {
	hash, err := argon2id.CreateHash(plain, argon2id.DefaultParams)
	if err != nil {
		return err
	}

	p.Hash = hash

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
		user.Password.Hash,
		user.GameTag,
		user.ProfilePicture,
	}

	err := m.pool.QueryRow(ctx, query, args...).Scan(
		&user.ID,
		&user.Username,
		&user.Password.Hash,
		&user.GameTag,
		&user.ProfilePicture,
		&user.CreatedAt,
		&user.Version,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		switch {
		case errors.As(err, &pgErr) && pgErr.Code == "23505":
			switch pgErr.ConstraintName {
			case "users_username_unique":
				return ErrDuplicateUsersUsername
			default:
				return err
			}
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

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := m.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Password.Hash,
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
		&u.Password.Hash,
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

func (m UserModel) Update(ctx context.Context, user *User) error {
	query := `
		UPDATE users
		SET username = $1, password_hash = $2, game_tag = $3, profile_picture = $4, version = version + 1
		WHERE id = $5 and version = $6
		RETURNING version`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	args := []any{user.Username, user.Password.Hash, user.GameTag, user.ProfilePicture, user.ID, user.Version}

	err := m.pool.QueryRow(ctx, query, args...).Scan(
		&user.Version,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return ErrEditConflict
		case errors.As(err, &pgErr) && pgErr.Code == "23505":
			switch pgErr.ConstraintName {
			case "users_username_unique":
				return ErrDuplicateUsersUsername
			default:
				return err
			}
		default:
			return err
		}
	}

	return nil
}

func (m UserModel) Delete(ctx context.Context, id int64) error {
	query := `
		DELETE FROM users
		WHERE id = $1`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tag, err := m.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return ErrNoRecord
	}

	return nil
}
