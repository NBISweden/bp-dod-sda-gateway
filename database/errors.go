package database

import "errors"

var (
	// ErrAlreadyExists indicates that an operation violates a uniqueness constraint.
	ErrAlreadyExists = errors.New("already exists")
)
