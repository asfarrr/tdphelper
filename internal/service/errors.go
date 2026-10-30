package service

import "errors"

var (
	// ErrInvalid marks input that fails validation.
	ErrInvalid = errors.New("invalid input")
	// ErrNotFound marks a missing record.
	ErrNotFound = errors.New("not found")
	// ErrInsufficientStock marks an attempt to take more filament than a spool holds.
	ErrInsufficientStock = errors.New("insufficient stock")
)
