package data

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sharasha07/clash-bot/internal/validator"
)

//go:generate mockgen -source=chats.go -destination=../mocks/chat_repo.go -package=mocks
type ChatRepository interface {
	GetAll(ctx context.Context, user_id int64, name string, filters Filters) ([]Chat, error)
	Insert(ctx context.Context, chat *Chat) error
	Get(ctx context.Context, id, user_id int64) (Chat, error)
	Update(ctx context.Context, chat *Chat) error
	Delete(ctx context.Context, id, user_id int64) error
	Touch(ctx context.Context, id, user_id int64) (Chat, error)
}

type Chat struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Name      string    `json:"name"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedAt time.Time `json:"created_at"`
	Version   int32     `json:"-"`
}

func (c Chat) Validate(v *validator.Validator) {
	v.Check(strings.TrimSpace(c.Name) != "", "name", "must not be empty")
	v.Check(utf8.RuneCountInString(c.Name) <= 10, "name", "must be a maximum of 10")
}

type ChatModel struct {
	pool *pgxpool.Pool
}

func (m ChatModel) GetAll(ctx context.Context, user_id int64, name string, filters Filters) ([]Chat, error) {
	query := fmt.Sprintf(`
		SELECT id, user_id, name, updated_at, created_at, version
		FROM chats
		WHERE user_id = $1 AND ($2 = '' OR name ILIKE '%%' || $2 || '%%')
		ORDER BY %s %s
		LIMIT $3 OFFSET $4`, filters.SortColumn(), filters.SortOrder())

	args := []any{user_id, name, filters.PageSize, filters.Offset()}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := m.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	chats := []Chat{}
	for rows.Next() {
		var chat Chat
		err := rows.Scan(
			&chat.ID,
			&chat.UserID,
			&chat.Name,
			&chat.UpdatedAt,
			&chat.CreatedAt,
			&chat.Version,
		)

		if err != nil {
			return nil, err
		}

		chats = append(chats, chat)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return chats, nil
}

func (m ChatModel) Insert(ctx context.Context, chat *Chat) error {
	query := `
		INSERT INTO chats (user_id, name)
		VALUES ($1, $2)
		RETURNING id, user_id, name, updated_at, created_at, version`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := m.pool.QueryRow(ctx, query, chat.UserID, chat.Name).Scan(
		&chat.ID,
		&chat.UserID,
		&chat.Name,
		&chat.UpdatedAt,
		&chat.CreatedAt,
		&chat.Version,
	)

	return err
}

func (m ChatModel) Get(ctx context.Context, id, user_id int64) (Chat, error) {
	query := `
		SELECT id, user_id, name, updated_at, created_at, version
		FROM chats
		WHERE id = $1 AND user_id = $2`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var chat Chat
	err := m.pool.QueryRow(ctx, query, id, user_id).Scan(
		&chat.ID,
		&chat.UserID,
		&chat.Name,
		&chat.UpdatedAt,
		&chat.CreatedAt,
		&chat.Version,
	)

	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return Chat{}, ErrNoRecord
		default:
			return Chat{}, err
		}
	}

	return chat, nil
}

func (m ChatModel) Update(ctx context.Context, chat *Chat) error {
	query := `
		UPDATE chats
		SET name = $1, updated_at = NOW(), version = version + 1
		WHERE id = $2 AND user_id = $3 AND version = $4
		RETURNING version`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	args := []any{chat.Name, chat.ID, chat.UserID, chat.Version}

	err := m.pool.QueryRow(ctx, query, args...).Scan(
		&chat.Version,
	)

	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}

	return nil
}

func (m ChatModel) Delete(ctx context.Context, id, user_id int64) error {
	query := `
		DELETE FROM chats
		WHERE id = $1 and user_id = $2`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tag, err := m.pool.Exec(ctx, query, id, user_id)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return ErrNoRecord
	}

	return nil
}

func (m ChatModel) Touch(ctx context.Context, id, user_id int64) (Chat, error) {
	query := `
		UPDATE chats
		SET updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, name, updated_at, created_at, version`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var chat Chat

	err := m.pool.QueryRow(ctx, query, id, user_id).Scan(
		&chat.ID, &chat.UserID, &chat.Name,
		&chat.UpdatedAt, &chat.CreatedAt, &chat.Version,
	)

	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return Chat{}, ErrNoRecord
		default:
			return Chat{}, err
		}
	}

	return chat, nil
}
