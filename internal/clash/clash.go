package clash

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	ErrNotFound           = errors.New("resource not found")
	ErrServiceUnavailable = errors.New("service unavailable")
	ErrResponseTooLarge   = errors.New("response too large")
)

//go:generate mockgen -source=clash.go -destination=../mocks/clash_client_mock.go -package=mocks
type ClashClient interface {
	GetPlayer(ctx context.Context, tag string) (string, error)
	GetPlayerBattleLog(ctx context.Context, tag string, limit int) (string, error)
	GetPlayersUpcomingChests(ctx context.Context, tag string, limit int) (string, error)
}

type apiClient struct {
	baseURL        string
	token          string
	http           *http.Client
	maxResultBytes int
}

func NewAPIClient(baseURL, token string, timeout time.Duration, maxResultBytes int) *apiClient {
	return &apiClient{
		baseURL:        strings.TrimRight(baseURL, "/"),
		token:          token,
		http:           &http.Client{Timeout: timeout},
		maxResultBytes: maxResultBytes,
	}
}

func (c *apiClient) GetPlayer(ctx context.Context, tag string) (string, error) {
	tag = strings.ToUpper(strings.TrimSpace(tag))

	return c.get(ctx, "/v1/players/"+url.PathEscape(tag), nil)
}

func (c *apiClient) GetPlayerBattleLog(ctx context.Context, tag string, limit int) (string, error) {
	tag = strings.ToUpper(strings.TrimSpace(tag))

	if limit > 50 {
		limit = 50
	}

	query := url.Values{}
	query.Set("limit", strconv.Itoa(limit))

	return c.get(ctx, "/v1/players/"+url.PathEscape(tag)+"/battlelog", query)
}

func (c *apiClient) GetPlayersUpcomingChests(ctx context.Context, tag string, limit int) (string, error) {
	tag = strings.ToUpper(strings.TrimSpace(tag))

	if limit > 50 {
		limit = 50
	}

	query := url.Values{}
	query.Set("limit", strconv.Itoa(limit))

	return c.get(ctx, "/v1/players/"+url.PathEscape(tag)+"/upcomingchests", query)
}

func (c *apiClient) get(ctx context.Context, path string, query url.Values) (string, error) {
	endpoint, err := url.Parse(c.baseURL + path)
	if err != nil {
		return "", err
	}

	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusNotFound:
		return "", ErrNotFound
	case resp.StatusCode != http.StatusOK:
		return "", ErrServiceUnavailable
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, int64(c.maxResultBytes)+1))
	if err != nil {
		return "", err
	}

	if len(body) > c.maxResultBytes {
		return "", ErrResponseTooLarge
	}

	return string(body), nil
}
