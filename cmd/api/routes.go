package main

import (
	"expvar"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)
	router.NotFound = http.HandlerFunc(app.notFoundResponse)

	// metrics
	router.HandlerFunc(http.MethodGet, "/health", app.health)
	router.Handler(http.MethodGet, "/debug", expvar.Handler())

	// users
	router.HandlerFunc(http.MethodPost, "/v1/users", app.createUserHandler)
	router.HandlerFunc(http.MethodPost, "/v1/users/:id/profile-picture", app.uploadProfilePictureHandler)
	router.HandlerFunc(http.MethodGet, "/v1/users/:id", app.showUserHandler)
	router.HandlerFunc(http.MethodPatch, "/v1/users/:id", app.updateUserHandler)
	router.HandlerFunc(http.MethodDelete, "/v1/users/:id", app.deleteUserHandler)

	// chats
	router.HandlerFunc(http.MethodGet, "/v1/chats", app.showChatsHandler)
	router.HandlerFunc(http.MethodPost, "/v1/chats", app.createChatHandler)
	router.HandlerFunc(http.MethodGet, "/v1/chats/:id", app.showChatHandler)
	router.HandlerFunc(http.MethodPatch, "/v1/chats/:id", app.updateChatHandler)
	router.HandlerFunc(http.MethodDelete, "/v1/chats/:id", app.deleteChatHandler)

	// messages
	router.HandlerFunc(http.MethodGet, "/v1/chats/:id/messages", app.showMessagesHandler)
	router.HandlerFunc(http.MethodPost, "/v1/chats/:id/messages", app.createMessageHandler)

	// tokens
	router.HandlerFunc(http.MethodPost, "/v1/auth/login", app.loginHandler)
	router.HandlerFunc(http.MethodPost, "/v1/auth/refresh", app.refreshTokenHandler)
	router.HandlerFunc(http.MethodPost, "/v1/auth/logout", app.logoutHandler)

	return app.metrics(app.recoverPanic(app.enableCORS(app.rateLimit(app.authenticate(router)))))
}
