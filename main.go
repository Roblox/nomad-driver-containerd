/*
Copyright 2020 Roblox Corporation

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0


Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	log "github.com/hashicorp/go-hclog"

	"github.com/Roblox/nomad-driver-containerd/containerd"

	"github.com/hashicorp/nomad/plugins"

	"github.com/spf13/cobra"
)

func main() {
	var cmd = &cobra.Command{
		Use:     containerd.PluginName,
		Short:   "Nomad task driver for launching containers using containerd",
		Version: containerd.PluginVersion,
		Run: func(cmd *cobra.Command, args []string) {
			// Serve the plugin
			plugins.Serve(factory)
		},
	}
	cmd.SetVersionTemplate("{{.Use}} {{.Version}}\n")
	cmd.Execute()
}

// factory returns a new instance of a nomad driver plugin
func factory(log log.Logger) interface{} {
	return containerd.NewPlugin(log)
}
