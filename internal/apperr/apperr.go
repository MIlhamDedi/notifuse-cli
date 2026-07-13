package apperr

import "fmt"

const (
	ExitOK       = 0
	ExitGeneral  = 1
	ExitUsage    = 2
	ExitUpstream = 3
)

type Error struct {
	Code    int
	Message string
}

func (e Error) Error() string {
	return e.Message
}

func New(code int, format string, args ...any) error {
	return Error{Code: code, Message: fmt.Sprintf(format, args...)}
}

func Code(err error) int {
	if err == nil {
		return ExitOK
	}
	if app, ok := err.(Error); ok {
		return app.Code
	}
	return ExitGeneral
}

func FromStatus(status int) error {
	if status >= 200 && status <= 299 {
		return nil
	}
	return New(ExitUpstream, "notifuse returned HTTP %d", status)
}
