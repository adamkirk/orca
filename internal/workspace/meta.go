package workspace

type ProjectMeta struct {
	Name          string
	Path          string
	WorkspaceName string

	// Type is recorded when the project is registered, so it's known without
	// loading the workspace config. Empty for projects registered before types
	// existed, which are services.
	Type ProjectType
}

type WorkspaceMeta struct {
	Name     string
	Path     string
	Projects []ProjectMeta
}
