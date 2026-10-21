package extension

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	orcatui "github.com/adamkirk/orca/internal/tui"
	"github.com/adamkirk/orca/internal/usererr"
	"github.com/adamkirk/orca/internal/workspace"
	"gopkg.in/yaml.v3"
)

// planned is an extension that's about to run, in the project that defines
// it, and the hook that runs it, if any.
type planned struct {
	// Empty when it's run by hand, with ext
	hook   string
	target target
	ext    workspace.Extension
}

func (p planned) trustKey() TrustKey {
	return TrustKey{Workspace: p.target.ws.Name, Project: p.target.project.Name, Extension: p.ext.Name}
}

// fingerprint changes whenever what the extension does changes. Its name and
// description aren't included, as changing them doesn't change what it does.
func fingerprint(ext workspace.Extension) string {
	data, err := yaml.Marshal(struct {
		Type workspace.ExtensionType
		Spec workspace.ExtensionSpec
	}{ext.Spec.Type(), ext.Spec})

	// The specs are plain structs, which always marshal
	if err != nil {
		panic(fmt.Sprintf("fingerprinting extension '%s': %s", ext.Name, err))
	}

	sum := sha256.Sum256(data)

	return hex.EncodeToString(sum[:])
}

// summary is what the extension does, in one line, as its config says it.
func summary(ext workspace.Extension) string {
	switch spec := ext.Spec.(type) {
	case workspace.CommandSpec:
		s := strings.Join(append([]string{spec.Command}, spec.DefaultArgs...), " ")

		if spec.Service != "" {
			s += fmt.Sprintf(" (in service '%s')", spec.Service)
		}

		if spec.Chdir != "" {
			s += fmt.Sprintf(" (in %s)", spec.Chdir)
		}

		return s
	case workspace.SyncFileSpec:
		return fmt.Sprintf("%s → %s", spec.Source, spec.Target)
	case workspace.SymlinkSpec:
		return fmt.Sprintf("%s → %s", spec.Source, spec.Target)
	default:
		panic(fmt.Sprintf("unknown extension spec %T", spec))
	}
}

func trusted(ext workspace.Extension) Trusted {
	return Trusted{Fingerprint: fingerprint(ext), Summary: summary(ext)}
}

// untrusted is an extension that needs asking about.
type untrusted struct {
	run planned

	// What was trusted before it changed, if it ever was
	was     Trusted
	changed bool
}

// confirm asks the developer to trust any of the extensions that are new, or
// have changed since they were trusted, before any of them run. Each is only
// asked about once, however many times it's planned. With yes, nothing is asked,
// and nothing is recorded.
func (m *Manager) confirm(runs []planned, yes bool) error {
	pending, err := m.untrusted(runs)

	if err != nil {
		return err
	}

	if len(pending) == 0 || yes {
		return nil
	}

	m.showUntrusted(pending)

	ok, err := m.tui.Confirm("Trust these and continue?")

	if errors.Is(err, orcatui.ErrNotInteractive) {
		return usererr.Wrap(
			fmt.Errorf("%w: %w", ErrTrustNeeded, err),
			"These extensions need trusting before they run, but orca can't ask here. Pass --yes to run them anyway.",
		)
	}

	if err != nil {
		return err
	}

	if !ok {
		return usererr.Wrap(ErrTrustDeclined, "Nothing was run, as the extensions weren't trusted.")
	}

	records := map[TrustKey]Trusted{}

	for _, u := range pending {
		records[u.run.trustKey()] = trusted(u.run.ext)
	}

	return usererr.Wrap(m.trust.Trust(records), "Failed to record the trusted extensions!")
}

// untrusted returns the runs whose extensions need asking about, once each, in
// the order they're first planned.
func (m *Manager) untrusted(runs []planned) ([]untrusted, error) {
	pending := []untrusted{}
	seen := map[TrustKey]bool{}

	for _, run := range runs {
		key := run.trustKey()

		if seen[key] {
			continue
		}

		seen[key] = true

		was, ok, err := m.trust.Get(key)

		if err != nil {
			return nil, usererr.Wrap(err, "Failed to read the trusted extensions!")
		}

		if ok && was.Fingerprint == fingerprint(run.ext) {
			continue
		}

		pending = append(pending, untrusted{run: run, was: was, changed: ok})
	}

	return pending, nil
}

func (m *Manager) showUntrusted(pending []untrusted) {
	allChanged := true
	rows := make([][]string, len(pending))

	for i, u := range pending {
		allChanged = allChanged && u.changed
		what := summary(u.run.ext)

		if u.changed && u.was.Summary != what {
			what += fmt.Sprintf("   (was: %s)", u.was.Summary)
		}

		rows[i] = []string{u.run.target.project.Name, u.run.hook, u.run.ext.Name, string(u.run.ext.Spec.Type()), what}
	}

	if allChanged {
		m.tui.Info("", "These extensions have changed since you trusted them:", "")
	} else {
		m.tui.Info("", "This workspace wants to run the following on your machine:", "")
	}

	m.tui.Table([]string{"Project", "Hook", "Extension", "Type", "Runs"}, rows)
}
