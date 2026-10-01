package main

import (
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
