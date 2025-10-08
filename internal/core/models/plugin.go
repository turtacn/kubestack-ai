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

package models

import (
	"github.com/kubestack-ai/kubestack-ai/internal/common/types/enum"
)

// PluginManifest provides a structured representation of a plugin's metadata. It is
// typically loaded from a manifest file (e.g., `plugin.yaml`) and serves as the
// blueprint for the plugin manager to load, manage, and understand the plugin's
// capabilities and dependencies.
type PluginManifest struct {
	// APIVersion specifies the version of the KubeStack-AI plugin API that this plugin conforms to.
	APIVersion string `json:"apiVersion" yaml:"apiVersion"`
	// Name is the unique, machine-readable name of the plugin (e.g., "redis-plugin").
	Name string `json:"name" yaml:"name"`
	// Version is the semantic version of the plugin itself (e.g., "1.2.3").
	Version string `json:"version" yaml:"version"`
	// Author is the name or organization that created the plugin.
	Author string `json:"author,omitempty" yaml:"author,omitempty"`
	// Description is a brief, human-readable summary of the plugin's purpose.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	// Entrypoint specifies the location of the plugin's executable code (e.g., a path to a .so file for Go plugins).
	Entrypoint string `json:"entrypoint" yaml:"entrypoint"`
	// Capabilities is a list of features or actions that the plugin supports.
	Capabilities []PluginCapability `json:"capabilities,omitempty" yaml:"capabilities,omitempty"`
	// Dependencies is a list of other plugins or external tools that this plugin requires to function.
	Dependencies []PluginDependency `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
}

// PluginInfo contains basic, human-readable information about a plugin, suitable
// for display in command-line outputs like `plugin list`.
type PluginInfo struct {
	// Name is the display name of the plugin.
	Name string `json:"name" yaml:"name"`
	// Version is the semantic version of the plugin.
	Version string `json:"version" yaml:"version"`
	// Description is a brief summary of the plugin's purpose.
	Description string `json:"description" yaml:"description"`
	// Author is the creator of the plugin.
	Author string `json:"author" yaml:"author"`
}

// PluginCapability describes a specific feature or action that the plugin supports,
// allowing the core engine to understand what a plugin can do.
type PluginCapability struct {
	// Name is a machine-readable identifier for the capability (e.g., "diagnose:redis", "collect-metrics:redis").
	Name string `json:"name" yaml:"name"`
	// Description is a human-readable explanation of the capability.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// PluginDependency describes a dependency on another plugin or an external tool
// that must be present for this plugin to function correctly.
type PluginDependency struct {
	// Name is the name of the required dependency.
	Name string `json:"name" yaml:"name"`
	// Version is a semantic version constraint for the dependency (e.g., ">=1.2.0, <2.0.0").
	Version string `json:"version" yaml:"version"`
}

// PluginConfiguration defines a generic structure for a plugin's specific
// configuration. Each plugin can define its own key-value pairs within the data map.
type PluginConfiguration struct {
	// Data holds the plugin-specific configuration settings.
	Data map[string]interface{} `json:"data" yaml:"data"`
}

// PluginStatus represents the runtime status of a loaded plugin, which is used
// for health monitoring and management by the plugin manager.
type PluginStatus struct {
	// Name is the name of the plugin.
	Name string `json:"name" yaml:"name"`
	// Version is the version of the plugin.
	Version string `json:"version" yaml:"version"`
	// Status is the current lifecycle status of the plugin (e.g., Active, Failed).
	Status enum.PluginStatus `json:"status" yaml:"status"`
	// Message provides additional details on the status, such as an error message if the status is "Failed".
	Message string `json:"message,omitempty" yaml:"message,omitempty"`
}

//Personal.AI order the ending
