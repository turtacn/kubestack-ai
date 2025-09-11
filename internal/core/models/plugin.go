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

// PluginManifest is the structured representation of a plugin's metadata. It is typically
// loaded from a manifest file (e.g., plugin.yaml) and serves as the blueprint for loading
// and managing the plugin.
type PluginManifest struct {
	APIVersion   string             `json:"apiVersion" yaml:"apiVersion"` // KubeStack-AI plugin API version this plugin conforms to.
	Name         string             `json:"name" yaml:"name"`
	Version      string             `json:"version" yaml:"version"` // Semantic version of the plugin.
	Author       string             `json:"author,omitempty" yaml:"author,omitempty"`
	Description  string             `json:"description,omitempty" yaml:"description,omitempty"`
	Entrypoint   string             `json:"entrypoint" yaml:"entrypoint"` // e.g., path to the .so file for Go plugins.
	Capabilities []PluginCapability `json:"capabilities,omitempty" yaml:"capabilities,omitempty"`
	Dependencies []PluginDependency `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
}

// PluginInfo contains basic, human-readable information about a plugin, often used for display purposes.
type PluginInfo struct {
	Name        string `json:"name" yaml:"name"`
	Version     string `json:"version" yaml:"version"`
	Description string `json:"description" yaml:"description"`
	Author      string `json:"author" yaml:"author"`
}

// PluginCapability describes a specific feature or action that the plugin supports.
type PluginCapability struct {
	Name        string `json:"name" yaml:"name"` // e.g., "diagnose:redis", "collect-metrics:redis"
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// PluginDependency describes a dependency on another plugin or an external tool.
type PluginDependency struct {
	Name    string `json:"name" yaml:"name"`
	Version string `json:"version" yaml:"version"` // A version constraint, e.g., ">=1.2.0, <2.0.0"
}

// PluginConfiguration defines the structure for a plugin's specific configuration.
// It acts as a generic container; each plugin will define its own specific fields within the data map.
type PluginConfiguration struct {
	Data map[string]interface{} `json:"data" yaml:"data"`
}

// PluginStatus represents the runtime status of a loaded plugin, used for health monitoring and management.
type PluginStatus struct {
	Name    string            `json:"name" yaml:"name"`
	Version string            `json:"version" yaml:"version"`
	Status  enum.PluginStatus `json:"status" yaml:"status"`
	Message string            `json:"message,omitempty" yaml:"message,omitempty"` // Provides details on the status, e.g., error message on failure.
}

//Personal.AI order the ending
