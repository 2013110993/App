// Filename: cmd/api/middleware.go

package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"app.federicorosado.net/internals/data"
	"app.federicorosado.net/internals/validator"
	"golang.org/x/time/rate"
)

func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				app.serverErrorResponse(w, r, fmt.Errorf("%s", err))
			}
		}()
		next.ServeHTTP(w, r)
	})

}

func (app *application) rateLimit(next http.Handler) http.Handler {
	// creaet a client type
	type Client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	var (
		mu      sync.Mutex
		clients = make(map[string]*Client)
	)
	// launch a background goroutine that remove old clients
	// from the clients map every minute
	go func() {
		for {
			time.Sleep(time.Minute)
			// Lock before starting to clean up
			mu.Lock()
			for ip, client := range clients {
				if time.Since(client.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}
		}
	}()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if app.config.limiter.enable {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				app.serverErrorResponse(w, r, err)
				return
			}
			// Lock
			mu.Lock()
			// check if the IP is in the map
			if _, found := clients[ip]; !found {
				clients[ip] = &Client{limiter: rate.NewLimiter(
					rate.Limit(app.config.limiter.rps),
					app.config.limiter.burst,
				)}
			}

			// update lastSeen
			clients[ip].lastSeen = time.Now()

			// check if the request is allowed
			if !clients[ip].limiter.Allow() {
				mu.Unlock()
				app.rateLimitExceededResponse(w, r)
				return
			}
			mu.Unlock()
		} // end of enable conditional
		next.ServeHTTP(w, r)
	})
}

// Authentication
func (app *application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// add a "vary: Authorization" header to the response
		// a note to the caches that the response may vary
		w.Header().Add("Vary", "Authorization")
		authorizationHeader := r.Header.Get("Authorization")
		// if no authorizationHeader found then we create a new anonymous user
		if authorizationHeader == "" {
			r = app.contextSetUser(r, data.AnonymousUser)
			next.ServeHTTP(w, r)
			return
		}
		// check if the provided authorization header is the right format
		headerParts := strings.Split(authorizationHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		// extract the token
		token := headerParts[1]
		// validate the token
		v := validator.New()

		if data.ValidateTokenPlaintext(v, token); !v.Valid() {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		// retrieve the detail about the user
		user, err := app.models.User.GetForToken(data.ScopeAuthentication, token)
		if err != nil {
			switch {
			case errors.Is(err, data.ErrRecordNotFound):
				app.invalidAuthenticationTokenResponse(w, r)
			default:
				app.serverErrorResponse(w, r, err)
			}
			return
		}
		// add the user information to the request context
		r = app.contextSetUser(r, user)
		// call the next handler in the chain
		next.ServeHTTP(w, r)
	})
}

// check for activated user
func (app *application) requiredAuthenticatedUser(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the user
		user := app.contextGetUser(r)
		// check for anonymous user
		if user.IsAnonymous() {
			app.authenticationRequiredResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// check for activated user
func (app *application) requiredActivatedUser(next http.HandlerFunc) http.HandlerFunc {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the user
		user := app.contextGetUser(r)
		// check if user is activated
		if !user.Activated {
			app.inactiveAccountResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
	return app.requiredAuthenticatedUser(fn)
}

// Check for specific permission
func (app *application) requiredPermission(code string, next http.HandlerFunc) http.HandlerFunc {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// get the user
		user := app.contextGetUser(r)
		// get the permission slice for the user
		permissions, err := app.models.Permissions.GetAllForUser(user.ID)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}
		// check for the permission
		if !permissions.Include(code) {
			app.notPermittedResponse(w, r)
			return
		}
		// ok
		next.ServeHTTP(w, r)
	})
	return app.requiredActivatedUser(fn)
}
