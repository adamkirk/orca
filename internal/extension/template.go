package extension

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"github.com/adamkirk/orca/internal/usererr"
	"github.com/adamkirk/orca/internal/workspace"
)

// templateProject is a project, as a template sees it.
type templateProject struct {
	name string
	dir  string
}

// Path is where the project is cloned. It fails for a project that isn't
// cloned, rather than giving an empty path.
func (p templateProject) Path() (string, error) {
	if p.dir == "" {
		return "", fmt.Errorf("%w: %s", ErrProjectNotCloned, p.name)
	}

	return p.dir, nil
}

// projectRef matches a reference to a project, e.g. ".Projects.svc-a". Go
// templates don't allow '-' in field names, which project names often have, so
// references are rewritten to call the project function instead.
var projectRef = regexp.MustCompile(`\.Projects\.([A-Za-z0-9_.-]+?)(\.Path\b)`)

// renderSource fills in a source path's references to the workspace's
// projects, e.g. "{{ .Projects.sdk.Path }}".
func renderSource(ws *workspace.Workspace, source string) (string, error) {
	if !strings.Contains(source, "{{") {
		return source, nil
	}

	projects := map[string]templateProject{}

	for _, p := range ws.Projects {
		dir := ""

		if p.IsRegistered {
			dir = p.ProjectDir
		}

		projects[p.Name] = templateProject{name: p.Name, dir: dir}
	}

	funcs := template.FuncMap{
		"project": func(name string) (templateProject, error) {
			p, ok := projects[name]

			if !ok {
				return templateProject{}, fmt.Errorf("%w: %s", workspace.ErrUnknownProject, name)
			}

			return p, nil
		},
	}

	rewritten := projectRef.ReplaceAllString(source, `(project "$1")$2`)
	tmpl, err := template.New("source").Funcs(funcs).Parse(rewritten)

	if err != nil {
		return "", usererr.Wrap(err, fmt.Sprintf("Invalid template in source '%s'!", source))
	}

	var out strings.Builder

	if err := tmpl.Execute(&out, nil); err != nil {
		if errors.Is(err, ErrProjectNotCloned) {
			return "", usererr.Wrap(err, fmt.Sprintf("Source '%s' refers to a project that isn't cloned; clone it with 'orca ws clone'.", source))
		}

		if errors.Is(err, workspace.ErrUnknownProject) {
			return "", usererr.Wrap(err, fmt.Sprintf("Source '%s' refers to a project that isn't in the workspace!", source))
		}

		return "", usererr.Wrap(err, fmt.Sprintf("Failed to fill in source '%s'!", source))
	}

	return out.String(), nil
}
