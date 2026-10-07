package plugin

import goplugin "github.com/hashicorp/go-plugin"

// CommandsPluginName is the name the CommandProvider is dispensed under.
const CommandsPluginName = "commands"

// Handshake must match between orca and its plugins. Bump ProtocolVersion on any
// breaking change to the gRPC protocol.
var Handshake = goplugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "ORCA_PLUGIN",
	MagicCookieValue: "4b0a1d6e-2f53-4c4e-9a52-7a1f6c1d2e83",
}

// PluginMap is the set of plugin types orca can dispense from a plugin binary.
var PluginMap = map[string]goplugin.Plugin{
	CommandsPluginName: &CommandProviderPlugin{},
}
