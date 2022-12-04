// Filename: internals/data/tokens.go

package data

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base32"
	"time"

	"app.federicorosado.net/internals/validator"
)

// Tokent Categories/Scopes
const (
	ScopeActivation     = "activation"
	ScopeAuthentication = "authentication"
)

// define the token type
type Token struct {
	Plaintext string    `json:"token"`
	Hash      []byte    `json:"-"`
	UserID    int64     `json:"-"`
	Expiry    time.Time `json:"expiry"`
	Scope     string    `json:"-"`
}

// generateToken() function returns a token
func generateToken(userID int64, ttl time.Duration, scope string) (*Token, error) {
	token := &Token{
		UserID: userID,
		Expiry: time.Now().Add(ttl),
		Scope:  scope,
	}
	// create a byte slice to hold random values and fill it with values
	// from CSPRNG
	randomBytes := make([]byte, 16)

	_, err := rand.Read(randomBytes)

	if err != nil {
		return nil, err
	}

	// encode the byte slice to a base-32 encoded string
	token.Plaintext = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)

	// hash the string token
	hash := sha256.Sum256([]byte(token.Plaintext))
	token.Hash = hash[:]

	return token, nil
}

// check that the plaintext token is 26 bytes long
func ValidateTokenPlaintext(v *validator.Validator, tokenPlaintext string) {
	v.Check(tokenPlaintext != "", "token", "must be 26 bytes long")
	v.Check(len(tokenPlaintext) == 26, "token", "must be 26 bytes long")

}

// Define the Token model
type TokenModel struct {
	DB *sql.DB
}

// create and insert a token into the tokens database
func (t *TokenModel) New(userID int64, ttl time.Duration, scope string) (*Token, error) {
	token, err := generateToken(userID, ttl, scope)
	if err != nil {
		return nil, err
	}

	err = t.Insert(token)
	return token, err
}

// Insert will insert a token into the tokens database
func (t *TokenModel) Insert(token *Token) error {
	query := `
			INSERT INTO tokens (hash, user_id, expiry, scope) 
			VALUES ($1, $2, $3, $4)
	`
	args := []interface{}{
		token.Hash,
		token.UserID,
		token.Expiry,
		token.Scope,
	}
	// create a context
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := t.DB.ExecContext(ctx, query, args...)

	return err
}

func (t *TokenModel) DeleteAllForUsers(scope string, userID int64) error {
	query := `
			DELETE FROM tokens 
			WHERE scope = $1 AND user_id = $2
	`
	args := []interface{}{
		scope,
		userID,
	}

	// create a context
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := t.DB.ExecContext(ctx, query, args...)

	return err
}
