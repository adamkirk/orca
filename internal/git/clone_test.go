package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLsRemoteHasBranch(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		branch   string
		expected bool
	}{
		{
			name:     "no output",
			output:   "",
			branch:   "main",
			expected: false,
		},
		{
			name:     "exact match",
			output:   "3f2a1b\trefs/heads/orca-adoption\n",
			branch:   "orca-adoption",
			expected: true,
		},
		{
			name:     "only a ref ending with the branch",
			output:   "3f2a1b\trefs/heads/team/orca-adoption\n",
			branch:   "orca-adoption",
			expected: false,
		},
		{
			name:     "match among several refs",
			output:   "3f2a1b\trefs/heads/team/orca-adoption\n9c8d7e\trefs/heads/orca-adoption\n",
			branch:   "orca-adoption",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, lsRemoteHasBranch(tt.output, tt.branch))
		})
	}
}
