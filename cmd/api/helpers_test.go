package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sharasha07/clash-bot/internal/assert"
)

func TestWriteJSON(t *testing.T) {
	tests := []struct {
		name       string
		input      envelope
		wantCode   int
		wantHeader http.Header
		checkBody  func(t *testing.T, resp *http.Response)
		wantErr    bool
	}{
		{
			name: "StatusOK with the user in the body",
			input: envelope{"user": map[string]any{
				"name": "saba",
				"age":  10,
			}},
			wantCode: http.StatusOK,
			wantHeader: http.Header{
				"Content-Type": []string{"application/json"},
			},
			checkBody: func(t *testing.T, resp *http.Response) {
				t.Helper()

				var result struct {
					User struct {
						Name string `json:"name"`
						Age  int    `json:"age"`
					}
				}

				err := json.NewDecoder(resp.Body).Decode(&result)
				if err != nil {
					t.Fatal(err)
				}

				assert.Equal(t, "saba", result.User.Name)
				assert.Equal(t, 10, result.User.Age)
			},
		},
		{
			name:     "StatusNotFound with the error message in the body",
			input:    envelope{"error": "resource not found"},
			wantCode: http.StatusNotFound,
			wantHeader: http.Header{
				"Content-Type": []string{"application/json"},
			},
			checkBody: func(t *testing.T, resp *http.Response) {
				t.Helper()

				var result struct {
					Error string `json:"error"`
				}

				err := json.NewDecoder(resp.Body).Decode(&result)
				if err != nil {
					t.Fatal(err)
				}

				assert.Equal(t, "resource not found", result.Error)
			},
		},
		{
			name:    "Unmarshalable type input",
			input:   envelope{"channel": make(chan int)},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApplication()
			rr := httptest.NewRecorder()

			err := app.writeJSON(rr, tt.wantCode, tt.input)
			if err != nil {
				if !tt.wantErr {
					t.Error("expected writeJSON to not return error")
				}
				if rr.Body.Len() != 0 {
					t.Errorf("expected empty body, got: %v", rr.Body.String())
				}
			} else {
				resp := rr.Result()
				defer resp.Body.Close()
				assert.Equal(t, tt.wantCode, resp.StatusCode)

				assert.EqualHeaders(t, tt.wantHeader, resp.Header)

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
	}{
		{
			name:      "empty body",
			inputBody: "",
			message:   "body must not be empty",
		},
		{
			name:      "badly-formed JSON",
			inputBody: `{"example1: "bubu", "example2": 10}`,
			message:   "body contains badly-formed JSON (at character 14)",
		},
		{
			name:      "Incorrect JSON type",
			inputBody: `{"example1": "bubu", "example2": "10"}`,
			message:   `body contains incorrect JSON type for field "example2"`,
		},
		{
			name:      "Large body",
			inputBody: `{"example1":"` + strings.Repeat("x", 1<<20) + `"}`,
			message:   "body must not be larger than 1048576 bytes",
		},
		{
			name:      "Unknown field",
			inputBody: `{"example1": "bubu", "whos_that": "10"}`,
			message:   `body contains unknown key "whos_that"`,
		},
		{
			name: "Doble JSON",
			inputBody: `{"example1": "bubu", "example2": 10}
			{"example1": "bubu", "example2": 10}`,
			message: "body must only contain a single JSON value",
		},
		{
			name:      "Success",
			inputBody: `{"example1": "bubu", "example2": 10}`,
			message:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApplication()
			req := httptest.NewRequest(http.MethodGet, "/health", strings.NewReader(tt.inputBody))
			rr := httptest.NewRecorder()

			var result dst
			err := app.readJSON(rr, req, &result)
			if err != nil {
				assert.Equal(t, tt.message, err.Error())
			} else {
				assert.Equal(t, "bubu", result.Example1)
				assert.Equal(t, 10, result.Example2)
			}
		})
	}
}
