package main

import (
	"errors"
	"fmt"
	"net/http"
)

func (app *application) sendError(w http.ResponseWriter, status int, message any) {
	err := app.writeJSON(w, status, envelope{"error": message})
	if err != nil {
		app.logger.Error("sendError failed", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (app *application) methodNotAllowedResponse(w http.ResponseWriter, r *http.Request) {
	message := fmt.Sprintf("%s method is not allowed for endpoint: %s", r.Method, r.URL.String())
	app.sendError(w, http.StatusMethodNotAllowed, message)
}

func (app *application) notFoundResponse(w http.ResponseWriter, r *http.Request) {
	message := "resource not found"
	app.sendError(w, http.StatusNotFound, message)
}

func (app *application) serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logger.Error("internal server error", "method", r.Method, "endpoint", r.URL.String(), "err", err)

	message := "internal server error"
	app.sendError(w, http.StatusInternalServerError, message)
}

func (app *application) rateLimitExceededResponse(w http.ResponseWriter) {
	message := "rate limit exceeded"
	app.sendError(w, http.StatusTooManyRequests, message)
}

func (app *application) badRequestResponse(w http.ResponseWriter, err error) {
	if err == nil {
		err = errors.New("bad request")
	}
	app.sendError(w, http.StatusBadRequest, err.Error())
}

func (app *application) failedValidationResponse(w http.ResponseWriter, errors map[string]string) {
	app.sendError(w, http.StatusUnprocessableEntity, errors)
}
