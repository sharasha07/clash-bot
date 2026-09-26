package data

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pascaldekloe/jwt"
)

type Token struct {
	TokenHash []byte
	UserID    int
	ExpiresAt time.Time
	CreatedAt time.Time
}

func NewAccessToken(userID int, jwtSecret string, ttl time.Duration) (string, error) {
	now := time.Now()

	claims := jwt.Claims{
		Subject:   strconv.FormatInt(int64(userID), 10),
		Issued:    jwt.NewNumericTime(now),
		NotBefore: jwt.NewNumericTime(now),
		Expires:   jwt.NewNumericTime(now.Add(ttl)),
		Issuer:    "github.com/sharasha07/clash-bot",
		Audiences: []string{"github.com/sharasha07/clashbot"},
	}

	jwtBytes, err := claims.HMACSign(jwt.HS256, []byte(jwtSecret))
	if err != nil {
		return "", err
	}

	return string(jwtBytes), nil
}

func NewRefreshToken() (string, error) {
	b := make([]byte, 32)

	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}

func tokenHash(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

type TokenModel struct {
	pool *pgxpool.Pool
}

func (m TokenModel) Insert(ctx context.Context, token string, userID int, ttl time.Duration) error {
	query := `
		INSERT INTO refresh_tokens(token_hash, user_id, expires_at)
		VALUES ($1, $2, $3)
		`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	expiry := time.Now().Add(ttl)

	_, err := m.pool.Exec(ctx, query, tokenHash(token), userID, expiry)
	return err
}

func (m TokenModel) GetUserID(ctx context.Context, token string) (int, error) {
	query := `
		SELECT user_id FROM refresh_tokens
		WHERE token_hash = $1 AND expires_at > NOW()`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var userID int
	err := m.pool.QueryRow(ctx, query, tokenHash(token)).Scan(&userID)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return 0, ErrNoRecord
		default:
			return 0, err
		}
	}

	return userID, nil
}

func (m TokenModel) Delete(ctx context.Context, token string) error {
	query := `DELETE FROM refresh_tokens WHERE token_hash = $1`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := m.pool.Exec(ctx, query, tokenHash(token))
	return err
}
