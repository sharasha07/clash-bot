package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/sharasha07/clash-bot/internal/data"
	"github.com/sharasha07/clash-bot/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestCreateUserHandler(t *testing.T) {
	app := newTestApplication(t)
	app.models.Users.(*mocks.MockUserRepository).EXPECT().Insert(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, user *data.User) error {
			if user.Username == "shaba" {
				return data.ErrDuplicateUsersUsername
			}

			user.ID = 5
			return nil
		}).Times(2)

	tests := []struct {
		name      string
		input     string
		wantCode  int
		checkBody func(t *testing.T, body io.Reader)
	}{
		{
			name:     "empty username, small password",
			input:    `{"username": "", "password": "saba123"}`,
			wantCode: http.StatusUnprocessableEntity,
			checkBody: func(t *testing.T, body io.Reader) {
				var result struct {
					Error map[string]string `json:"error"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, 2, len(result.Error))
				assert.Equal(t, "must not be empty", result.Error["username"])
				assert.Equal(t, "must be more than 8 characters", result.Error["password"])
			},
		},
		{
			name:     "big username, empty password",
			input:    `{"username": "shabashabashaba1", "password": ""}`,
			wantCode: http.StatusUnprocessableEntity,
			checkBody: func(t *testing.T, body io.Reader) {
				var result struct {
					Error map[string]string `json:"error"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, 2, len(result.Error))
				assert.Equal(t, "must be a maximum of 15 characters", result.Error["username"])
				assert.Equal(t, "must not be empty", result.Error["password"])
			},
		},
		{
			name:     "whitespace username, big password",
			input:    fmt.Sprintf(`{"username": "    ", "password": "%s"}`, strings.Repeat("a", 41)),
			wantCode: http.StatusUnprocessableEntity,
			checkBody: func(t *testing.T, body io.Reader) {
				var result struct {
					Error map[string]string `json:"error"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, 2, len(result.Error))
				assert.Equal(t, "must not be empty", result.Error["username"])
				assert.Equal(t, "must be a maximum of 40 characters", result.Error["password"])
			},
		},
		{
			name:     "duplicate username",
			input:    `{"username": "shaba", "password": "shaba1234"}`,
			wantCode: http.StatusUnprocessableEntity,
			checkBody: func(t *testing.T, body io.Reader) {
				var result struct {
					Error map[string]string `json:"error"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, 1, len(result.Error))
				assert.Equal(t, "must be unique", result.Error["username"])
			},
		},
		{
			name:     "successful input",
			input:    `{"username": "luka", "password": "lukaluka123"}`,
			wantCode: http.StatusCreated,
			checkBody: func(t *testing.T, body io.Reader) {
				var result struct {
					User data.User `json:"user"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, int64(5), result.User.ID)
				assert.Equal(t, "luka", result.User.Username)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/v1/users", strings.NewReader(tt.input))

			app.createUserHandler(rr, req)

			resp := rr.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.wantCode, resp.StatusCode)
			assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

			if tt.checkBody != nil {
				tt.checkBody(t, resp.Body)
			}
		})
	}
}

func TestUploadProfilePictureHandler(t *testing.T) {
	app := newTestApplication(t)
	app.cfg.R2.PublicURL = "public"
	app.cfg.R2.Bucket = "test_bucket"

	app.s3Client.(*mocks.MockS3ObjectStorage).EXPECT().
		PutObject(gomock.Any(), gomock.Any()).Return(nil, nil).Times(2)

	app.models.Users.(*mocks.MockUserRepository).EXPECT().
		Update(gomock.Any(), gomock.Any()).Return(nil).Times(2)

	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	var pngBuf, jpegBuf, gifBuf bytes.Buffer

	jpeg.Encode(&jpegBuf, img, nil)
	png.Encode(&pngBuf, img)
	gif.Encode(&gifBuf, img, nil)

	tests := []struct {
		name      string
		image     bytes.Buffer
		userID    int
		wantCode  int
		checkBody func(t *testing.T, body io.Reader)
	}{
		{
			name:     "anonymous user",
			image:    pngBuf,
			userID:   0,
			wantCode: http.StatusUnauthorized,
			checkBody: func(t *testing.T, body io.Reader) {
				var result struct {
					Error string `json:"error"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, "authentication required to access this endpoint", result.Error)
			},
		},
		{
			name:     "unauthorized user",
			image:    pngBuf,
			userID:   2,
			wantCode: http.StatusForbidden,
			checkBody: func(t *testing.T, body io.Reader) {
				var result struct {
					Error string `json:"error"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, "user is not permitted to access this resource", result.Error)
			},
		},
		{
			name:     "large body size",
			image:    *bytes.NewBufferString(strings.Repeat("a", 5<<20+1)),
			userID:   1,
			wantCode: http.StatusBadRequest,
			checkBody: func(t *testing.T, body io.Reader) {
				var result struct {
					Error string `json:"error"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, "malformed upload", result.Error)
			},
		},
		{
			name:     "invalid image",
			image:    *bytes.NewBufferString("image"),
			userID:   1,
			wantCode: http.StatusBadRequest,
			checkBody: func(t *testing.T, body io.Reader) {
				var result struct {
					Error string `json:"error"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, "invalid image", result.Error)
			},
		},
		{
			name:     "unsupported image type",
			image:    gifBuf,
			userID:   1,
			wantCode: http.StatusBadRequest,
			checkBody: func(t *testing.T, body io.Reader) {
				var result struct {
					Error string `json:"error"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, "unsupported image type", result.Error)
			},
		},
		{
			name:     "success with .png",
			image:    pngBuf,
			userID:   1,
			wantCode: http.StatusOK,
			checkBody: func(t *testing.T, body io.Reader) {
				var result struct {
					User data.User `json:"user"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				endpoint, err := url.JoinPath(app.cfg.R2.PublicURL, app.cfg.R2.Bucket, "users/1/profile_picture")
				require.NoError(t, err)

				assert.Equal(t, int64(1), result.User.ID)
				assert.Equal(t, endpoint, *result.User.ProfilePicture)
			},
		},
		{
			name:     "success with .jpeg",
			image:    jpegBuf,
			userID:   1,
			wantCode: http.StatusOK,
			checkBody: func(t *testing.T, body io.Reader) {
				var result struct {
					User data.User `json:"user"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				endpoint, err := url.JoinPath(app.cfg.R2.PublicURL, app.cfg.R2.Bucket, "users/1/profile_picture")
				require.NoError(t, err)

				assert.Equal(t, int64(1), result.User.ID)
				assert.Equal(t, endpoint, *result.User.ProfilePicture)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()

			var b bytes.Buffer
			writer := multipart.NewWriter(&b)

			part, err := writer.CreateFormFile("avatar", "profile-picture")
			require.NoError(t, err)
			io.Copy(part, &tt.image)

			writer.Close()

			req := httptest.NewRequest(http.MethodPost, "/v1/users/1/profile-picture", &b)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			if tt.userID != 0 {
				req = contextSetUser(req, &data.User{ID: int64(tt.userID)})
			} else {
				req = contextSetUser(req, data.AnonymousUser)
			}

			params := httprouter.Params{{Key: "id", Value: "1"}}
			req = req.WithContext(context.WithValue(req.Context(), httprouter.ParamsKey, params))

			app.uploadProfilePictureHandler(rr, req)

			resp := rr.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.wantCode, resp.StatusCode)
			assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

			if tt.checkBody != nil {
				tt.checkBody(t, resp.Body)
			}
		})
	}
}
