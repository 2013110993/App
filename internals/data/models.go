// Filename: internal/data/models.go

package data

import (
	"database/sql"
	"errors"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

// A wrapper for out data models
type Models struct {
	Service ServiceModel
	User    UserModel
	Tokens  TokenModel
}

// NewModels() allows us to create new models
func NewModels(db *sql.DB) *Models {
	return &Models{
		Service: ServiceModel{DB: db},
		User:    UserModel{DB: db},
		Tokens:  TokenModel{DB: db},
	}
}
