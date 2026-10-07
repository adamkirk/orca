// An example orca plugin that adds a `hello` command.
//
// Build it into orca's plugin directory with `make build-example-plugins`, then
// run `orca hello` or `orca hello --name you`.
package main

import (
	"context"
	"fmt"

	"github.com/adamkirk/orca/pkg/plugin"
)

func main() {
	plugin.Serve(&hello{})
}

type hello struct{}

func (h *hello) Commands() ([]plugin.CommandSpec, error) {
	return []plugin.CommandSpec{
		{
			Use:   "hello",
			Short: "Says hello world.",
			Flags: []plugin.FlagSpec{
				{
					Name:      "name",
					Shorthand: "n",
					Usage:     "Who to say hello to.",
					Type:      plugin.FlagString,
					Default:   "world",
				},
			},
		},
	}, nil
}

func (h *hello) Execute(_ context.Context, req plugin.ExecuteRequest) (int, error) {
	fmt.Printf("hello %s\n", req.Flags["name"])

	return 0, nil
}
