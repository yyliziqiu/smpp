package smpp

import (
	"errors"
	"fmt"

	"github.com/linxGnu/gosmpp/data"
)

var (
	ErrBindFailed       = errors.New("bind failed")
	ErrAuthFailed       = errors.New("auth failed")
	ErrWindowFull       = errors.New("window full")
	ErrNotAllowed       = errors.New("not allowed")
	ErrConnectionClosed = errors.New("connection closed")
	ErrChannelClosed    = errors.New("channel closed")
	ErrResponseTimeout  = errors.New("response timeout")
)

type StatusError struct {
	status data.CommandStatusType
}

func NewStatusError(status data.CommandStatusType) *StatusError {
	return &StatusError{status: status}
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("(%d) %s", e.status, e.status.Desc())
}
