// Filename cmd/api/routes

package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *application) routes() http.Handler {
	// Create new http router instance
	router := httprouter.New()
	router.NotFound = http.HandlerFunc(app.notFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.MethodNotAllowedReponse)
	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthcheckHandler)
	router.HandlerFunc(http.MethodGet, "/v1/services", app.requiredPermission("rm:read", app.listServiceHandler))
	router.HandlerFunc(http.MethodPost, "/v1/services", app.requiredPermission("rm:write", app.createServiceHandler))
	router.HandlerFunc(http.MethodGet, "/v1/services/:id", app.requiredPermission("rm:read", app.showSerivceHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/services/:id", app.requiredPermission("rm:write", app.updateServiceHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/services/:id", app.requiredPermission("rm:write", app.deleteServiceHandler))
	router.HandlerFunc(http.MethodPost, "/v1/users", app.registerUserHandler)
	router.HandlerFunc(http.MethodPut, "/v1/users/activate", app.activateUserHandler)
	router.HandlerFunc(http.MethodPost, "/v1/tokens/authentication", app.createAuthenticationTokenHandler)

	return app.recoverPanic(app.enableCORS(app.rateLimit(app.authenticate(router))))
}
