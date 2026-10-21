package workspace

import (
	"testing"

	"github.com/adamkirk/orca/internal/workspace/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func parseExtensions(t *testing.T, contents string) []model.Extension {
	var cfg struct {
		Extensions []model.Extension
	}

	require.NoError(t, yaml.Unmarshal([]byte(contents), &cfg))

	return cfg.Extensions
}

func Test_buildExtensions(t *testing.T) {
	tests := []struct {
		name         string
		config       string
		want         []Extension
		wantProblems []string
	}{
		{
			name: "a command, which is the default type",
			config: `
extensions:
  - name: artisan
    service: app
    command: php artisan
    defaultArgs: [list]
    chdir: src
    description: Run an artisan command
  - name: install
    type: command
    command: npm ci
`,
			want: []Extension{
				{Name: "artisan", Description: "Run an artisan command", Spec: CommandSpec{Command: "php artisan", Service: "app", DefaultArgs: []string{"list"}, Chdir: "src"}},
				{Name: "install", Spec: CommandSpec{Command: "npm ci"}},
			},
		},
		{
			name: "a syncFile",
			config: `
extensions:
  - name: env
    type: syncFile
    source: .env.example
    target: .env
`,
			want: []Extension{
				{Name: "env", Spec: SyncFileSpec{Source: ".env.example", Target: ".env"}},
			},
		},
		{
			name: "a symlink",
			config: `
extensions:
  - name: link-sdk
    type: symlink
    source: "{{ .Projects.sdk.Path }}"
    target: node_modules/@acme/sdk
`,
			want: []Extension{
				{Name: "link-sdk", Spec: SymlinkSpec{Source: "{{ .Projects.sdk.Path }}", Target: "node_modules/@acme/sdk"}},
			},
		},
		{
			name:   "no extensions",
			config: `extensions: []`,
			want:   []Extension{},
		},
		{
			name: "fields that don't belong to the type",
			config: `
extensions:
  - name: env
    type: syncFile
    source: .env.example
    target: .env
    service: app
  - name: install
    command: npm ci
    target: node_modules
`,
			wantProblems: []string{
				"extension 'env': 'service' can't be used with type 'syncFile' (it takes: source, target)",
				"extension 'install': 'target' can't be used with type 'command' (it takes: command, service, defaultArgs, chdir)",
			},
		},
		{
			name: "missing required fields",
			config: `
extensions:
  - name: env
    type: syncFile
    source: .env.example
  - name: install
    command:
  - command: npm ci
`,
			wantProblems: []string{
				"extension 'env': 'target' is required for type 'syncFile'",
				"extension 'install': 'command' is required for type 'command'",
				"extension at index 2: 'name' is required",
			},
		},
		{
			name: "an unknown type",
			config: `
extensions:
  - name: env
    type: copy
    src: .env.example
`,
			wantProblems: []string{
				"extension 'env': unknown type 'copy' (expected command, symlink, syncFile)",
			},
		},
		{
			name: "a name defined twice",
			config: `
extensions:
  - name: install
    command: npm ci
  - name: install
    command: yarn
`,
			wantProblems: []string{
				"extension 'install': defined more than once",
			},
		},
		{
			name: "fields of the wrong shape",
			config: `
extensions:
  - name: install
    command: npm ci
    defaultArgs: ci
`,
			wantProblems: []string{
				"extension 'install': invalid fields for type 'command': yaml: unmarshal errors:\n  line 5: cannot unmarshal !!str `ci` into []string",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, problems := buildExtensions(parseExtensions(t, tt.config))

			if len(tt.wantProblems) > 0 {
				assert.Equal(t, tt.wantProblems, problems)
				return
			}

			assert.Empty(t, problems)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_extensionKinds(t *testing.T) {
	specs := map[ExtensionType]ExtensionSpec{
		ExtensionTypeCommand:  CommandSpec{},
		ExtensionTypeSyncFile: SyncFileSpec{},
		ExtensionTypeSymlink:  SymlinkSpec{},
	}

	assert.Len(t, extensionKinds, len(specs), "every type needs a spec, and every spec a kind")

	for name, kind := range extensionKinds {
		t.Run(string(name), func(t *testing.T) {
			spec, ok := specs[name]

			require.True(t, ok, "no spec for the type")
			assert.Equal(t, name, spec.Type())
			assert.NotEmpty(t, kind.fields)
		})
	}
}
