package main

import (
	"maps"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetrics(t *testing.T) {
	totalRequestsReceived.Set(0)
	totalResponsesSent.Set(0)
	totalProcessingTimeMicroseconds.Set(0)
	totalResponsesSentByStatus.Init()

	app := newTestApplication(t)

	serve := func(status int) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
		})

		app.metrics(next).ServeHTTP(rr, req)
	}

	serve(http.StatusOK)
	serve(http.StatusOK)
	serve(http.StatusMethodNotAllowed)

	assert.Equal(t, "3", totalRequestsReceived.String())
	assert.Equal(t, "3", totalResponsesSent.String())
	assert.Equal(t, "2", totalResponsesSentByStatus.Get("200").String())
	assert.Equal(t, "1", totalResponsesSentByStatus.Get("405").String())
}

func TestRecoverPanic(t *testing.T) {
	app := newTestApplication(t)

	tests := []struct {
		name        string
		next        http.HandlerFunc
		wantStatus  int
		wantHeaders http.Header
	}{
		{
			name: "no panic",
			next: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusCreated)
			},
			wantStatus:  http.StatusCreated,
			wantHeaders: nil,
		},
		{
			name: "panic",
			next: func(w http.ResponseWriter, r *http.Request) {
				panic("panic attack!!")
			},
			wantStatus:  http.StatusInternalServerError,
			wantHeaders: http.Header{"Connection": []string{"close"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)

			app.recoverPanic(tt.next).ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Result().StatusCode)
			assert.Subset(t, rr.Result().Header, tt.wantHeaders)
		})
	}
}

func TestEnableCORS(t *testing.T) {
	app := newTestApplication(t)
	app.cfg.CORS.TrustedOrigins = []string{"https://example.com"}

	tests := []struct {
		name        string
		method      string
		headers     http.Header
		wantHeaders http.Header
	}{
		{
			name:    "no origin",
			method:  http.MethodGet,
			headers: nil,
			wantHeaders: http.Header{
				"Vary": []string{"Origin", "Access-Control-Request-Method"},
			},
		},
		{
			name:    "untrusted origin",
			method:  http.MethodGet,
			headers: http.Header{"Origin": []string{"https://notexample.com"}},
			wantHeaders: http.Header{
				"Vary": []string{"Origin", "Access-Control-Request-Method"},
			},
		},
		{
			name:    "trusted origin",
			method:  http.MethodGet,
			headers: http.Header{"Origin": []string{"https://example.com"}},
			wantHeaders: http.Header{
				"Vary":                        []string{"Origin", "Access-Control-Request-Method"},
				"Access-Control-Allow-Origin": []string{"https://example.com"},
			},
		},
		{
			name:   "trusted pre-flight",
			method: http.MethodOptions,
			headers: http.Header{
				"Origin":                        []string{"https://example.com"},
				"Access-Control-Request-Method": []string{"blabla"},
			},
			wantHeaders: http.Header{
				"Vary":                         []string{"Origin", "Access-Control-Request-Method"},
				"Access-Control-Allow-Origin":  []string{"https://example.com"},
				"Access-Control-Allow-Methods": []string{"GET, HEAD, POST, PUT, PATCH, DELETE, OPTIONS"},
				"Access-Control-Allow-Headers": []string{"Authorization, Content-Type"},
			},
		},
		{
			name:   "trusted pre-flight without request-method header",
			method: http.MethodOptions,
			headers: http.Header{
				"Origin": []string{"https://example.com"},
			},
			wantHeaders: http.Header{
				"Vary":                        []string{"Origin", "Access-Control-Request-Method"},
				"Access-Control-Allow-Origin": []string{"https://example.com"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, "/", nil)
			maps.Copy(req.Header, tt.headers)

			app.enableCORS(http.HandlerFunc(app.health)).ServeHTTP(rr, req)

			assert.Subset(t, rr.Result().Header, tt.wantHeaders)
		})
	}
}

func TestRateLimit(t *testing.T) {
	app := newTestApplication(t)
	app.cfg.Limiter.RPS = 2
	app.cfg.Limiter.Burst = 4

	tests := []struct {
		name    string
		enabled bool
		exceed  bool
	}{
		{
			name:    "not enabled, not exceeded",
			enabled: false,
			exceed:  false,
		},
		{
			name:    "not enabled, exceeded",
			enabled: false,
			exceed:  true,
		},
		{
			name:    "enabled, not exceeded",
			enabled: true,
			exceed:  false,
		},
		{
			name:    "enabled, exceeded",
			enabled: true,
			exceed:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app.cfg.Limiter.Enabled = tt.enabled

			handler := app.rateLimit(http.HandlerFunc(app.health))

			if tt.exceed {
				for i := range app.cfg.Limiter.Burst + 1 {
					rr := httptest.NewRecorder()
					req := httptest.NewRequest(http.MethodGet, "/health", nil)
					req.RemoteAddr = "192.0.2.1:1234"

					handler.ServeHTTP(rr, req)

					if tt.enabled && i == app.cfg.Limiter.Burst {
						assert.Equal(t, http.StatusTooManyRequests, rr.Result().StatusCode)
					} else {
						assert.Equal(t, http.StatusOK, rr.Result().StatusCode)
					}
				}
			} else {
				rr := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, "/health", nil)
				req.RemoteAddr = "192.0.2.1:1234"

				app.rateLimit(http.HandlerFunc(app.health)).ServeHTTP(rr, req)

				assert.Equal(t, http.StatusOK, rr.Result().StatusCode)
			}
		})
	}
}
