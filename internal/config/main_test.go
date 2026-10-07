package config

import (
	"testing"

	"github.com/adamkirk/orca/internal/common"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const configFilePath = "/orca.yaml"
const expectedDefault = `logging:
    level: none
    format: text
workspaces: []
currentWorkspace: ""
`

const existingConfigFileContents = `logging:
    level: debug
    format: json
workspaces: []
currentWorkspace: blah
`

const existingWithMultipleWorkspaces = `logging:
    level: debug
    format: json
workspaces:
    - name: meh
      path: /path/meh
    - name: blah
      path: /path/blah
currentWorkspace: blah
`

const existingWithMultipleWorkspacesAndProjects = `logging:
    level: debug
    format: json
workspaces:
    - name: meh
      path: /path/meh
      projects:
        - name: meh1
          path: /projects/meh/1
        - name: meh2
          path: /projects/meh/2
    - name: blah
      path: /path/blah
      projects:
        - name: blah1
          path: /projects/blah/1
        - name: blah2
          path: /projects/blah/2
currentWorkspace: blah
`

func Test_ConfigIsCreatedIfNotExists(t *testing.T) {
	fs := afero.NewMemMapFs()
	cfg := NewDefaultConfig(
		fs,
		configFilePath,
	)

	err := cfg.LoadOrCreate()

	assert.Nil(t, err)
	exists, err := afero.Exists(fs, configFilePath)
	require.Nil(t, err)
	assert.True(t, exists)

	contents, err := afero.ReadFile(fs, configFilePath)
	require.Nil(t, err)
	assert.Equal(t, expectedDefault, string(contents))
}

func Test_ConfigIsLoadedIfExists(t *testing.T) {
	useExistingConfig(t)
}

func useExistingConfig(t *testing.T) (afero.Fs, *Config) {
	fs := afero.NewMemMapFs()
	err := afero.WriteFile(fs, configFilePath, []byte(existingConfigFileContents), 0755)
	require.Nil(t, err)
	cfg := NewDefaultConfig(
		fs,
		configFilePath,
	)

	err = cfg.LoadOrCreate()

	assert.Nil(t, err)
	exists, err := afero.Exists(fs, configFilePath)
	require.Nil(t, err)
	assert.True(t, exists)

	expected := config{
		Logging: configLogging{
			Level:  "debug",
			Format: "json",
		},
		CurrentWorkspace: "blah",
		Workspaces:       []configWorkspace{},
	}
	assert.Equal(t, expected, *cfg.persisted)
	assert.Equal(t, expected, *cfg.runtimeConfig)

	return fs, cfg
}

func useAddWorkspace(t *testing.T, fs afero.Fs, cfg *Config) {
	err := cfg.AddWorkspace("/path/meh", "meh")
	assertInternalConfigsAreDifferent(t, cfg)

	require.Nil(t, err)

	expected := config{
		Logging: configLogging{
			Level:  "debug",
			Format: "json",
		},
		CurrentWorkspace: "blah",
		Workspaces: []configWorkspace{
			{
				Name: "meh",
				Path: "/path/meh",
			},
		},
	}
	assert.Equal(t, expected, *cfg.persisted)
	assert.Equal(t, expected, *cfg.runtimeConfig)

	contents, err := afero.ReadFile(fs, configFilePath)
	expectedContents := `logging:
    level: debug
    format: json
workspaces:
    - name: meh
      path: /path/meh
      projects: []
currentWorkspace: blah
`
	require.Nil(t, err)
	assert.Equal(t, expectedContents, string(contents))

	err = cfg.AddWorkspace("/other/path/meh", "meh")

	assert.Equal(t, common.ErrWorkspaceAlreadyExists{
		Name: "meh",
	}, err)
}

func Test_AddWorkspace(t *testing.T) {
	fs, cfg := useExistingConfig(t)
	useAddWorkspace(t, fs, cfg)
}

func Test_SwitchWorkspace(t *testing.T) {
	fs, cfg := useExistingConfig(t)
	useAddWorkspace(t, fs, cfg)

	err := cfg.SwitchWorkspace("meh")
	require.Nil(t, err)

	contents, err := afero.ReadFile(fs, configFilePath)
	expectedContents := `logging:
    level: debug
    format: json
workspaces:
    - name: meh
      path: /path/meh
      projects: []
currentWorkspace: meh
`
	require.Nil(t, err)
	assert.Equal(t, expectedContents, string(contents))

	err = cfg.SwitchWorkspace("missing")
	require.Equal(t, common.ErrUnknownWorkspace{
		Name: "missing",
	}, err)
	assertInternalConfigsAreDifferent(t, cfg)
}

func Test_GetAllWorkspaceMeta(t *testing.T) {
	fs := afero.NewMemMapFs()
	err := afero.WriteFile(fs, configFilePath, []byte(existingWithMultipleWorkspaces), 0755)
	require.Nil(t, err)
	cfg := NewDefaultConfig(
		fs,
		configFilePath,
	)

	err = cfg.LoadOrCreate()
	require.Nil(t, err)

	locations := cfg.GetAllWorkspaceMeta()

	assert.Equal(t, []common.WorkspaceMeta{
		{
			Name: "meh",
			Path: "/path/meh",
		},
		{
			Name: "blah",
			Path: "/path/blah",
		},
	}, locations)
}

func Test_GetWorkspaceMeta(t *testing.T) {
	fs := afero.NewMemMapFs()
	err := afero.WriteFile(fs, configFilePath, []byte(existingWithMultipleWorkspaces), 0755)
	require.Nil(t, err)
	cfg := NewDefaultConfig(
		fs,
		configFilePath,
	)

	err = cfg.LoadOrCreate()
	require.Nil(t, err)

	location, err := cfg.GetWorkspaceMeta("blah")

	require.Nil(t, err)
	assert.Equal(t, common.WorkspaceMeta{
		Name: "blah",
		Path: "/path/blah",
	}, location)

	_, err = cfg.GetWorkspaceMeta("missing")
	require.Equal(t, common.ErrUnknownWorkspace{
		Name: "missing",
	}, err)
}

func Test_LoggingGetters(t *testing.T) {
	_, cfg := useExistingConfig(t)

	assert.Equal(t, "debug", cfg.GetLoggingLevel())
	assert.Equal(t, "json", cfg.GetLoggingFormat())
}

func Test_GetPluginDirs(t *testing.T) {
	_, cfg := useExistingConfig(t)

	assert.Empty(t, cfg.GetPluginDirs())

	fs := afero.NewMemMapFs()
	err := afero.WriteFile(fs, configFilePath, []byte(existingConfigFileContents+`plugins:
    dirs:
        - /plugins/a
        - /plugins/b
`), 0755)
	require.Nil(t, err)

	cfg = NewDefaultConfig(fs, configFilePath)
	require.Nil(t, cfg.LoadOrCreate())

	assert.Equal(t, []string{"/plugins/a", "/plugins/b"}, cfg.GetPluginDirs())
}

// Used to ensure that during loading or any mutations we do not end up referencing
// the same struct. Should be called after any mutation functions throughout the
// tests.
func assertInternalConfigsAreDifferent(t *testing.T, cfg *Config) {
	assert.NotSame(t, cfg.runtimeConfig, cfg.persisted)
}

// Ensures that the 'runtimeConfig' and 'persisted' config are in fact two different
// configs
func Test_InternalConfigsAreDifferent(t *testing.T) {
	// Test when it's loaded from a file
	_, cfg := useExistingConfig(t)

	assertInternalConfigsAreDifferent(t, cfg)

	// Test when it's created for the first time
	fs := afero.NewMemMapFs()
	cfg = NewDefaultConfig(
		fs,
		configFilePath,
	)

	err := cfg.LoadOrCreate()
	require.Nil(t, err)

	assertInternalConfigsAreDifferent(t, cfg)
}

func Test_GetRuntimeConfig(t *testing.T) {
	// Test when it's loaded from a file
	_, cfg := useExistingConfig(t)

	assertInternalConfigsAreDifferent(t, cfg)
	assert.Same(t, cfg.runtimeConfig, cfg.GetRuntimeConfig())
}

func Test_ProjectExists(t *testing.T) {
	fs := afero.NewMemMapFs()
	err := afero.WriteFile(fs, configFilePath, []byte(existingWithMultipleWorkspacesAndProjects), 0755)
	require.Nil(t, err)
	cfg := NewDefaultConfig(
		fs,
		configFilePath,
	)

	err = cfg.LoadOrCreate()
	require.Nil(t, err)

	exists, err := cfg.ProjectExists("meh", "meh1")

	require.Nil(t, err)
	assert.True(t, exists)

	exists, err = cfg.ProjectExists("meh", "blah1")

	require.Nil(t, err)
	assert.False(t, exists)

	exists, err = cfg.ProjectExists("blah", "blah2")

	require.Nil(t, err)
	assert.True(t, exists)

	exists, err = cfg.ProjectExists("blah", "missing")

	require.Nil(t, err)
	assert.False(t, exists)

	exists, err = cfg.ProjectExists("missing", "blah")

	require.Equal(t, common.ErrUnknownWorkspace{
		Name: "missing",
	}, err)
	assert.False(t, exists)
}

func Test_GetAllProjectMeta(t *testing.T) {
	fs := afero.NewMemMapFs()
	err := afero.WriteFile(fs, configFilePath, []byte(existingWithMultipleWorkspacesAndProjects), 0755)
	require.Nil(t, err)
	cfg := NewDefaultConfig(
		fs,
		configFilePath,
	)

	err = cfg.LoadOrCreate()
	require.Nil(t, err)

	res := cfg.GetAllProjectMeta()

	assert.Equal(t, []common.ProjectMeta{
		{
			Name:          "meh1",
			WorkspaceName: "meh",
			Path:          "/projects/meh/1",
		},
		{
			Name:          "meh2",
			WorkspaceName: "meh",
			Path:          "/projects/meh/2",
		},
		{
			Name:          "blah1",
			WorkspaceName: "blah",
			Path:          "/projects/blah/1",
		},
		{
			Name:          "blah2",
			WorkspaceName: "blah",
			Path:          "/projects/blah/2",
		},
	}, res)
}

func Test_SetProjectPath(t *testing.T) {
	tests := []struct {
		name             string
		initialFile      string
		wsName           string
		projectName      string
		projectPath      string
		expect           *config
		expectErr        error
		expectFileOnDisk string
	}{
		{
			name:        "when a workspace exists with no projects",
			initialFile: existingWithMultipleWorkspaces,
			wsName:      "meh",
			projectName: "meh1",
			projectPath: "/path/meh/1",
			expect: &config{
				Logging: configLogging{
					Level:  "debug",
					Format: "json",
				},
				Workspaces: []configWorkspace{
					{
						Name: "meh",
						Path: "/path/meh",
						Projects: []configProject{
							{
								Name: "meh1",
								Path: "/path/meh/1",
							},
						},
					},
					{
						Name: "blah",
						Path: "/path/blah",
					},
				},
				CurrentWorkspace: "blah",
			},
			expectErr: nil,
			expectFileOnDisk: `logging:
    level: debug
    format: json
workspaces:
    - name: meh
      path: /path/meh
      projects:
        - name: meh1
          path: /path/meh/1
    - name: blah
      path: /path/blah
      projects: []
currentWorkspace: blah
`,
		},

		{
			name:        "updating path for a project",
			initialFile: existingWithMultipleWorkspaces,
			wsName:      "meh",
			projectName: "meh1",
			projectPath: "/other/path/meh/1",
			expect: &config{
				Logging: configLogging{
					Level:  "debug",
					Format: "json",
				},
				Workspaces: []configWorkspace{
					{
						Name: "meh",
						Path: "/path/meh",
						Projects: []configProject{
							{
								Name: "meh1",
								Path: "/other/path/meh/1",
							},
						},
					},
					{
						Name: "blah",
						Path: "/path/blah",
					},
				},
				CurrentWorkspace: "blah",
			},
			expectErr: nil,
			expectFileOnDisk: `logging:
    level: debug
    format: json
workspaces:
    - name: meh
      path: /path/meh
      projects:
        - name: meh1
          path: /other/path/meh/1
    - name: blah
      path: /path/blah
      projects: []
currentWorkspace: blah
`,
		},

		{
			name:        "updating path for project where many projects exist",
			initialFile: existingWithMultipleWorkspacesAndProjects,
			wsName:      "blah",
			projectName: "blah2",
			projectPath: "/other/path/blah/2",
			expect: &config{
				Logging: configLogging{
					Level:  "debug",
					Format: "json",
				},
				Workspaces: []configWorkspace{
					{
						Name: "meh",
						Path: "/path/meh",
						Projects: []configProject{
							{
								Name: "meh1",
								Path: "/projects/meh/1",
							},
							{
								Name: "meh2",
								Path: "/projects/meh/2",
							},
						},
					},
					{
						Name: "blah",
						Path: "/path/blah",
						Projects: []configProject{
							{
								Name: "blah1",
								Path: "/projects/blah/1",
							},
							{
								Name: "blah2",
								Path: "/other/path/blah/2",
							},
						},
					},
				},
				CurrentWorkspace: "blah",
			},
			expectErr: nil,
			expectFileOnDisk: `logging:
    level: debug
    format: json
workspaces:
    - name: meh
      path: /path/meh
      projects:
        - name: meh1
          path: /projects/meh/1
        - name: meh2
          path: /projects/meh/2
    - name: blah
      path: /path/blah
      projects:
        - name: blah1
          path: /projects/blah/1
        - name: blah2
          path: /other/path/blah/2
currentWorkspace: blah
`,
		},

		{
			name:        "updating a project in a workspace that doesn't exist",
			initialFile: existingWithMultipleWorkspacesAndProjects,
			wsName:      "missing",
			projectName: "blah2",
			projectPath: "/other/path/blah/2",
			expect:      nil,
			expectErr: common.ErrUnknownWorkspace{
				Name: "missing",
			},
			expectFileOnDisk: existingWithMultipleWorkspacesAndProjects,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			fs := afero.NewMemMapFs()
			err := afero.WriteFile(fs, configFilePath, []byte(test.initialFile), 0755)
			require.Nil(tt, err)
			cfg := NewDefaultConfig(
				fs,
				configFilePath,
			)

			err = cfg.LoadOrCreate()
			require.Nil(tt, err)

			err = cfg.SetProjectPath(test.wsName, test.projectName, test.projectPath)

			require.Equal(tt, test.expectErr, err)

			if test.expect != nil {
				assert.Equal(tt, test.expect, cfg.runtimeConfig)
				assert.Equal(tt, test.expect, cfg.persisted)
				assertInternalConfigsAreDifferent(tt, cfg)
			}

			contents, err := afero.ReadFile(fs, configFilePath)
			require.Nil(t, err)
			assert.Equal(t, test.expectFileOnDisk, string(contents))
		})
	}
}
