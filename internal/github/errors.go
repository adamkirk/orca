package github

import "fmt"

type ErrUnexpectedApiError struct {
	Msg string
}

func (err ErrUnexpectedApiError) Error() string {
	return fmt.Sprintf("unexpected API error: %s", err.Msg)
}

type ErrRateLimitedByApi struct {
	Api string
}

func (err ErrRateLimitedByApi) Error() string {
	return fmt.Sprintf("rate limit exceeded for '%s'", err.Api)
}
