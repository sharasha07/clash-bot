package e2e

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/sharasha07/clash-bot/internal/data"
	"github.com/stretchr/testify/assert"
)

func TestCreateUserHandler(t *testing.T) {
	tests := []struct {
		name      string
		method    string
		url       string
		body      io.Reader
		wantCode  int
		checkBody func(t *testing.T, resp *http.Response)
	}{
		{
			name:     "Success",
			method:   http.MethodPost,
			url:      apiURL + "/v1/users",
			body:     strings.NewReader(`{"username": "giorgi", "password": "giorgi123"}`),
			wantCode: http.StatusCreated,
			checkBody: func(t *testing.T, resp *http.Response) {
				var result struct {
					User data.User `json:"user"`
				}

				err := json.NewDecoder(resp.Body).Decode(&result)
				if err != nil {
					t.Fatal(err)
				}

				assert.Equal(t, "giorgi", result.User.Username)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, tt.url, tt.body)
			if err != nil {
				t.Fatal(err)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			assert.Equal(t, tt.wantCode, resp.StatusCode)

			if tt.checkBody != nil {
				tt.checkBody(t, resp)
			}
		})
	}
}
