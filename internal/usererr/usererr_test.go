package usererr_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/adamkirk/orca/internal/usererr"
	"github.com/stretchr/testify/assert"
)

var errCause = errors.New("permission denied")

func Test_Wrap_Nil(t *testing.T) {
	assert.NoError(t, usererr.Wrap(nil, "Failed!"))
}

func Test_Wrap_KeepsTheChain(t *testing.T) {
	err := usererr.Wrap(errCause, "Failed to save!")

	assert.ErrorIs(t, err, errCause)
	assert.EqualError(t, err, "Failed to save!: permission denied")
}

func Test_Message(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		expectMsg   string
		expectCause error
		expectOk    bool
	}{
		{
			name: "no message",
			err:  errCause,
		},
		{
			name:        "single message",
			err:         usererr.Wrap(errCause, "Failed to save!"),
			expectMsg:   "Failed to save!",
			expectCause: errCause,
			expectOk:    true,
		},
		{
			name:        "innermost message wins",
			err:         usererr.Wrap(usererr.Wrap(errCause, "Failed to checkout branch"), "Failed to checkout"),
			expectMsg:   "Failed to checkout branch",
			expectCause: errCause,
			expectOk:    true,
		},
		{
			name:        "found through other wrapping",
			err:         fmt.Errorf("checking out: %w", usererr.Wrap(errCause, "Failed to checkout branch")),
			expectMsg:   "Failed to checkout branch",
			expectCause: errCause,
			expectOk:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, cause, ok := usererr.Message(tt.err)

			assert.Equal(t, tt.expectMsg, msg)
			assert.Equal(t, tt.expectCause, cause)
			assert.Equal(t, tt.expectOk, ok)
		})
	}
}
