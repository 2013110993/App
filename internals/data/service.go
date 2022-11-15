// Filename : internal/data/service.go

package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"app.federicorosado.net/internals/validator"
)

type Service struct {
	ID          int64     `json:"id"`
	CreatedAt   time.Time `json:"-"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Version     int32     `json:"version"`
}

// define a ServiceModel object that wraps a sql.DB connection pool
type ServiceModel struct {
	DB *sql.DB
}

func ValidateService(v *validator.Validator, service *Service) {
	// Use the Check() method to execute our validation checks
	v.Check(service.Title != "", "title", "must be provided")
	v.Check(len(service.Title) <= 200, "title", "must not be more than 200 bytes long")

	v.Check(service.Description != "", "description", "must be provided")
	v.Check(len(service.Description) <= 2000, "description", "must not be more than 2000 bytes long")
}

// Insert() allows us  to create a new Service
func (m ServiceModel) Insert(service *Service) error {
	query := `
		INSERT INTO services (title, description)
		VALUES ($1, $2)
		RETURNING id, created_at, version
	`

	// Collect the data fields into a slice
	args := []interface{}{
		service.Title, service.Description,
	}
	// Create a context
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	// Cleanup to prevent memory leaks
	defer cancel()

	return m.DB.QueryRowContext(ctx, query, args...).Scan(&service.ID, &service.CreatedAt, &service.Version)

}

// Get() allows us to retrieve a specific Service
func (m ServiceModel) Get(id int64) (*Service, error) {
	// Ensure that there is a valid id
	if id < 1 {
		return nil, ErrRecordNotFound
	}
	// Create the query
	query := `
		SELECT id, created_at, title, description, version
		FROM services
		WHERE id = $1
	`
	// Declare a Services variable to hold the returned data
	var service Service
	// Create a context
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	// Cleanup to prevent memory leaks
	defer cancel()
	// Execute the query using QueryRow()
	err := m.DB.QueryRowContext(ctx, query, id).Scan(
		&service.ID,
		&service.CreatedAt,
		&service.Title,
		&service.Description,
		&service.Version,
	)
	// Handle any errors
	if err != nil {
		// Check the type of error
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	// Success
	return &service, nil
}

// Update() allows us to edit/alter a specific Service
// Optimistic locking (version number)
func (m ServiceModel) Update(service *Service) error {
	// Create the query
	query := `
		UPDATE services
		SET title = $1, description = $2, version = version + 1
		WHERE id = $3
		AND version = $4
		RETURNING version
	`
	args := []interface{}{
		service.Title,
		service.Description,
		service.ID,
		service.Version,
	}

	// Create a context
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	// Cleanup to prevent memory leaks
	defer cancel()

	// Check for edit conflicts
	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&service.Version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}
	return nil
}

// Delete() removes a specific Service
func (m ServiceModel) Delete(id int64) error {
	// Ensure that there is a valid id
	if id < 1 {
		return ErrRecordNotFound
	}
	// Create the delete query
	query := `
		DELETE FROM services
		WHERE id = $1
	`

	// Create a context
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	// Cleanup to prevent memory leaks
	defer cancel()

	// Execute the query
	result, err := m.DB.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	// Check how many rows were affected by the delete operation. We
	// call the RowsAffected() method on the result variable
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	// Check if no rows were affected
	if rowsAffected == 0 {
		return ErrRecordNotFound
	}
	return nil
}

// The GetAll() method retuns a list of all the service sorted by id
func (m ServiceModel) GetAll(title string, filters Filters) ([]*Service, Metadata, error) {
	// Construct the query
	query := fmt.Sprintf(`
		SELECT COUNT(*) OVER(), id, created_at, title, description,
		       version
		FROM services
		WHERE (to_tsvector('simple', title) @@ plainto_tsquery('simple', $1) OR $1 = '')
		ORDER BY %s %s, id ASC
		LIMIT $2 OFFSET $3`, filters.sortColumn(), filters.sortOrder())

	// Create a 3-second-timout context
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	// Execute the query
	args := []interface{}{title, filters.limit(), filters.offset()}
	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	// Close the resultset
	defer rows.Close()
	totalRecords := 0
	// Initialize an empty slice to hold the Service data
	services := []*Service{}
	// Iterate over the rows in the resultset
	for rows.Next() {
		var service Service
		// Scan the values from the row into service
		err := rows.Scan(
			&totalRecords,
			&service.ID,
			&service.CreatedAt,
			&service.Title,
			&service.Description,
			&service.Version,
		)
		if err != nil {
			return nil, Metadata{}, err
		}
		// Add the Service to our slice
		services = append(services, &service)
	}
	// Check for errors after looping through the resultset
	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}
	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)
	// Return the slice of Service
	return services, metadata, nil
}
