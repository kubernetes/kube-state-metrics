/*
Copyright 2022 The Kubernetes Authors All rights reserved.

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

package options

import (
	"os"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

const autoloadZsh = `Generate the autocompletion script for the zsh shell.

If shell completion is not already enabled in your environment you will need
to enable it.  You can execute the following once:

        echo "autoload -U compinit; compinit" >> ~/.zshrc

To load completions in your current shell session:

        source <(kube-state-metrics completion zsh); compdef _kube-state-metrics kube-state-metrics

To load completions for every new session, execute once:

#### Linux:

        kube-state-metrics completion zsh > "${fpath[1]}/_kube-state-metrics"

#### macOS:

        kube-state-metrics completion zsh > $(brew --prefix)/share/zsh/site-functions/_kube-state-metrics

You will need to start a new shell for this setup to take effect.

Usage:
  kube-state-metrics completion zsh [flags]

Flags:
  --no-descriptions   disable completion descriptions
`
const autoloadBash = `Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:

        source <(kube-state-metrics completion bash)

To load completions for every new session, execute once:

#### Linux:

        kube-state-metrics completion bash > /etc/bash_completion.d/kube-state-metrics

#### macOS:

        kube-state-metrics completion bash > $(brew --prefix)/etc/bash_completion.d/kube-state-metrics

You will need to start a new shell for this setup to take effect.

Usage:
  kube-state-metrics completion bash

Flags:
  --no-descriptions   disable completion descriptions
`

const autoloadFish = `Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

        kube-state-metrics completion fish | source

To load completions for every new session, execute once:

        kube-state-metrics completion fish > ~/.config/fish/completions/kube-state-metrics.fish

You will need to start a new shell for this setup to take effect.

Usage:
  kube-state-metrics completion fish [flags]

Flags:
  --no-descriptions   disable completion descriptions
`

// FetchLoadInstructions returns instructions for enabling autocompletion for a particular shell.
func FetchLoadInstructions(shell string) string {
	switch shell {
	case "zsh":
		return autoloadZsh
	case "bash":
		return autoloadBash
	case "fish":
		return autoloadFish
	default:
		return ""
	}
}

var completionCommand = &cobra.Command{
	Use:                   "completion [bash|zsh|fish]",
	Short:                 "Generate completion script for kube-state-metrics.",
	DisableFlagsInUseLine: true,
	Aliases:               []string{"comp", "c"},
	ValidArgs:             []string{"bash", "zsh", "fish"},
	Args:                  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			_ = cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			_ = cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			_ = cmd.Root().GenFishCompletion(os.Stdout, true)
		}
		klog.FlushAndExit(klog.ExitFlushTimeout, 0)
	},
	Example: "kube-state-metrics completion bash > /tmp/kube-state-metrics.bash && source /tmp/kube-state-metrics.bash # for shells compatible with bash",
}

// InitCommand defines the root command that others will latch onto.
var InitCommand = &cobra.Command{
	Use:   "kube-state-metrics",
	Short: "Add-on agent to generate and expose cluster-level metrics.",
	Long:  "kube-state-metrics is a simple service that listens to the Kubernetes API server and generates metrics about the state of the objects.",
	Args:  cobra.NoArgs,
}
