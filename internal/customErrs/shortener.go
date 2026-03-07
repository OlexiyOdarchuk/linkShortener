package customerrs

import "errors"

var (
	ErrInvalidCharacter = errors.New("invalid character")
	ErrCodeIsBusy       = errors.New("code is busy")
)
