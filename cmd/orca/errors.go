package main

import "fmt"

type ErrArgumentRequired struct {
	Name  string
	Index int
}

func (err ErrArgumentRequired) Error() string {
	return fmt.Sprintf("the '%s' argument must be supplied at position %d", err.Name, err.Index)
}

type ErrTooManyArguments struct {
	Expected int
}

func (err ErrTooManyArguments) Error() string {
	return fmt.Sprintf("too many arguments supplied, expected only %d", err.Expected)
}
