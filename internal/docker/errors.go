package docker

import "fmt"

type ErrInvalidValueForOverlayModifier struct {
	Label   string
	Message string
	Service string
}

func (err ErrInvalidValueForOverlayModifier) Error() string {
	svc := ""

	if err.Service != "" {
		svc = fmt.Sprintf("(service=%s) ", err.Service)
	}

	return fmt.Sprintf("invalid value for label '%s' modifier in compose file %s: %s", err.Label, svc, err.Message)
}
