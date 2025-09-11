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

// Package version provides the single source of truth for the application's version.
package version

import (
	"fmt"
	"runtime"
)

// These variables are populated at build time via -ldflags.
// Example:
// go build -ldflags "-X 'github.com/kubestack-ai/kubestack-ai/pkg/version.Version=v1.0.0' \
// -X 'github.com/kubestack-ai/kubestack-ai/pkg/version.GitCommit=...'"
var (
	Version   = "dev" // Default to 'dev' for local builds
	GitCommit = "unknown"
	BuildDate = "unknown"
	GoVersion = runtime.Version()
	Platform  = fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
)

// Info holds all version-related information.
type Info struct {
	Version   string `json:"version"`
	GitCommit string `json:"gitCommit"`
	BuildDate string `json:"buildDate"`
	GoVersion string `json:"goVersion"`
	Platform  string `json:"platform"`
}

// Get returns a struct containing all the version information.
func Get() Info {
	return Info{
		Version:   Version,
		GitCommit: GitCommit,
		BuildDate: BuildDate,
		GoVersion: GoVersion,
		Platform:  Platform,
	}
}

// String returns a nicely formatted, multi-line string representation of the version info,
// suitable for printing in a CLI `version` command.
func (i Info) String() string {
	return fmt.Sprintf(
		`Version: %s
Git Commit: %s
Build Date: %s
Go Version: %s
Platform: %s`,
		i.Version, i.GitCommit, i.BuildDate, i.GoVersion, i.Platform,
	)
}

// TODO: Implement a function to check for new versions against a remote source like the GitHub releases API.
// This would allow the CLI to prompt users to upgrade.
//
// TODO: Implement a function to check for API compatibility between different components,
// for example, ensuring a plugin's required API version is compatible with the main application's version.

//Personal.AI order the ending
