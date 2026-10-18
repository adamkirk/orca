package hostsys

import (
	"bytes"
	"context"
	"fmt"

	"github.com/Masterminds/semver/v3"
	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/common"
	"github.com/minio/selfupdate"
)

type VersioningStrategy string

const VersioningStrategyLatest VersioningStrategy = "latest"
const VersioningStrategySpecific VersioningStrategy = "specific"

func (hs *HostSystem) replaceSelf(version string) error {
	os, err := hs.getOrcaOSForDownload()

	if err != nil {
		return err
	}

	arch, err := hs.getOrcaArchForDownload()

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

	contents, err := hs.getExecutableFromArchiveUrl(url, "orca")

	if err != nil {
		return err
	}

	reader := bytes.NewReader(contents)

	return selfupdate.Apply(reader, selfupdate.Options{})
}

func (hs *HostSystem) ensureSelfVersionExists(v *semver.Version) error {
	exists, err := hs.github.VersionExists(orcaRepoOrg, orcaRepoName, fmt.Sprintf("v%s", v.String()))

	if err != nil {
		return err
	}

	if !exists {
		return common.ErrVersionNotFound{
			Version: v.String(),
		}
	}

	return nil
}

func (hs *HostSystem) getVersionForSelf(cmd UpdateSelfCommand) (*semver.Version, error) {
	if cmd.Strategy == VersioningStrategySpecific {
		v, err := semver.NewVersion(cmd.SpecifiedVersion)

		if err != nil {
			return nil, err
		}

		if err := hs.ensureSelfVersionExists(v); err != nil {
			return nil, err
		}

		return v, nil
	}

	return hs.github.LatestVersion(orcaRepoOrg, orcaRepoName)
}

type UpdateSelfCommand struct {
	bus.Returns[bus.Void]

	Strategy         VersioningStrategy
	SpecifiedVersion string
}

func (UpdateSelfCommand) MessageName() string {
	return "sys.self-update"
}

func (hs *HostSystem) UpdateSelf(_ context.Context, cmd UpdateSelfCommand) error {
	v, err := hs.getVersionForSelf(cmd)

	if err != nil {
		return hs.tui.RecordIfError("Failed to find version!", err)
	}

	hs.tui.Info(fmt.Sprintf("Switching to version: %s", v.String()))

	err = hs.replaceSelf(v.String())

	if err != nil {
		return hs.tui.RecordIfError(
			"Failed to apply update!",
			err,
		)
	}

	hs.tui.Success("Successfully switched!")
	return nil
}
