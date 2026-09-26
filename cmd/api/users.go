package main

import (
	"errors"
	"fmt"
	"image"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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

func (app *application) uploadProfilePicture(w http.ResponseWriter, r *http.Request) {
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

	const maxRequestSize = 5 << 20
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestSize)

	err = r.ParseMultipartForm(maxRequestSize)
	if err != nil {
		app.badRequestResponse(w, errors.New("malformed upload"))
		return
	}

	file, _, err := r.FormFile("avatar")
	if err != nil {
		switch {
		case errors.Is(err, http.ErrMissingFile):
			app.badRequestResponse(w, errors.New("avatar file is required"))
		default:
			app.badRequestResponse(w, errors.New("malformed upload"))
		}
		return
	}
	defer file.Close()

	_, format, err := image.DecodeConfig(file)
	if err != nil {
		app.badRequestResponse(w, errors.New("invalid image"))
		return
	}

	supportedFormats := []string{"jpeg", "png", "webp"}
	if !slices.Contains(supportedFormats, format) {
		app.badRequestResponse(w, errors.New("unsupported image type"))
		return
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	key := fmt.Sprintf("users/%d/profile_picture", id)

	endpoint, err := url.JoinPath(app.cfg.R2.PublicURL, key)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	endpoint += "?v=" + strconv.FormatInt(time.Now().UnixNano(), 10)

	_, err = app.s3Client.PutObject(r.Context(),
		&s3.PutObjectInput{
			Bucket:       aws.String(app.cfg.R2.Bucket),
			Key:          aws.String(key),
			Body:         file,
			ContentType:  aws.String("image/" + format),
			CacheControl: aws.String("public, max-age=3600"),
		},
	)

	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	user.ProfilePicture = &endpoint

	err = app.models.Users.Update(r.Context(), user)
	if err != nil {
		_, delErr := app.s3Client.DeleteObject(r.Context(), &s3.DeleteObjectInput{
			Bucket: aws.String(app.cfg.R2.Bucket),
			Key:    aws.String(key),
		})

		if delErr != nil {
			app.logger.Error("failed to remove orphaned profile picture", "err", delErr)
		}

		switch {
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

func (app *application) deleteUserHandler(w http.ResponseWriter, r *http.Request) {
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

	err = app.models.Users.Delete(r.Context(), id)
	if err != nil {
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
