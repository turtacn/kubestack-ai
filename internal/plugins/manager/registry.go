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

package manager

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"
	"sync"

	"github.com/Masterminds/semver/v3"
	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
	"gopkg.in/yaml.v2"
)

const manifestFileName = "plugin.yaml"

// localRegistry is a concrete implementation of PluginRegistry that discovers plugins from the local filesystem.
// It scans specified directories for plugin subdirectories, each containing a manifest file.
type localRegistry struct {
	log        logger.Logger
	pluginDirs []string
	// The registry stores a map of plugin names to a sorted list (desc) of their available manifests.
	manifests  map[string][]*models.PluginManifest
	mu         sync.RWMutex
}

// NewRegistry creates a new local plugin registry that discovers plugins from one
// or more directories on the filesystem. It performs an initial scan upon creation.
//
// Parameters:
//   pluginDirs ([]string): A slice of directory paths to scan for plugins.
//
// Returns:
//   interfaces.PluginRegistry: A new, initialized local plugin registry.
//   error: An error if the initial plugin scan fails.
func NewRegistry(pluginDirs []string) (interfaces.PluginRegistry, error) {
	r := &localRegistry{
		log:        logger.NewLogger("plugin-registry"),
		pluginDirs: pluginDirs,
		manifests:  make(map[string][]*models.PluginManifest),
	}
	if err := r.Scan(); err != nil {
		return nil, fmt.Errorf("failed to perform initial plugin scan: %w", err)
	}
	return r, nil
}

// Scan walks the configured plugin directories, looking for subdirectories that
// contain a `plugin.yaml` manifest file. It parses each manifest, resolves the
// plugin's entrypoint path, and stores it in memory. It also sorts the available
// versions for each plugin to ensure the latest is always first. This method is
// thread-safe.
//
// Returns:
//   error: An error is not expected in the current implementation, but the signature
//          allows for future enhancements like returning errors on invalid manifests.
func (r *localRegistry) Scan() error {
	r.log.Info("Scanning for plugins...")
	r.mu.Lock()
	defer r.mu.Unlock()

	r.manifests = make(map[string][]*models.PluginManifest) // Clear existing manifests

	for _, dir := range r.pluginDirs {
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			r.log.Debugf("Could not read plugin directory '%s' (this may be normal): %v", dir, err)
			continue
		}

		for _, file := range files {
			if !file.IsDir() {
				continue
			}
			pluginDir := filepath.Join(dir, file.Name())
			manifestPath := filepath.Join(pluginDir, manifestFileName)

			data, err := ioutil.ReadFile(manifestPath)
			if err != nil {
				r.log.Warnf("Could not read manifest file at '%s': %v", manifestPath, err)
				continue
			}

			var manifest models.PluginManifest
			if err := yaml.Unmarshal(data, &manifest); err != nil {
				r.log.Warnf("Failed to parse manifest file at '%s': %v", manifestPath, err)
				continue
			}

			// The entrypoint in the manifest is relative to the plugin's root directory.
			manifest.Entrypoint = filepath.Join(pluginDir, manifest.Entrypoint)
			r.manifests[manifest.Name] = append(r.manifests[manifest.Name], &manifest)
		}
	}

	// For each plugin, sort its available versions in descending order.
	for name := range r.manifests {
		sort.Slice(r.manifests[name], func(i, j int) bool {
			vI, errI := semver.NewVersion(r.manifests[name][i].Version)
			vJ, errJ := semver.NewVersion(r.manifests[name][j].Version)
			if errI != nil || errJ != nil {
				return false // If versions are not valid semver, maintain original order
			}
			return vI.GreaterThan(vJ)
		})
	}

	r.log.Infof("Plugin scan complete. Found %d unique plugins.", len(r.manifests))
	// TODO: Implement other requirements like security scanning as part of the scan process.
	return nil
}

// FindPlugin searches the in-memory cache for a plugin that matches the given
// name and semantic version constraint. If the constraint is empty, it returns
// the latest available version. This method is thread-safe.
//
// Parameters:
//   name (string): The name of the plugin to find.
//   versionConstraint (string): A semantic versioning constraint (e.g., ">=1.2.0, <2.0.0").
//
// Returns:
//   *models.PluginManifest: The manifest of the first version found that satisfies the constraint.
//   error: An error if the plugin is not found or the version constraint is invalid.
func (r *localRegistry) FindPlugin(name string, versionConstraint string) (*models.PluginManifest, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	versions, ok := r.manifests[name]
	if !ok || len(versions) == 0 {
		return nil, fmt.Errorf("plugin '%s' not found in registry", name)
	}

	if versionConstraint == "" {
		r.log.Debugf("No version constraint for '%s', returning latest version %s.", name, versions[0].Version)
		return versions[0], nil
	}

	constraint, err := semver.NewConstraint(versionConstraint)
	if err != nil {
		return nil, fmt.Errorf("invalid version constraint '%s': %w", versionConstraint, err)
	}

	for _, manifest := range versions {
		v, err := semver.NewVersion(manifest.Version)
		if err != nil {
			continue
		}
		if constraint.Check(v) {
			r.log.Debugf("Found version %s for plugin '%s' matching constraint '%s'.", v.String(), name, versionConstraint)
			return manifest, nil
		}
	}

	return nil, fmt.Errorf("no version of plugin '%s' matches constraint '%s'", name, versionConstraint)
}

// ListAvailablePlugins returns a slice of all discovered plugin manifests from the cache.
// This method is thread-safe.
//
// Returns:
//   []*models.PluginManifest: A slice containing all loaded plugin manifests.
//   error: An error is not expected in this implementation.
func (r *localRegistry) ListAvailablePlugins() ([]*models.PluginManifest, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var all []*models.PluginManifest
	for _, versions := range r.manifests {
		all = append(all, versions...)
	}
	return all, nil
}

//Personal.AI order the ending
