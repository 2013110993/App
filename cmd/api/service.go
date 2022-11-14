package main

import (
	"errors"
	"fmt"
	"net/http"

	"app.federicorosado.net/internals/data"
	"app.federicorosado.net/internals/validator"
)

// createServiceHandler() for the "POST /v1/service" endpoint
func (app *application) createServiceHandler(w http.ResponseWriter, r *http.Request) {
	// Our target decode destination
	var input struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	// Initialize a new json.Decoder instance
	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Copy the values from the input struct to a new Service struct
	service := &data.Service{
		Title:       input.Title,
		Description: input.Description,
	}

	// Initialize a new Validator instance
	v := validator.New()

	// Check the map to determine if there were any validation errors
	if data.ValidateService(v, service); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Create a Service
	err = app.models.Service.Insert(service)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
	// Create a Location header for the newly created resource/Service
	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/service/%d", service.ID))
	// Write the JSON response with 201 - Created status code with the body
	// being the Service data and the header being the headers map
	err = app.writeJSON(w, http.StatusCreated, envelope{"service": service}, headers)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// showServiceHandler for the "GET /v1/service/:id" endpoint
func (app *application) showSerivceHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	// Fetch the specific service
	service, err := app.models.Service.Get(id)
	// Handle errors
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"service": service}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

}

func (app *application) updateServiceHandler(w http.ResponseWriter, r *http.Request) {
	// This method does a partial replacement
	// Get the id for the service that needs updating
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}
	// Fetch the orginal record from the database
	service, err := app.models.Service.Get(id)
	// Handle errors
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	// Create an input struct to hold data read in from the client
	// We update input struct to use pointers because pointers have a
	// default value of nil
	// If a field remains nil then we know that the client did not update it
	var input struct {
		Title       *string `json:"title"`
		Description *string `json:"description"`
	}

	// Initialize a new json.Decoder instance
	err = app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Check for updates
	if input.Title != nil {
		service.Title = *input.Title
	}
	if input.Description != nil {
		service.Description = *input.Description
	}

	// Perform validation on the updated Description. If validation fails, then
	// we send a 422 - Unprocessable Entity respose to the client
	// Initialize a new Validator instance
	v := validator.New()

	// Check the map to determine if there were any validation errors
	if data.ValidateService(v, service); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	// Pass the updated Service record to the Update() method
	err = app.models.Service.Update(service)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	// Write the data returned by Get()
	err = app.writeJSON(w, http.StatusOK, envelope{"service": service}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

}

func (app *application) deleteServiceHandler(w http.ResponseWriter, r *http.Request) {
	// Get the id for the service that needs updating
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}
	// Delete the Service from the database. Send a 404 Not Found status code to the
	// client if there is no matching record
	err = app.models.Service.Delete(id)
	// Handle errors
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	// Return 200 Status OK to the client with a success message
	err = app.writeJSON(w, http.StatusOK, envelope{"message": "service successfully deleted"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// The listServiceHandler() allows the client to see a listing of service
// based on a set of criteria
func (app *application) listServiceHandler(w http.ResponseWriter, r *http.Request) {
	// Create an input struct to hold our query parameters
	var input struct {
		Title string
		data.Filters
	}
	// Initialize a validator
	v := validator.New()
	// Get the URL values map
	qs := r.URL.Query()
	// Use the helper methods to extract the values
	input.Title = app.readString(qs, "title", "")
	//input.Message = app.readString(qs, "message", "")
	// Get the page information
	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)
	// Get the sort information
	input.Filters.Sort = app.readString(qs, "sort", "id")
	// Specific the allowed sort values
	input.Filters.SortList = []string{"id", "title", "-id", "-title"}
	// Check for validation errors
	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Get a listing of all service
	service, metadata, err := app.models.Service.GetAll(input.Title, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	// Send a JSON response containg all the service
	err = app.writeJSON(w, http.StatusOK, envelope{"service": service, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}
