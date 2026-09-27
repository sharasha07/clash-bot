package data

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:generate mockgen -source=chats.go -destination=../mocks/chat_repo.go -package=mocks
type ChatRepository interface {
	Insert(ctx context.Context, chat *Chat) error
	Get(ctx context.Context, id, user_id int64) (Chat, error)
}

type Chat struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Name      string    `json:"name"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedAt time.Time `json:"created_at"`
	Version   int       `json:"-"`
}

type ChatModel struct {
	pool *pgxpool.Pool
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
