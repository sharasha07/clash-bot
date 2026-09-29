package main

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/sharasha07/clash-bot/internal/data"
	"github.com/sharasha07/clash-bot/internal/validator"
)

func (app *application) showMessagesHandler(w http.ResponseWriter, r *http.Request) {
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

	if _, err := app.models.Chats.Get(r.Context(), id, user.ID); err != nil {
		switch {
		case errors.Is(err, data.ErrNoRecord):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	v := validator.New()
	qs := r.URL.Query()

	var filters data.Filters
	filters.Page = app.readInt(qs, "page", 1, v)
	filters.PageSize = app.readInt(qs, "page_size", 10, v)
	filters.Sort = app.readString(qs, "sort", "-id")
	filters.SortSafeList = []string{"id", "-id", "created_at", "-created_at"}

	if filters.Validate(v); !v.Valid() {
		app.failedValidationResponse(w, v.Errors)
		return
	}

	messages, err := app.models.Messages.GetAll(r.Context(), id, user.ID, filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if err := app.writeJSON(w, http.StatusOK, envelope{"messages": messages}); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) createMessageHandler(w http.ResponseWriter, r *http.Request) {
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

	ctx, cancel := context.WithTimeout(r.Context(), app.cfg.Server.WriteTimeout-5*time.Second)
	defer cancel()

	if _, err := app.models.Chats.Get(ctx, id, user.ID); err != nil {
		switch {
		case errors.Is(err, data.ErrNoRecord):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	var input struct {
		Content string `json:"content"`
	}

	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, err)
		return
	}

	message := data.Message{
		ChatID:  id,
		Role:    data.RoleUser,
		Content: strings.TrimSpace(input.Content),
	}

	v := validator.New()
	if message.Validate(v); !v.Valid() {
		app.failedValidationResponse(w, v.Errors)
		return
	}

	if err := app.models.Messages.Insert(ctx, &message); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if _, err := app.models.Chats.Touch(ctx, id, user.ID); err != nil {
		app.logger.Error("failed to touch chat", "err", err, "chat_id", id)
	}

	history, err := app.models.Messages.GetAll(ctx, id, user.ID, data.Filters{
		Page:         1,
		PageSize:     20,
		Sort:         "-id",
		SortSafeList: []string{"id", "-id", "created_at", "-created_at"},
	})

	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	slices.Reverse(history)

	prompt := app.buildPrompt(history)

	var assistantMessage *data.Message

	reply, err := app.llmClient.generateReply(ctx, prompt, user.GameTag)
	if err != nil {
		app.logger.Error("failed to generate reply", "err", err, "chat_id", id)
	} else {
		assistantMessage = &data.Message{
			ChatID:  id,
			Role:    data.RoleAssistant,
			Content: reply,
		}

		if err := app.models.Messages.Insert(ctx, assistantMessage); err != nil {
			app.logger.Error("failed to store reply", "err", err, "chat_id", id)
			assistantMessage = nil
		}
	}

	if err := app.writeJSON(w, http.StatusCreated, envelope{"message": message, "reply": assistantMessage}); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
