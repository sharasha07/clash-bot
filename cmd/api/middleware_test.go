package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sharasha07/clash-bot/internal/assert"
)

func TestMetrics(t *testing.T) {
	totalRequestsReceived.Set(0)
	totalResponsesSent.Set(0)
	totalProcessingTimeMicroseconds.Set(0)
	totalResponsesSentByStatus.Init()

	stub1 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	stub2 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	app := newTestApplication()

	serve := func(stub http.HandlerFunc) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rr := httptest.NewRecorder()

		app.metrics(stub).ServeHTTP(rr, req)
	}

	serve(stub1)
	serve(stub1)
	serve(stub2)

	assert.Equal(t, "3", totalRequestsReceived.String())
	assert.Equal(t, "3", totalResponsesSent.String())
	assert.Equal(t, "2", totalResponsesSentByStatus.Get("200").String())
	assert.Equal(t, "1", totalResponsesSentByStatus.Get("404").String())
	assert.Equal(t, true, totalProcessingTimeMicroseconds.Value() > 0)
}

func TestRecoverPanic(t *testing.T) {
	tests := []struct {
		name       string
		stub       http.HandlerFunc
		wantCode   int
		wantHeader http.Header
	}{
		{
			name: "No panic",
			stub: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}),
			wantCode:   http.StatusOK,
			wantHeader: nil,
		},
		{
			name: "Panic",
			stub: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				panic("Panic attack!")
			}),
			wantCode:   http.StatusInternalServerError,
			wantHeader: http.Header{"Connection": []string{"close"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			rr := httptest.NewRecorder()

			app := newTestApplication()
			app.recoverPanic(tt.stub).ServeHTTP(rr, req)

			resp := rr.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.wantCode, resp.StatusCode)

			assert.EqualHeaders(t, tt.wantHeader, resp.Header)

		})
	}
}

func TestEnableCORS(t *testing.T) {
	tests := []struct {
		name       string
		isTrusted  bool
		method     string
		header     http.Header
		stub       http.HandlerFunc
		wantCode   int
		wantHeader http.Header
	}{
		{
			name:      "No Origin",
			isTrusted: false,
			method:    http.MethodGet,
			header:    nil,
			stub: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}),
			wantCode: http.StatusOK,
			wantHeader: http.Header{
				"Vary": []string{"Origin", "Access-Control-Request-Method"},
			},
		},
		{
			name:      "Trusted Origin",
			isTrusted: true,
			method:    http.MethodPost,
			header: http.Header{
				"Origin": []string{"http://localhost:6767"},
			},
			stub: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusCreated)
			}),
			wantCode: http.StatusCreated,
			wantHeader: http.Header{
				"Vary":                        []string{"Origin", "Access-Control-Request-Method"},
				"Access-Control-Allow-Origin": []string{"http://localhost:6767"},
			},
		},
		{
			name:      "Untrusted Origin",
			isTrusted: false,
			method:    http.MethodDelete,
			header: http.Header{
				"Origin": []string{"http://localhost:6767"},
			},
			stub: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusCreated)
			}),
			wantCode: http.StatusCreated,
			wantHeader: http.Header{
				"Vary": []string{"Origin", "Access-Control-Request-Method"},
			},
		},
		{
			name:      "Trusted Origin + Options method + Access-Control-Request-Method",
			isTrusted: true,
			method:    http.MethodOptions,
			header: http.Header{
				"Origin":                        []string{"http://localhost:6767"},
				"Access-Control-Request-Method": []string{http.MethodPatch},
			},
			stub: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusCreated)
			}),
			wantCode: http.StatusOK,
			wantHeader: http.Header{
				"Vary":                         []string{"Origin", "Access-Control-Request-Method"},
				"Access-Control-Allow-Origin":  []string{"http://localhost:6767"},
				"Access-Control-Allow-Methods": []string{"OPTIONS, PUT, PATCH, DELETE"},
				"Access-Control-Allow-Headers": []string{"Authorization, Content-Type"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/health", nil)
			req.Header = tt.header
			rr := httptest.NewRecorder()

			app := newTestApplication()

			if tt.isTrusted {
				app.cfg.CORS.TrustedOrigins = append(app.cfg.CORS.TrustedOrigins, tt.header.Get("Origin"))
			}

			app.enableCORS(tt.stub).ServeHTTP(rr, req)

			resp := rr.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.wantCode, resp.StatusCode)
			assert.EqualHeaders(t, tt.wantHeader, resp.Header)
			if tt.wantHeader.Get("Access-Control-Allow-Origin") == "" {
				assert.Equal(t, "", resp.Header.Get("Access-Control-Allow-Origin"))
			}
		})
	}
}
