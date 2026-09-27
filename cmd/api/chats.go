package main

import (
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/sharasha07/clash-bot/internal/data"
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

	qs := r.URL.Query()
	input.name = qs.Get("name")
	if err := app.readFilters(r, &input.Filters); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if input.Filters.Sort == "" {
		input.Filters.Sort = "-updated_at"
	}

	sortList := []string{"id", "-id", "updated_at", "-updated_at"}

	if !slices.Contains(sortList, input.Filters.Sort) {
		app.failedValidationResponse(w, map[string]string{
			"sort": fmt.Sprintf("must be in: %v", sortList),
		})
		return
	}

	if err := app.validate.Struct(input); err != nil {
		app.failedValidationResponse(w, app.fieldErrors(err))
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
