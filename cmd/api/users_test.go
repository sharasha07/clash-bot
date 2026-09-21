package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sharasha07/clash-bot/internal/assert"
	"github.com/sharasha07/clash-bot/internal/data"
)

func TestCreateUserHandler(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantCode  int
		checkBody func(t *testing.T, resp *http.Response)
	}{
		{
			name:     "User successfully created",
			input:    `{"username": "luka", "password": "lukaluka123"}`,
			wantCode: http.StatusCreated,
			checkBody: func(t *testing.T, resp *http.Response) {
				t.Helper()

				var result struct {
					User data.User `json:"user"`
				}

				err := json.NewDecoder(resp.Body).Decode(&result)
				if err != nil {
					t.Fatal(err)
				}

				assert.Equal(t, "luka", result.User.Username)
			},
		},
		{
			name:     "Long username, valid password",
			input:    `{"username": "lukalukaluka", "password": "lukaluka123"}`,
			wantCode: http.StatusUnprocessableEntity,
			checkBody: func(t *testing.T, resp *http.Response) {
				t.Helper()

				var result struct {
					Error map[string]string `json:"error"`
				}

				err := json.NewDecoder(resp.Body).Decode(&result)
				if err != nil {
					t.Fatal(err)
				}

				assert.Equal(t, "must not be more than 10 characters", result.Error["username"])
				assert.Equal(t, 1, len(result.Error))
			},
		},
		{
			name:     "empty username, empty password",
			input:    `{"username": "", "password": ""}`,
			wantCode: http.StatusUnprocessableEntity,
			checkBody: func(t *testing.T, resp *http.Response) {
				t.Helper()

				var result struct {
					Error map[string]string `json:"error"`
				}

				err := json.NewDecoder(resp.Body).Decode(&result)
				if err != nil {
					t.Fatal(err)
				}

				assert.Equal(t, "must be provided", result.Error["username"])
				assert.Equal(t, "must be provided", result.Error["password"])
				assert.Equal(t, 2, len(result.Error))
			},
		},
		{
			name:     "Valid username, short password",
			input:    `{"username": "luka", "password": "luka123"}`,
			wantCode: http.StatusUnprocessableEntity,
			checkBody: func(t *testing.T, resp *http.Response) {
				t.Helper()

				var result struct {
					Error map[string]string `json:"error"`
				}

				err := json.NewDecoder(resp.Body).Decode(&result)
				if err != nil {
					t.Fatal(err)
				}

				assert.Equal(t, "must be at least 8 characters", result.Error["password"])
				assert.Equal(t, 1, len(result.Error))
			},
		},
		{
			name:     "Invalid username, invalid password",
			input:    fmt.Sprintf(`{"username": "lukalukaluka", "password": "%s"}`, strings.Repeat("luka", 8)),
			wantCode: http.StatusUnprocessableEntity,
			checkBody: func(t *testing.T, resp *http.Response) {
				t.Helper()

				var result struct {
					Error map[string]string `json:"error"`
				}

				err := json.NewDecoder(resp.Body).Decode(&result)
				if err != nil {
					t.Fatal(err)
				}

				assert.Equal(t, "must not be more than 10 characters", result.Error["username"])
				assert.Equal(t, "must not be more than 30 characters", result.Error["password"])
				assert.Equal(t, 2, len(result.Error))
			},
		},
		{
			name:     "Username unique violation",
			input:    `{"username": "shaba", "password": "luka1234"}`,
			wantCode: http.StatusUnprocessableEntity,
			checkBody: func(t *testing.T, resp *http.Response) {
				t.Helper()

				var result struct {
					Error map[string]string `json:"error"`
				}

				err := json.NewDecoder(resp.Body).Decode(&result)
				if err != nil {
					t.Fatal(err)
				}

				assert.Equal(t, "must be unique", result.Error["username"])
				assert.Equal(t, 1, len(result.Error))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/v1/users", strings.NewReader(tt.input))
			rr := httptest.NewRecorder()

			app := newTestApplication()
			app.createUserHandler(rr, req)

			resp := rr.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.wantCode, resp.StatusCode)
			assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
			if tt.checkBody != nil {
				tt.checkBody(t, resp)
			}
		})
	}
}
