package main

import (
	"errors"
	"net/http"

	"github.com/sharasha07/clash-bot/internal/data"
	"github.com/sharasha07/clash-bot/internal/validator"
)

func (app *application) createUserHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, err)
		return
	}

	user := data.User{Username: input.Username}
	err = user.Password.Set(input.Password)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	v := validator.New()
	if user.Validate(v); !v.Valid() {
		app.failedValidationResponse(w, v.Errors())
		return
	}

	err = app.models.Users.Insert(r.Context(), &user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrUniqueViolation):
			v.AddError("username", "must be unique")
			app.failedValidationResponse(w, v.Errors())
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusCreated, envelope{"user": user})
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}
