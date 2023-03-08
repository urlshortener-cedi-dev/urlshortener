package model

import "fmt"

type NotAllowedError struct {
	Username      string
	Operation     string
	ShortlinkName string
}

func NewNotAllowedError(username, operation, shortlinkName string) *NotAllowedError {
	return &NotAllowedError{
		Username:      username,
		Operation:     operation,
		ShortlinkName: shortlinkName,
	}
}

func (e *NotAllowedError) Error() string {
	return fmt.Sprintf("Operation '%s' for user '%s' is not allowed for ShortLink '%s'",
		e.Operation,
		e.Username,
		e.ShortlinkName,
	)
}
