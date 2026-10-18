package github

import (
	"errors"
	"fmt"
)

var ErrUnexpectedApiError = errors.New("unexpected API error")

type ErrRateLimitedByApi struct {
	Api string
}

func (err ErrRateLimitedByApi) Error() string {
	return fmt.Sprintf("rate limit exceeded for '%s'", err.Api)
}
