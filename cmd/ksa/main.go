// Copyright © 2024 KubeStack-AI Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This is the main entrypoint for the KubeStack-AI (ksa) command-line application.
package main

import (
	"github.com/kubestack-ai/kubestack-ai/internal/cli/commands"
)

func main() {
	// The Execute function from the commands package handles all command parsing,
	// flag handling, and execution logic.
	commands.Execute()
}

//Personal.AI order the ending
