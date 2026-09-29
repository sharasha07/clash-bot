package clash

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	ErrInvalidTag   = errors.New("invalid tag")
	ErrNotFound     = errors.New("resource not found")
	ErrUnauthorized = errors.New("unauthorized")
	ErrRateLimited  = errors.New("rate limited")
	ErrUpstream     = errors.New("upstream error")
)

//go:generate mockgen -source=clash.go -destination=../mocks/clash_api.go -package=mocks
type ClashAPI interface {
	GetPlayer(ctx context.Context, tag string) (json.RawMessage, error)
	GetPlayerBattleLog(ctx context.Context, tag string, limit int) (json.RawMessage, error)
	GetPlayersUpcomingChests(ctx context.Context, tag string, limit int) (json.RawMessage, error)
}

type Client struct {
	baseURL        string
	token          string
	http           *http.Client
	maxResultBytes int
}

func NewClient(baseURL, token string, timeout time.Duration, maxResultBytes int) *Client {
	return &Client{
		baseURL:        strings.TrimRight(baseURL, "/"),
		token:          token,
		http:           &http.Client{Timeout: timeout},
		maxResultBytes: maxResultBytes,
	}
}

func (c *Client) GetPlayer(ctx context.Context, tag string) (json.RawMessage, error) {
	tag = strings.ToUpper(strings.TrimSpace(tag))

	return c.get(ctx, "/v1/players/"+url.PathEscape(tag), nil)
}

func (c *Client) GetPlayerBattleLog(ctx context.Context, tag string, limit int) (json.RawMessage, error) {
	tag = strings.ToUpper(strings.TrimSpace(tag))

	if limit > 50 {
		limit = 50
	}

	query := url.Values{}
	query.Set("limit", strconv.Itoa(limit))

	return c.get(ctx, "/v1/players/"+url.PathEscape(tag)+"/battlelog", query)
}

func (c *Client) GetPlayersUpcomingChests(ctx context.Context, tag string, limit int) (json.RawMessage, error) {
	tag = strings.ToUpper(strings.TrimSpace(tag))

	if limit > 50 {
		limit = 50
	}

	query := url.Values{}
	query.Set("limit", strconv.Itoa(limit))

	return c.get(ctx, "/v1/players/"+url.PathEscape(tag)+"/upcomingchests", query)
}

func (c *Client) get(ctx context.Context, path string, query url.Values) (json.RawMessage, error) {
	endpoint := c.baseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusNotFound:
		return nil, ErrNotFound
	case resp.StatusCode == http.StatusUnauthorized, resp.StatusCode == http.StatusForbidden:
		return nil, ErrUnauthorized
	case resp.StatusCode == http.StatusTooManyRequests:
		return nil, ErrRateLimited
	case resp.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("%w: unexpected status %d", ErrUpstream, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, int64(c.maxResultBytes)+1))
	if err != nil {
		return nil, err
	}

	if len(body) > c.maxResultBytes {
		return json.Marshal(map[string]any{
			"truncated": true,
			"reason": fmt.Sprintf(
				"response exceeded %d bytes, try a narrower request", c.maxResultBytes),
		})
	}

	return json.RawMessage(body), nil
}
