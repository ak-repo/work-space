package password

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

var ErrMismatch = errors.New("password: hash and password do not match")

// Hash returns a bcrypt hash of the plain-text password.
func Hash(plain string, cost int) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), cost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Verify compares a plain-text password against a bcrypt hash.
// Returns ErrMismatch if they do not match.
func Verify(hash, plain string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return ErrMismatch
	}
	return err
}
