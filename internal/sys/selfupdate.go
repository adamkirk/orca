package sys

import (
	"bytes"
	"context"
	"fmt"

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
