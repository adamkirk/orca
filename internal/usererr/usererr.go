// Package usererr attaches messages meant for the user to errors.
//
// Code that fails returns an error wrapped with a message, rather than printing
// it, and the CLI shows the message once, when the error reaches it:
//
//	if err := os.MkdirAll(dir, 0o755); err != nil {
//		return usererr.Wrap(err, "Failed to create the certificates directory!")
//	}
package usererr

import "errors"

type userError struct {
	msg string
	err error
}

func (e *userError) Error() string {
	return e.msg + ": " + e.err.Error()
}

func (e *userError) Unwrap() error {
	return e.err
}

// Wrap attaches a message for the user to err. It returns nil if err is nil, so
// it can wrap the result of a call directly.
func Wrap(err error, msg string) error {
	if err == nil {
		return nil
	}

	return &userError{msg: msg, err: err}
}

// Message returns the innermost (most specific) message in err's chain, along
// with the error it was attached to.
func Message(err error) (msg string, cause error, ok bool) {
	for err != nil {
		var u *userError

		if !errors.As(err, &u) {
			break
		}

		msg, cause, ok = u.msg, u.err, true
		err = u.err
	}

	return msg, cause, ok
}
