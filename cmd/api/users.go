package main

import (
	"errors"
	"net/http"
	"strings"

	"github.com/sharasha07/clash-bot/internal/data"
)

func (app *application) createUserHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Username string `json:"username" validate:"required,max=10"`
		Password string `json:"password" validate:"required,min=8,max=30"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, err)
		return
	}

	input.Username = strings.TrimSpace(input.Username)
	if err := app.validate.Struct(input); err != nil {
		app.failedValidationResponse(w, app.fieldErrors(err))
		return
	}

	user := data.User{Username: input.Username}
	err = user.SetPassword(input.Password)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.models.Users.Insert(r.Context(), &user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateUsersUsername):
			app.failedValidationResponse(w, map[string]string{"username": "must be unique"})
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

func (app *application) showUserHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil || id <= 0 {
		app.notFoundResponse(w, r)
		return
	}

	user, err := app.models.Users.GetByID(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrNoRecord):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"user": user})
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) updateUserHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil || id <= 0 {
		app.notFoundResponse(w, r)
		return
	}

	user := contextGetUser(r)
	if user.IsAnonymous() {
		app.authenticationRequiredResponse(w)
		return
	}

	if user.ID != id {
		app.forbiddenResponse(w)
		return
	}

	var input struct {
		Username *string `json:"username" validate:"omitempty,max=10"`
		Passowrd *string `json:"password" validate:"omitempty,min=8,max=30"`
		GameTag  *string `json:"game_tag" validate:"omitempty"`
	}

	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, err)
		return
	}

	if err := app.validate.Struct(&input); err != nil {
		app.failedValidationResponse(w, app.fieldErrors(err))
		return
	}

	if input.Username != nil {
		user.Username = *input.Username
	}

	if input.Passowrd != nil {
		err := user.SetPassword(*input.Passowrd)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}
	}

	if input.GameTag != nil {
		user.GameTag = input.GameTag
	}

	err = app.models.Users.Update(r.Context(), user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateUsersUsername):
			app.failedValidationResponse(w, map[string]string{
				"username": "must be unique",
			})
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"user": user})
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}
