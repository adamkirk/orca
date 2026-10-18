package controller

import "fmt"

type ErrUnknownService struct {
	Name string
}

func (err ErrUnknownService) Error() string {
	return fmt.Sprintf("unknown service: %s", err.Name)
}
