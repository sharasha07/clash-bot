package main

import (
	"errors"
	"net/http"
	"strings"

	"github.com/sharasha07/clash-bot/internal/data"
	"github.com/sharasha07/clash-bot/internal/validator"
)

func (app *application) showChatsHandler(w http.ResponseWriter, r *http.Request) {
	user := contextGetUser(r)
	if user.IsAnonymous() {
		app.authenticationRequiredResponse(w)
		return
	}

	var input struct {
		name string
		data.Filters
	}

	v := validator.New()
	qs := r.URL.Query()

	input.name = app.readString(qs, "name", "")
	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 10, v)
	input.Filters.Sort = app.readString(qs, "sort", "-updated_at")
	input.SortSafeList = []string{"id", "-id", "updated_at", "-updated_at"}

	if input.Filters.Validate(v); !v.Valid() {
		app.failedValidationResponse(w, v.Errors)
		return
	}

	chats, err := app.models.Chats.GetAll(r.Context(), user.ID, input.name, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if err := app.writeJSON(w, http.StatusOK, envelope{"chats": chats}); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) createChatHandler(w http.ResponseWriter, r *http.Request) {
	user := contextGetUser(r)
	if user.IsAnonymous() {
		app.authenticationRequiredResponse(w)
		return
	}

	var input struct {
		Name string `json:"name"`
	}

	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, err)
		return
	}

	v := validator.New()

	chat := data.Chat{
		UserID: user.ID,
		Name:   strings.TrimSpace(input.Name),
	}

	if chat.Validate(v); !v.Valid() {
		app.failedValidationResponse(w, v.Errors)
		return
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
	user := contextGetUser(r)
	if user.IsAnonymous() {
		app.authenticationRequiredResponse(w)
		return
	}

	id, err := app.readIDParam(r)
	if err != nil || id <= 0 {
		app.notFoundResponse(w, r)
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
	user := contextGetUser(r)
	if user.IsAnonymous() {
		app.authenticationRequiredResponse(w)
		return
	}

	id, err := app.readIDParam(r)
	if err != nil || id <= 0 {
		app.notFoundResponse(w, r)
		return
	}

	var input struct {
		Name *string `json:"name"`
	}

	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, err)
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

	if input.Name != nil {
		chat.Name = strings.TrimSpace(*input.Name)
	}

	v := validator.New()
	if chat.Validate(v); !v.Valid() {
		app.failedValidationResponse(w, v.Errors)
		return
	}

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
	user := contextGetUser(r)
	if user.IsAnonymous() {
		app.authenticationRequiredResponse(w)
		return
	}

	id, err := app.readIDParam(r)
	if err != nil || id <= 0 {
		app.notFoundResponse(w, r)
		return
	}

	if err := app.models.Chats.Delete(r.Context(), id, user.ID); err != nil {
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
