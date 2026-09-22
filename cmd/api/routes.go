package main

import (
	"expvar"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)
	router.NotFound = http.HandlerFunc(app.notFoundResponse)

	router.HandlerFunc(http.MethodGet, "/health", app.health)
	router.Handler(http.MethodGet, "/debug", expvar.Handler())

	router.HandlerFunc(http.MethodPost, "/v1/users", app.createUserHandler)

	return app.recoverPanic(app.metrics(app.enableCORS(app.rateLimit(router))))
}
