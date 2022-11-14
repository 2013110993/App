// Filename: cmd/api/tokens.go
package main

import (
	"errors"
	"net/http"
	"time"

	"app.federicorosado.net/internals/data"
	"app.federicorosado.net/internals/validator"
)

func (app *application) createAuthenticationTokenHandler(w http.ResponseWriter, r *http.Request) {
	// parse the email and password from the request body

	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := app.readJSON(w, r, &input)

	if err != nil {

		app.badResquestReponse(w, r, err)
		return
	}

	//create new validator
	v := validator.New()

	data.ValidateEmail(v, input.Email)
	data.ValidatePasswordPlaintext(v, input.Password)

	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Get the user details based on the email and password provided
	user, err := app.models.User.GetByEmail(input.Email)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.invalidCredentialsResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	// check if the password matches
	match, err := user.Password.Matches(input.Password)

	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// if password dont match then return invalid credentials
	if !match {
		app.invalidCredentialsResponse(w, r)
		return
	}

	// password is correct, so we will generate a auth token
	token, err := app.models.Tokens.New(user.ID, 24*time.Hour, data.ScopeAuthentication)

	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// return the auth token to the client
	err = app.writeJSON(w, http.StatusCreated, envelope{"authentication_token": token}, nil)

	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
