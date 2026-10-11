package workspaces

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseGithubRepo(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected githubRepo
		wantErr  bool
	}{
		{
			name:  "shorthand",
			input: "acme/workspace",
			expected: githubRepo{
				Owner:    "acme",
				Name:     "workspace",
				CloneURL: "git@github.com:acme/workspace.git",
			},
		},
		{
			name:    "ssh url",
			input:   "git@github.com:acme/workspace.git",
			wantErr: true,
		},
		{
			name:    "missing owner",
			input:   "workspace",
			wantErr: true,
		},
		{
			name:    "empty repo",
			input:   "acme/",
			wantErr: true,
		},
		{
			name:    "https url",
			input:   "https://github.com/acme/workspace",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := parseGithubRepo(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
