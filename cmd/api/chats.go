package main

import (
	"errors"
	"net/http"

	"github.com/sharasha07/clash-bot/internal/data"
)

func (app *application) createChatHandler(w http.ResponseWriter, r *http.Request) {
	user := contextGetUser(r)

	if user.IsAnonymous() {
		app.authenticationRequiredResponse(w)
		return
	}

	var input struct {
		Name string `json:"name" validate:"required,max=20"`
	}

	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, err)
		return
	}

	if err := app.validate.Struct(input); err != nil {
		app.failedValidationResponse(w, app.fieldErrors(err))
		return
	}

	chat := data.Chat{
		UserID: user.ID,
		Name:   input.Name,
	}

	if err := app.models.Chats.Insert(r.Context(), &chat); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if err := app.writeJSON(w, http.StatusCreated, envelope{"chat": chat}); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) showChatHandler(w http.ResponseWriter, r *http.Request) {
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

	chat, err := app.models.Chats.Get(r.Context(), id, user.ID)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrNoRecord):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	if err := app.writeJSON(w, http.StatusOK, envelope{"chat": chat}); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) updateChatHandler(w http.ResponseWriter, r *http.Request) {
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

	chat, err := app.models.Chats.Get(r.Context(), id, user.ID)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrNoRecord):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	var input struct {
		Name string `json:"name" validate:"required,max=20"`
	}

	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, err)
		return
	}

	if err := app.validate.Struct(input); err != nil {
		app.failedValidationResponse(w, app.fieldErrors(err))
		return
	}

	chat.Name = input.Name

	if err := app.models.Chats.Update(r.Context(), &chat); err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	if err := app.writeJSON(w, http.StatusOK, envelope{"chat": chat}); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) deleteChatHandler(w http.ResponseWriter, r *http.Request) {
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

	if err := app.models.Chats.Delete(r.Context(), id); err != nil {
		switch {
		case errors.Is(err, data.ErrNoRecord):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
