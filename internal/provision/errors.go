package provision

import "fmt"

type ErrUnsupportedProvisioner struct {
	Message string
}

func (err ErrUnsupportedProvisioner) Error() string {
	return fmt.Sprintf("unsupported provisioner: %s", err.Message)
}
