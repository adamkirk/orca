package docker

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_attachArgs(t *testing.T) {
	assert.Equal(t, []string{"exec", "-it"}, attachArgs("exec", nil), "interactive, on the terminal")
	assert.Equal(t, []string{"run", "-T"}, attachArgs("run", &bytes.Buffer{}), "not attached, when the output goes elsewhere")
}
