package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteJSON(t *testing.T) {
	tests := []struct {
		name      string
		data      envelope
		wantCode  int
		checkBody func(t *testing.T, resp *http.Response)
		wantErr   bool
	}{
		{
			name: "status ok with user as envelope data",
			data: envelope{"user": map[string]any{
				"name": "saba",
				"age":  10,
			}},
			wantCode: http.StatusOK,
			checkBody: func(t *testing.T, resp *http.Response) {
				t.Helper()

				var result struct {
					User struct {
						Name string `json:"name"`
						Age  int    `json:"age"`
					} `json:"user"`
				}

				err := json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, "saba", result.User.Name)
				assert.Equal(t, 10, result.User.Age)
			},
			wantErr: false,
		},
		{
			name:     "status not found with error as envelope data",
			data:     envelope{"error": "resource not found"},
			wantCode: http.StatusNotFound,
			checkBody: func(t *testing.T, resp *http.Response) {
				t.Helper()

				var result struct {
					Error string `json:"error"`
				}

				err := json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, "resource not found", result.Error)
			},
			wantErr: false,
		},
		{
			name:      "unmarshalable type error for channel",
			data:      envelope{"channel": make(chan int)},
			wantCode:  0,
			checkBody: nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApplication(t)
			rr := httptest.NewRecorder()

			err := app.writeJSON(rr, tt.wantCode, tt.data)

			if tt.wantErr {
				require.Error(t, err)
				assert.Empty(t, rr.Body.String())
				assert.Empty(t, rr.Header().Get("Content-Type"))
			} else {
				require.NoError(t, err)

				resp := rr.Result()
				defer resp.Body.Close()

				assert.Equal(t, tt.wantCode, resp.StatusCode)
				assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

				if tt.checkBody != nil {
					tt.checkBody(t, resp)
				}
			}
		})
	}
}

func TestReadJSON(t *testing.T) {
	type dst struct {
		Example1 string `json:"example1"`
		Example2 int    `json:"example2"`
	}

	tests := []struct {
		name      string
		inputBody string
		message   string
		wantErr   bool
	}{
		{
			name:      "success",
			inputBody: `{"example1": "bubu", "example2": 10}`,
			message:   "",
			wantErr:   false,
		},
		{
			name:      "empty request body",
			inputBody: "",
			message:   "body must not be empty",
			wantErr:   true,
		},
		{
			name:      "badly formed JSON in request body",
			inputBody: `{"example1: "bubu", "example2": 10}`,
			message:   "body contains badly-formed JSON (at character 14)",
			wantErr:   true,
		},
		{
			name:      "unexpected EOF in request body",
			inputBody: `{"example1": "bubu"`,
			message:   "body contains badly-formed JSON",
			wantErr:   true,
		},
		{
			name:      "incorrect JSON type in request body",
			inputBody: `{"example1": "bubu", "example2": "10"}`,
			message:   `body contains incorrect JSON type for field "example2"`,
			wantErr:   true,
		},
		{
			name:      "large request body",
			inputBody: `{"example1":"` + strings.Repeat("x", 1<<20) + `"}`,
			message:   "body must not be larger than 1048576 bytes",
			wantErr:   true,
		},
		{
			name:      "unknown field in request body",
			inputBody: `{"example1": "bubu", "whos_that": 10}`,
			message:   `body contains unknown key "whos_that"`,
			wantErr:   true,
		},
		{
			name: "2 valid json input in request body",
			inputBody: `{"example1": "bubu", "example2": 10}
			{"example1": "bubu", "example2": 10}`,
			message: "body must only contain a single JSON value",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApplication(t)
			req := httptest.NewRequest(http.MethodGet, "/health", strings.NewReader(tt.inputBody))
			rr := httptest.NewRecorder()

			var result dst
			err := app.readJSON(rr, req, &result)

			if tt.wantErr {
				require.EqualError(t, err, tt.message)
			} else {
				require.NoError(t, err)
				assert.Equal(t, "bubu", result.Example1)
				assert.Equal(t, 10, result.Example2)
			}
		})
	}
}
