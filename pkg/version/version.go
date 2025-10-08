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
	// Version is the semantic version of the application. It is populated at build time.
	Version = "dev" // Default to 'dev' for local builds
	// GitCommit is the short git commit hash of the source code. It is populated at build time.
	GitCommit = "unknown"
	// BuildDate is the date when the binary was built. It is populated at build time.
	BuildDate = "unknown"
	// GoVersion is the version of the Go compiler used to build the binary.
	GoVersion = runtime.Version()
	// Platform is the operating system and architecture for which the binary was built.
	Platform = fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
)

// Info holds all version-related information in a structured format.
// This struct is useful for serializing version data to formats like JSON.
type Info struct {
	// Version is the semantic version of the application.
	Version string `json:"version"`
	// GitCommit is the short git commit hash of the source code.
	GitCommit string `json:"gitCommit"`
	// BuildDate is the date when the binary was built.
	BuildDate string `json:"buildDate"`
	// GoVersion is the version of the Go compiler used.
	GoVersion string `json:"goVersion"`
	// Platform is the operating system and architecture.
	Platform string `json:"platform"`
}

// Get returns a struct containing all the version information, populated from the
// package-level variables.
//
// Returns:
//   Info: A struct filled with the application's version details.
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
//
// Returns:
//   string: A formatted string containing all version details.
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
