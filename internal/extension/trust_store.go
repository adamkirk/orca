package extension

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

// TrustKey is an extension, by where it's defined: the workspace config, or a
// project's config.
type TrustKey struct {
	Workspace string

	// Empty for an extension defined in the workspace config
	Project   string
	Extension string
}

// Trusted is what was trusted for an extension.
type Trusted struct {
	// Changes whenever the extension's config does
	Fingerprint string `yaml:"fingerprint"`

	// What it did, to show what's changed when it's asked about again
	Summary string `yaml:"summary"`
}

type trustedWorkspace struct {
	Workspace map[string]Trusted            `yaml:"workspace,omitempty"`
	Projects  map[string]map[string]Trusted `yaml:"projects,omitempty"`
}

type trustFile struct {
	Workspaces map[string]trustedWorkspace `yaml:"workspaces"`
}

// TrustStore records which extensions the developer has trusted to run. It's
// kept apart from the user config, as it's written often, and isn't meant to be
// edited.
type TrustStore struct {
	fs   afero.Fs
	path string

	// Read once, when first needed
	file *trustFile
}

func NewTrustStore(fs afero.Fs, path string) *TrustStore {
	return &TrustStore{
		fs:   fs,
		path: path,
	}
}

func (s *TrustStore) load() (*trustFile, error) {
	if s.file != nil {
		return s.file, nil
	}

	file := &trustFile{}
	data, err := afero.ReadFile(s.fs, s.path)

	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("reading %s: %w", s.path, err)
	}

	if err == nil {
		if err := yaml.Unmarshal(data, file); err != nil {
			return nil, fmt.Errorf("decoding %s: %w", s.path, err)
		}
	}

	if file.Workspaces == nil {
		file.Workspaces = map[string]trustedWorkspace{}
	}

	s.file = file

	return file, nil
}

// Get returns what was trusted for the extension, if it ever was.
func (s *TrustStore) Get(key TrustKey) (Trusted, bool, error) {
	file, err := s.load()

	if err != nil {
		return Trusted{}, false, err
	}

	ws := file.Workspaces[key.Workspace]
	exts := ws.Workspace

	if key.Project != "" {
		exts = ws.Projects[key.Project]
	}

	trusted, ok := exts[key.Extension]

	return trusted, ok, nil
}

// Trust records the extensions as trusted, replacing anything recorded for
// them before.
func (s *TrustStore) Trust(trusted map[TrustKey]Trusted) error {
	file, err := s.load()

	if err != nil {
		return err
	}

	for key, t := range trusted {
		ws := file.Workspaces[key.Workspace]

		if key.Project == "" {
			if ws.Workspace == nil {
				ws.Workspace = map[string]Trusted{}
			}

			ws.Workspace[key.Extension] = t
		} else {
			if ws.Projects == nil {
				ws.Projects = map[string]map[string]Trusted{}
			}

			if ws.Projects[key.Project] == nil {
				ws.Projects[key.Project] = map[string]Trusted{}
			}

			ws.Projects[key.Project][key.Extension] = t
		}

		file.Workspaces[key.Workspace] = ws
	}

	return s.save(file)
}

func (s *TrustStore) save(file *trustFile) error {
	data, err := yaml.Marshal(file)

	if err != nil {
		return fmt.Errorf("encoding %s: %w", s.path, err)
	}

	if err := s.fs.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return fmt.Errorf("creating the directory for %s: %w", s.path, err)
	}

	// Written in full, then moved into place, so it's never left half written
	tmp := fmt.Sprintf("%s.%d.tmp", s.path, os.Getpid())

	if err := afero.WriteFile(s.fs, tmp, data, 0600); err != nil {
		return fmt.Errorf("writing %s: %w", s.path, err)
	}

	if err := s.fs.Rename(tmp, s.path); err != nil {
		_ = s.fs.Remove(tmp)

		return fmt.Errorf("writing %s: %w", s.path, err)
	}

	return nil
}
