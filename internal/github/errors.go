package github

import "errors"

var (
	ErrUnexpectedAPIResponse = errors.New("unexpected API response")
	ErrRateLimited           = errors.New("rate limit exceeded")
)
