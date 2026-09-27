package main

import (
	"errors"
	"net/http"

	"github.com/alexedwards/argon2id"
	"github.com/sharasha07/clash-bot/internal/data"
	"github.com/sharasha07/clash-bot/internal/validator"
)

func (app *application) loginHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, err)
		return
	}

	user, err := app.models.Users.GetByUsername(r.Context(), input.Username)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrNoRecord):
			app.invalidCredentialsResponse(w)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	match, err := argon2id.ComparePasswordAndHash(input.Password, string(user.Password.Hash))
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if !match {
		app.invalidCredentialsResponse(w)
		return
	}

	jwtToken, err := data.NewAccessToken(user.ID, app.cfg.JWT.Secret, app.cfg.JWT.AccessTTL)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	refreshToken, err := data.NewRefreshToken()
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.models.Tokens.Insert(r.Context(), refreshToken, user.ID, app.cfg.JWT.RefreshTTL)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{
		"access_token":  jwtToken,
		"refresh_token": refreshToken,
	})

	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) refreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, err)
		return
	}

	if input.RefreshToken == "" {
		v := validator.New()
		v.Add("refresh_token", "must be provided")
		app.failedValidationResponse(w, v.Errors)
		return
	}

	userID, err := app.models.Tokens.GetUserID(r.Context(), input.RefreshToken)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrNoRecord):
			app.invalidAuthenticationTokenResponse(w)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	newAccessToken, err := data.NewAccessToken(userID, app.cfg.JWT.Secret, app.cfg.JWT.AccessTTL)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"access_token": newAccessToken})
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) logoutHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		RefreshToken string `json:"refresh_token"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, err)
		return
	}

	if input.RefreshToken == "" {
		v := validator.New()
		v.Add("refresh_token", "must be provided")
		app.failedValidationResponse(w, v.Errors)
		return
	}

	err = app.models.Tokens.Delete(r.Context(), input.RefreshToken)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
