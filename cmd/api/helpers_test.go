package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/sharasha07/clash-bot/internal/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadIDParam(t *testing.T) {
	t.Parallel()

	app := newTestApplication(t)

	tests := []struct {
		name    string
		id      string
		wantID  int64
		wantErr bool
	}{
		{"negative integer id", "-4", 0, true},
		{"decimal number id", "2.3", 0, true},
		{"string id", "second", 0, true},
		{"zero id", "0", 0, true},
		{"empty id", "", 0, true},
		{"positive integer id", "5", 5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{}
			params := httprouter.Params{{Key: "id", Value: tt.id}}
			req = req.WithContext(context.WithValue(req.Context(), httprouter.ParamsKey, params))

			gotID, gotErr := app.readIDParam(req)

			assert.Equal(t, tt.wantID, gotID)

			if tt.wantErr {
				assert.EqualError(t, gotErr, "invalid id parameter")
			} else {
				assert.NoError(t, gotErr)
			}
		})
	}
}

func TestReadString(t *testing.T) {
	t.Parallel()

	app := newTestApplication(t)

	tests := []struct {
		name         string
		qs           url.Values
		key          string
		defaultValue string
		want         string
	}{
		{
			name:         "non-existent key",
			qs:           url.Values{},
			key:          "sort",
			defaultValue: "id",
			want:         "id",
		},
		{
			name:         "existent key shadowing default value",
			qs:           url.Values{"sort": []string{"name"}},
			key:          "sort",
			defaultValue: "id",
			want:         "name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := app.readString(tt.qs, tt.key, tt.defaultValue)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestReadInt(t *testing.T) {
	t.Parallel()

	app := newTestApplication(t)

	tests := []struct {
		name         string
		qs           url.Values
		key          string
		defaultValue int
		wantValue    int
		wantMessage  bool
	}{
		{
			name:         "non-existent key",
			qs:           url.Values{},
			key:          "page",
			defaultValue: 1,
			wantValue:    1,
			wantMessage:  false,
		},
		{
			name:         "existent decimal key",
			qs:           url.Values{"page": []string{"2.3"}},
			key:          "page",
			defaultValue: 1,
			wantValue:    1,
			wantMessage:  true,
		},
		{
			name:         "existent string key",
			qs:           url.Values{"page": []string{"one"}},
			key:          "page",
			defaultValue: 1,
			wantValue:    1,
			wantMessage:  true,
		},
		{
			name:         "existent key shadowing default value",
			qs:           url.Values{"page": []string{"4"}},
			key:          "page",
			defaultValue: 1,
			wantValue:    4,
			wantMessage:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := validator.New()
			got := app.readInt(tt.qs, tt.key, tt.defaultValue, v)

			assert.Equal(t, tt.wantValue, got)

			if tt.wantMessage {
				assert.Equal(t, "must be an integer value", v.Errors[tt.key])
			}
		})
	}
}

func TestWriteJSON(t *testing.T) {
	t.Parallel()

	app := newTestApplication(t)

	tests := []struct {
		name      string
		status    int
		env       envelope
		wantErr   bool
		checkBody func(t *testing.T, body io.Reader)
	}{
		{
			name:    "marshal error",
			status:  http.StatusMethodNotAllowed,
			env:     envelope{"channel": make(chan int)},
			wantErr: true,
			checkBody: func(t *testing.T, body io.Reader) {
				t.Helper()

				data, err := io.ReadAll(body)
				require.NoError(t, err)

				assert.Empty(t, data)
			},
		},
		{
			name:    "send error message",
			status:  http.StatusMethodNotAllowed,
			env:     envelope{"error": "method not allowed"},
			wantErr: false,
			checkBody: func(t *testing.T, body io.Reader) {
				t.Helper()

				var result struct {
					Error string `json:"error"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, "method not allowed", result.Error)
			},
		},
		{
			name:   "send user",
			status: http.StatusCreated,
			env: envelope{"user": struct {
				ID       int    `json:"id"`
				Username string `json:"username"`
			}{
				ID:       10,
				Username: "saba",
			}},
			wantErr: false,
			checkBody: func(t *testing.T, body io.Reader) {
				t.Helper()

				var result struct {
					User struct {
						ID       int    `json:"id"`
						Username string `json:"username"`
					} `json:"user"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, 10, result.User.ID)
				assert.Equal(t, "saba", result.User.Username)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()

			gotErr := app.writeJSON(rr, tt.status, tt.env)

			resp := rr.Result()
			defer resp.Body.Close()

			if tt.wantErr {
				require.Error(t, gotErr)

				assert.Equal(t, http.StatusOK, resp.StatusCode)
			} else {
				require.NoError(t, gotErr)

				assert.Equal(t, tt.status, resp.StatusCode)
				assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
			}

			if tt.checkBody != nil {
				tt.checkBody(t, resp.Body)
			}
		})
	}
}

func TestReadJSON(t *testing.T) {
	t.Parallel()

	app := newTestApplication(t)

	type dst struct {
		Example1 int    `json:"example1"`
		Example2 string `json:"example2"`
	}

	tests := []struct {
		name        string
		input       string
		wantMessage string
	}{
		{
			name:        "empty body error",
			input:       "",
			wantMessage: "body must not be empty",
		},
		{
			name:        "badly-formed JSON error",
			input:       `{"example1: 5, "example2": "saba"}`,
			wantMessage: "body contains badly-formed JSON (at character 17)",
		},
		{
			name:        "badly-formed JSON (unexpected EOF) error",
			input:       `{"example1": 5, "example2`,
			wantMessage: "body contains badly-formed JSON",
		},
		{
			name:        "unmarshal type error",
			input:       `{"example1": "5", "example2": "saba"}`,
			wantMessage: `body contains incorrect JSON type for field "example1"`,
		},
		{
			name:        "large body error",
			input:       `{"example2": "` + strings.Repeat("s", 1<<20),
			wantMessage: "body must not be larger than 1048576 bytes",
		},
		{
			name:        "unknown field error",
			input:       `{"exam1": 5, "example2": "saba"}`,
			wantMessage: `body contains unknown key "exam1"`,
		},
		{
			name:        "doble json error",
			input:       `{"example1": 5, "example2": "saba"} {"example1": 5, "example2": "saba"}`,
			wantMessage: `body must only contain a single JSON value`,
		},
		{
			name:        "no error",
			input:       `{"example1": 5, "example2": "saba"}`,
			wantMessage: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", strings.NewReader(tt.input))

			var result dst

			gotErr := app.readJSON(rr, req, &result)

			if tt.wantMessage == "" {
				require.NoError(t, gotErr)

				assert.Equal(t, result.Example1, 5)
				assert.Equal(t, result.Example2, "saba")
			} else {
				require.Error(t, gotErr)

				assert.EqualError(t, gotErr, tt.wantMessage)
			}
		})
	}
}
