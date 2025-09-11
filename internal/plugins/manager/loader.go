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
	"plugin"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// goPluginLoader is a concrete implementation of PluginLoader for standard Go plugins,
// which are compiled as shared object (.so) files.
type goPluginLoader struct {
	log logger.Logger
}

// NewLoader creates a new plugin loader. In the future, this could take a configuration
// to decide which type of loader to create (e.g., for scripts, containers).
func NewLoader() interfaces.PluginLoader {
	return &goPluginLoader{
		log: logger.NewLogger("plugin-loader"),
	}
}

// Load takes a plugin manifest and loads the plugin's code into memory,
// returning an initialized instance that implements the MiddlewarePlugin interface.
func (l *goPluginLoader) Load(manifest *models.PluginManifest) (interfaces.MiddlewarePlugin, error) {
	l.log.Infof("Attempting to load Go plugin from entrypoint: %s", manifest.Entrypoint)

	// Security checks would be performed here in a production system, for example:
	// 1. Signature Verification: Check if the .so file is signed by a trusted key to ensure its integrity.
	// 2. Sandbox Setup: Prepare a sandboxed environment if untrusted plugins are allowed (this is very complex).
	// if err := l.verifySignature(manifest.Entrypoint); err != nil { ... }

	p, err := plugin.Open(manifest.Entrypoint)
	if err != nil {
		return nil, fmt.Errorf("failed to open plugin file '%s': %w", manifest.Entrypoint, err)
	}

	// By convention, we expect each plugin to export a symbol named "New".
	// This symbol must be a factory function that creates an instance of the plugin.
	newFuncSymbol, err := p.Lookup("New")
	if err != nil {
		return nil, fmt.Errorf("failed to find required symbol 'New' in plugin '%s': %w", manifest.Name, err)
	}

	// Type-assert the symbol to the expected factory function signature.
	// The signature is: func() (interfaces.MiddlewarePlugin, error)
	newFunc, ok := newFuncSymbol.(func() (interfaces.MiddlewarePlugin, error))
	if !ok {
		return nil, fmt.Errorf("symbol 'New' in plugin '%s' has an incorrect function signature", manifest.Name)
	}

	// Call the factory function to create the plugin instance.
	pluginInstance, err := newFunc()
	if err != nil {
		return nil, fmt.Errorf("the factory function 'New' for plugin '%s' returned an error on creation: %w", manifest.Name, err)
	}

	// As a final sanity check, ensure the loaded plugin's name matches the manifest.
	if pluginInstance.Name() != manifest.Name {
		l.log.Warnf("Loaded plugin identified itself as '%s', but manifest name is '%s'.", pluginInstance.Name(), manifest.Name)
	}

	l.log.Infof("Successfully loaded and instantiated plugin: %s", manifest.Name)
	return pluginInstance, nil
}

// TODO: Implement loaders for other plugin formats if the system needs to support them.
// For example, a loader for plugins written as Python scripts:
//
// type scriptPluginLoader struct { ... }
// func (l *scriptPluginLoader) Load(...) (interfaces.MiddlewarePlugin, error) {
//   // Logic to execute a script and communicate with it, perhaps over stdin/stdout or a unix socket.
//   ...
// }
//
// Or for plugins distributed as containers:
//
// type containerPluginLoader struct { ... }
// func (l *containerPluginLoader) Load(...) (interfaces.MiddlewarePlugin, error) {
//   // Logic to start a container and interact with it via an exposed API.
//   ...
// }

//Personal.AI order the ending
