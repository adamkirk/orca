package sys

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"runtime"

	"github.com/Masterminds/semver/v3"
	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/usererr"
	"github.com/minio/selfupdate"
)

type VersioningStrategy string

const VersioningStrategyLatest VersioningStrategy = "latest"
const VersioningStrategySpecific VersioningStrategy = "specific"

func (m *Manager) replaceSelf(version string) error {
	os, err := m.getOrcaOSForDownload()

	if err != nil {
		return err
	}

	arch, err := m.getOrcaArchForDownload()

	if err != nil {
		return err
	}

	url := fmt.Sprintf(
		"https://github.com/%s/%s/releases/download/v%s/orca_%s_%s.tar.gz",
		orcaRepoOrg,
		orcaRepoName,
		version,
		os,
		arch,
	)

	contents, err := m.getExecutableFromArchiveUrl(url, "orca")

	if err != nil {
		return err
	}

	reader := bytes.NewReader(contents)

	return selfupdate.Apply(reader, selfupdate.Options{})
}

func (m *Manager) ensureSelfVersionExists(v *semver.Version) error {
	exists, err := m.github.VersionExists(orcaRepoOrg, orcaRepoName, fmt.Sprintf("v%s", v.String()))

	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("%w: %s", ErrVersionNotFound, v.String())
	}

	return nil
}

func (m *Manager) getVersionForSelf(cmd UpdateSelfCommand) (*semver.Version, error) {
	if cmd.Strategy == VersioningStrategySpecific {
		v, err := semver.NewVersion(cmd.SpecifiedVersion)

		if err != nil {
			return nil, err
		}

		if err := m.ensureSelfVersionExists(v); err != nil {
			return nil, err
		}

		return v, nil
	}

	return m.github.LatestVersion(orcaRepoOrg, orcaRepoName)
}

type UpdateSelfCommand struct {
	bus.Returns[bus.Void]

	Strategy         VersioningStrategy
	SpecifiedVersion string
}

func (UpdateSelfCommand) MessageName() string {
	return "sys.self-update"
}

func (m *Manager) UpdateSelf(_ context.Context, cmd UpdateSelfCommand) error {
	v, err := m.getVersionForSelf(cmd)

	if err != nil {
		return usererr.Wrap(err, "Failed to find version!")
	}

	m.tui.Info(fmt.Sprintf("Switching to version: %s", v.String()))

	err = m.replaceSelf(v.String())

	if err != nil {
		return usererr.Wrap(
			err,
			"Failed to apply update!",
		)
	}

	m.tui.Success("Successfully switched!")
	return nil
}

func (m *Manager) extractExecutableFromArchive(archive []byte, path string) (io.Reader, error) {
	buf := bytes.NewReader(archive)
	gzr, err := gzip.NewReader(buf)
	if err != nil {
		return nil, err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()

		switch {

		// if no more files are found return
		case err == io.EOF:
			return nil, ErrInvalidArchive

		// return any other error
		case err != nil:
			return nil, err

		// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			continue
		}

		if header.Name == path {
			return tr, nil
		}
	}
}

func (m *Manager) getExecutableFromArchiveUrl(url string, executableName string) ([]byte, error) {
	archive, err := m.getFileFromUrl(url)

	if err != nil {
		return nil, err
	}

	executable, err := m.extractExecutableFromArchive(archive, executableName)

	if err != nil {
		return nil, err
	}

	return io.ReadAll(executable)
}

func (m *Manager) getFileFromUrl(url string) ([]byte, error) {

	resp, err := http.Get(url)

	if err != nil {
		return nil, err
	}

	contents, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	if err := resp.Body.Close(); err != nil {
		return nil, err
	}

	return contents, nil
}

func (m *Manager) getOrcaOSForDownload() (string, error) {
	switch os := runtime.GOOS; os {
	case "darwin":
		return "Darwin", nil
	case "linux":
		return "Linux", nil
	case "windows":
		return "Windows", nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedOS, os)
	}
}

func (m *Manager) getOrcaArchForDownload() (string, error) {
	switch arch := runtime.GOARCH; arch {
	case "amd64":
		return "X86_64", nil
	case "arm64":
		return "arm64", nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedArchitecture, arch)
	}
}
