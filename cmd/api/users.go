package main

import (
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/sharasha07/clash-bot/internal/data"
	"github.com/sharasha07/clash-bot/internal/validator"
)

func (app *application) createUserHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, err)
		return
	}

	user := data.User{Username: strings.TrimSpace(input.Username)}
	user.Password.Plain = input.Password

	v := validator.New()
	if user.Validate(v); !v.Valid() {
		app.failedValidationResponse(w, v.Errors)
		return
	}

	if err := user.Password.SetHash(input.Password); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if err := app.models.Users.Insert(r.Context(), &user); err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateUsersUsername):
			v.Add("username", "must be unique")
			app.failedValidationResponse(w, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	if err := app.writeJSON(w, http.StatusCreated, envelope{"user": user}); err != nil {
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

	supportedFormats := []string{"jpeg", "png"}
	if !slices.Contains(supportedFormats, format) {
		app.badRequestResponse(w, errors.New("unsupported image type"))
		return
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	key := fmt.Sprintf("users/%d/profile_picture", id)

	endpoint, err := url.JoinPath(app.cfg.R2.PublicURL, app.cfg.R2.Bucket, key)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

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

	user := contextGetUser(r)
	if user.IsAnonymous() {
		app.authenticationRequiredResponse(w)
		return
	}

	if id != user.ID {
		app.forbiddenResponse(w)
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
		Username *string `json:"username"`
		Password *string `json:"password"`
		GameTag  *string `json:"game_tag"`
	}

	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, err)
		return
	}

	if input.Username != nil {
		user.Username = strings.TrimSpace(*input.Username)
	}

	if input.GameTag != nil {
		user.GameTag = input.GameTag
	}

	v := validator.New()
	if user.ValidateProfile(v); !v.Valid() {
		app.failedValidationResponse(w, v.Errors)
		return
	}

	if input.Password != nil {
		user.Password.Plain = *input.Password

		if user.Password.Validate(v); !v.Valid() {
			app.failedValidationResponse(w, v.Errors)
			return
		}

		if err := user.Password.SetHash(*input.Password); err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}
	}

	err = app.models.Users.Update(r.Context(), user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateUsersUsername):
			v.Add("username", "must be unique")
			app.failedValidationResponse(w, v.Errors)
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

	key := fmt.Sprintf("users/%d/profile_picture", id)

	_, delErr := app.s3Client.DeleteObject(r.Context(), &s3.DeleteObjectInput{
		Bucket: aws.String(app.cfg.R2.Bucket),
		Key:    aws.String(key),
	})

	if delErr != nil {
		app.logger.Error("failed to remove orphaned profile picture", "err", delErr)
	}

	w.WriteHeader(http.StatusNoContent)
}
