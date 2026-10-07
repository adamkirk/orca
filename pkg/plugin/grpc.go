package plugin

import (
	"context"
	"errors"

	pluginv1 "github.com/adamkirk/orca/pkg/plugin/proto/v1"
	goplugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

// CommandProviderPlugin wires a CommandProvider into go-plugin over gRPC. Impl is
// only set on the plugin side.
type CommandProviderPlugin struct {
	goplugin.NetRPCUnsupportedPlugin

	Impl CommandProvider
}

func (p *CommandProviderPlugin) GRPCServer(_ *goplugin.GRPCBroker, s *grpc.Server) error {
	pluginv1.RegisterCommandProviderServer(s, &grpcServer{impl: p.Impl})

	return nil
}

func (p *CommandProviderPlugin) GRPCClient(_ context.Context, _ *goplugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &grpcClient{client: pluginv1.NewCommandProviderClient(c)}, nil
}

// grpcClient runs in orca and implements CommandProvider by calling the plugin.
type grpcClient struct {
	client pluginv1.CommandProviderClient
}

func (c *grpcClient) Commands() ([]CommandSpec, error) {
	resp, err := c.client.GetCommands(context.Background(), &pluginv1.GetCommandsRequest{})

	if err != nil {
		return nil, err
	}

	return commandSpecsFromProto(resp.GetCommands()), nil
}

func (c *grpcClient) Execute(ctx context.Context, req ExecuteRequest) (int, error) {
	resp, err := c.client.Execute(ctx, &pluginv1.ExecuteRequest{
		CommandPath: req.CommandPath,
		Args:        req.Args,
		Flags:       req.Flags,
	})

	if err != nil {
		return 1, err
	}

	if resp.GetError() != "" {
		return int(resp.GetExitCode()), errors.New(resp.GetError())
	}

	return int(resp.GetExitCode()), nil
}

// grpcServer runs in the plugin and forwards calls to the plugin's implementation.
type grpcServer struct {
	pluginv1.UnimplementedCommandProviderServer

	impl CommandProvider
}

func (s *grpcServer) GetCommands(_ context.Context, _ *pluginv1.GetCommandsRequest) (*pluginv1.GetCommandsResponse, error) {
	specs, err := s.impl.Commands()

	if err != nil {
		return nil, err
	}

	return &pluginv1.GetCommandsResponse{Commands: commandSpecsToProto(specs)}, nil
}

func (s *grpcServer) Execute(ctx context.Context, req *pluginv1.ExecuteRequest) (*pluginv1.ExecuteResponse, error) {
	code, err := s.impl.Execute(ctx, ExecuteRequest{
		CommandPath: req.GetCommandPath(),
		Args:        req.GetArgs(),
		Flags:       req.GetFlags(),
	})

	resp := &pluginv1.ExecuteResponse{ExitCode: int32(code)}

	if err != nil {
		resp.Error = err.Error()

		if resp.ExitCode == 0 {
			resp.ExitCode = 1
		}
	}

	return resp, nil
}

func commandSpecsToProto(specs []CommandSpec) []*pluginv1.CommandSpec {
	out := make([]*pluginv1.CommandSpec, len(specs))

	for i, s := range specs {
		flags := make([]*pluginv1.FlagSpec, len(s.Flags))

		for j, f := range s.Flags {
			flags[j] = &pluginv1.FlagSpec{
				Name:      f.Name,
				Shorthand: f.Shorthand,
				Usage:     f.Usage,
				Type:      flagTypeToProto(f.Type),
				Default:   f.Default,
			}
		}

		out[i] = &pluginv1.CommandSpec{
			Use:         s.Use,
			Short:       s.Short,
			Long:        s.Long,
			Flags:       flags,
			Subcommands: commandSpecsToProto(s.Subcommands),
		}
	}

	return out
}

func commandSpecsFromProto(specs []*pluginv1.CommandSpec) []CommandSpec {
	out := make([]CommandSpec, len(specs))

	for i, s := range specs {
		flags := make([]FlagSpec, len(s.GetFlags()))

		for j, f := range s.GetFlags() {
			flags[j] = FlagSpec{
				Name:      f.GetName(),
				Shorthand: f.GetShorthand(),
				Usage:     f.GetUsage(),
				Type:      flagTypeFromProto(f.GetType()),
				Default:   f.GetDefault(),
			}
		}

		out[i] = CommandSpec{
			Use:         s.GetUse(),
			Short:       s.GetShort(),
			Long:        s.GetLong(),
			Flags:       flags,
			Subcommands: commandSpecsFromProto(s.GetSubcommands()),
		}
	}

	return out
}

func flagTypeToProto(t FlagType) pluginv1.FlagType {
	if t == FlagBool {
		return pluginv1.FlagType_FLAG_TYPE_BOOL
	}

	return pluginv1.FlagType_FLAG_TYPE_STRING
}

func flagTypeFromProto(t pluginv1.FlagType) FlagType {
	if t == pluginv1.FlagType_FLAG_TYPE_BOOL {
		return FlagBool
	}

	return FlagString
}
