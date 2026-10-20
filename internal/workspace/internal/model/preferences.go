package model

type ProjectPreferences struct {
	// Nil when it isn't set, which means the project is enabled
	Enabled *bool
}

type Preferences struct {
	Projects map[string]ProjectPreferences
}
