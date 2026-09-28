package data

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sharasha07/clash-bot/internal/validator"
)

const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

//go:generate mockgen -source=messages.go -destination=../mocks/message_repo.go -package=mocks
type MessageRepository interface {
	GetAll(ctx context.Context, chat_id, user_id int64, filters Filters) ([]Message, error)
	Insert(ctx context.Context, message *Message) error
}

type Message struct {
	ID        int64     `json:"id"`
	ChatID    int64     `json:"chat_id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

func (m *Message) Validate(v *validator.Validator) {
	v.Check(m.Role == RoleUser || m.Role == RoleAssistant, "role", "must be either 'user' or 'assistant'")
	v.Check(strings.TrimSpace(m.Content) != "", "content", "must not be empty")
	v.Check(utf8.RuneCountInString(m.Content) <= 4000, "content", "must be a maximum of 4000 characters")
}

type MessageModel struct {
	pool *pgxpool.Pool
}

func (m MessageModel) GetAll(ctx context.Context, chat_id, user_id int64, filters Filters) ([]Message, error) {
	query := fmt.Sprintf(`
		SELECT m.id, m.chat_id, m.role, m.content, m.created_at
		FROM messages m
		JOIN chats c ON c.id = m.chat_id
		WHERE m.chat_id = $1 AND c.user_id = $2
		ORDER BY m.%s %s
		LIMIT $3 OFFSET $4`, filters.SortColumn(), filters.SortOrder())

	args := []any{chat_id, user_id, filters.PageSize, filters.Offset()}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := m.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := []Message{}
	for rows.Next() {
		var message Message

		err := rows.Scan(
			&message.ID,
			&message.ChatID,
			&message.Role,
			&message.Content,
			&message.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		messages = append(messages, message)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

func (m MessageModel) Insert(ctx context.Context, message *Message) error {
	query := `
		INSERT INTO messages (chat_id, role, content)
		VALUES ($1, $2, $3)
		RETURNING id, chat_id, role, content, created_at`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := m.pool.QueryRow(ctx, query, message.ChatID, message.Role, message.Content).Scan(
		&message.ID,
		&message.ChatID,
		&message.Role,
		&message.Content,
		&message.CreatedAt,
	)

	return err
}
