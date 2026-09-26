package main

import (
	"errors"
	"expvar"
	"fmt"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/felixge/httpsnoop"
	"github.com/pascaldekloe/jwt"
	"github.com/sharasha07/clash-bot/internal/data"
	"golang.org/x/time/rate"
)

var (
	totalRequestsReceived           = expvar.NewInt("total_requests_received")
	totalResponsesSent              = expvar.NewInt("total_responses_sent")
	totalProcessingTimeMicroseconds = expvar.NewInt("total_processing_time_microseconds")
	totalResponsesSentByStatus      = expvar.NewMap("total_responses_sent_by_status")
)

func (app *application) metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		totalRequestsReceived.Add(1)

		metrics := httpsnoop.CaptureMetrics(next, w, r)

		totalResponsesSent.Add(1)
		totalProcessingTimeMicroseconds.Add(metrics.Duration.Microseconds())
		totalResponsesSentByStatus.Add(strconv.Itoa(metrics.Code), 1)
	})
}

func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				app.serverErrorResponse(w, r, fmt.Errorf("%v", err))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func (app *application) enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Origin")
		w.Header().Add("Vary", "Access-Control-Request-Method")

		origin := r.Header.Get("Origin")

		if origin != "" {
			if slices.Contains(app.cfg.CORS.TrustedOrigins, origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)

				if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
					w.Header().Set("Access-Control-Allow-Methods", "OPTIONS, PUT, PATCH, DELETE")
					w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")

					w.WriteHeader(http.StatusOK)
					return
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

func (app *application) rateLimit(next http.Handler) http.Handler {
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)

	go func() {
		for {
			time.Sleep(time.Minute)

			mu.Lock()

			for ip, client := range clients {
				if time.Since(client.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}

			mu.Unlock()
		}
	}()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if app.cfg.Limiter.Enabled {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				app.serverErrorResponse(w, r, err)
				return
			}

			mu.Lock()

			if _, found := clients[ip]; !found {
				clients[ip] = &client{limiter: rate.NewLimiter(rate.Limit(app.cfg.Limiter.RPS), app.cfg.Limiter.Burst)}
			}

			clients[ip].lastSeen = time.Now()

			if !clients[ip].limiter.Allow() {
				mu.Unlock()
				app.rateLimitExceededResponse(w)
				return
			}

			mu.Unlock()
		}

		next.ServeHTTP(w, r)
	})
}

func (app *application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			r = contextSetUser(r, data.AnonymousUser)

			next.ServeHTTP(w, r)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			app.invalidAuthenticationTokenResponse(w)
			return
		}

		token := parts[1]

		claims, err := jwt.HMACCheck([]byte(token), []byte(app.cfg.JWT.Secret))
		if err != nil {
			app.invalidAuthenticationTokenResponse(w)
			return
		}

		if !claims.Valid(time.Now()) || claims.Issuer != "github.com/sharasha07/clash-bot" || !claims.AcceptAudience("github.com/sharasha07/clash-bot") {
			app.invalidAuthenticationTokenResponse(w)
			return
		}

		userID, err := strconv.ParseInt(claims.Subject, 10, 64)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}

		user, err := app.models.Users.GetByID(r.Context(), userID)
		if err != nil {
			switch {
			case errors.Is(err, data.ErrNoRecord):
				app.invalidAuthenticationTokenResponse(w)
			default:
				app.serverErrorResponse(w, r, err)
			}
			return
		}

		r = contextSetUser(r, &user)

		next.ServeHTTP(w, r)
	})
}
