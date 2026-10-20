package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseRemoteBranches(t *testing.T) {
	local := Branches{{Name: "main", Current: true}, {Name: "feature/local"}}

	tests := []struct {
		name     string
		output   string
		expected Branches
	}{
		{
			name:     "no remotes",
			output:   "",
			expected: Branches{},
		},
		{
			name:   "remote-only branches, in order",
			output: "feature/x\nfeature/a\n",
			expected: Branches{
				{Name: "feature/x", RemoteOnly: true},
				{Name: "feature/a", RemoteOnly: true},
			},
		},
		{
			name:     "branches that are also local are left out",
			output:   "main\nfeature/local\nfeature/x\n",
			expected: Branches{{Name: "feature/x", RemoteOnly: true}},
		},
		{
			name:     "the remote's HEAD is left out",
			output:   "HEAD\nmain\n",
			expected: Branches{},
		},
		{
			name:     "on several remotes, listed once",
			output:   "feature/x\nfeature/x\n",
			expected: Branches{{Name: "feature/x", RemoteOnly: true}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, parseRemoteBranches(tt.output, local))
		})
	}
}

func TestBranchesSearch(t *testing.T) {
	branches := Branches{{Name: "main"}, {Name: "feature/Payments"}, {Name: "feature/search"}}

	tests := []struct {
		name     string
		search   string
		expected Branches
	}{
		{name: "part of a name", search: "search", expected: Branches{{Name: "feature/search"}}},
		{name: "ignoring the search's case", search: "SEARCH", expected: Branches{{Name: "feature/search"}}},
		{name: "ignoring the branch's case", search: "payments", expected: Branches{{Name: "feature/Payments"}}},
		{name: "several matches", search: "feature/", expected: Branches{{Name: "feature/Payments"}, {Name: "feature/search"}}},
		{name: "no matches", search: "nope", expected: Branches{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, branches.Search(tt.search))
		})
	}
}
