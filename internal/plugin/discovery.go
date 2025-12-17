package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// PluginManifest describes a plugin's metadata
type PluginManifest struct {
	ID           string   `yaml:"id"`
	Name         string   `yaml:"name"`
	Version      string   `yaml:"version"`
	Type         string   `yaml:"type"`
	Description  string   `yaml:"description"`
	Author       string   `yaml:"author"`
	Homepage     string   `yaml:"homepage"`
	License      string   `yaml:"license"`
	Requires     []string `yaml:"requires"`
	Capabilities []string `yaml:"capabilities"`
	EntryPoint   string   `yaml:"entry_point"` // Path to .so file or script
}

// Discovery handles automatic plugin discovery
type Discovery struct {
	pluginDirs []string
}

// NewDiscovery creates a new discovery instance
func NewDiscovery(dirs ...string) *Discovery {
	return &Discovery{
		pluginDirs: dirs,
	}
}

// DiscoverPlugins scans directories for plugin manifests
func (d *Discovery) DiscoverPlugins() ([]*PluginManifest, error) {
	manifests := make([]*PluginManifest, 0)
	
	for _, dir := range d.pluginDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}
		
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
		}
		
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			
			// Look for manifest.yaml in each subdirectory
			manifestPath := filepath.Join(dir, entry.Name(), "manifest.yaml")
			manifest, err := d.loadManifest(manifestPath)
			if err != nil {
				// Skip directories without valid manifests
				continue
			}
			
			manifests = append(manifests, manifest)
		}
	}
	
	return manifests, nil
}

// DiscoverSharedLibraries finds .so files that might be plugins
func (d *Discovery) DiscoverSharedLibraries() ([]string, error) {
	libraries := make([]string, 0)
	
	for _, dir := range d.pluginDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}
		
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			
			if !info.IsDir() && strings.HasSuffix(path, ".so") {
				libraries = append(libraries, path)
			}
			
			return nil
		})
		
		if err != nil {
			return nil, fmt.Errorf("failed to walk directory %s: %w", dir, err)
		}
	}
	
	return libraries, nil
}

// loadManifest loads a plugin manifest from a file
func (d *Discovery) loadManifest(path string) (*PluginManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}
	
	var manifest PluginManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}
	
	// Validate required fields
	if manifest.ID == "" || manifest.Name == "" || manifest.Version == "" {
		return nil, fmt.Errorf("manifest missing required fields")
	}
	
	// Resolve relative entry point path
	if manifest.EntryPoint != "" && !filepath.IsAbs(manifest.EntryPoint) {
		manifest.EntryPoint = filepath.Join(filepath.Dir(path), manifest.EntryPoint)
	}
	
	return &manifest, nil
}

// ValidateManifest validates a plugin manifest
func ValidateManifest(manifest *PluginManifest) error {
	if manifest.ID == "" {
		return fmt.Errorf("plugin ID is required")
	}
	if manifest.Name == "" {
		return fmt.Errorf("plugin name is required")
	}
	if manifest.Version == "" {
		return fmt.Errorf("plugin version is required")
	}
	if manifest.Type == "" {
		return fmt.Errorf("plugin type is required")
	}
	
	// Validate type
	validTypes := map[string]bool{
		"middleware":  true,
		"diagnostic":  true,
		"action":      true,
		"integration": true,
	}
	
	if !validTypes[manifest.Type] {
		return fmt.Errorf("invalid plugin type: %s", manifest.Type)
	}
	
	return nil
}
