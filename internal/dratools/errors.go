package dratools

import "fmt"

type Error struct {
	Kind string
	Msg  string
	Err  error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Msg, e.Err)
	}
	return e.Msg
}

func (e *Error) Unwrap() error { return e.Err }

func newError(kind, msg string) error { return &Error{Kind: kind, Msg: msg} }

func wrapError(kind, msg string, err error) error {
	return &Error{Kind: kind, Msg: msg, Err: err}
}
