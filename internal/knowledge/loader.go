package knowledge

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"gopkg.in/yaml.v2"
)

// RuleLoader handles loading and reloading of rules.
type RuleLoader struct {
	kb         *KnowledgeBase
	log        logger.Logger
	watchPaths []string
	watcher    *fsnotify.Watcher
	mu         sync.Mutex
	loadedFiles map[string]string // ID -> Filepath
}

// NewRuleLoader creates a new RuleLoader.
func NewRuleLoader(kb *KnowledgeBase) *RuleLoader {
	return &RuleLoader{
		kb:          kb,
		log:         logger.NewLogger("rule-loader"),
		loadedFiles: make(map[string]string),
	}
}

// LoadFromFile loads rules from a YAML file.
func (rl *RuleLoader) LoadFromFile(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", path, err)
	}

	var rules []Rule
	if err := yaml.Unmarshal(data, &rules); err != nil {
		return fmt.Errorf("failed to parse YAML in %s: %w", path, err)
	}

	successCount := 0
	rl.mu.Lock()
	defer rl.mu.Unlock()

	for _, rule := range rules {
		// Create a copy of the loop variable
		r := rule
		if err := rl.kb.AddRule(&r); err != nil {
			rl.log.Errorf("Failed to add rule %s from %s: %v", r.ID, path, err)
			continue
		}
		rl.loadedFiles[r.ID] = path
		successCount++
	}

	rl.log.Infof("Loaded %d rules from %s", successCount, path)
	return nil
}

// LoadFromDirectory loads all .yaml or .json files from a directory.
func (rl *RuleLoader) LoadFromDirectory(dir string) error {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		ext := filepath.Ext(file.Name())
		if ext == ".yaml" || ext == ".yml" || ext == ".json" {
			path := filepath.Join(dir, file.Name())
			if err := rl.LoadFromFile(path); err != nil {
				rl.log.Warnf("Failed to load rules from %s: %v", path, err)
			}
		}
	}
	return nil
}

// WatchAndReload watches for file changes and reloads rules.
func (rl *RuleLoader) WatchAndReload(ctx context.Context, paths []string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	rl.watcher = watcher
	rl.watchPaths = paths

	for _, path := range paths {
		if err := watcher.Add(path); err != nil {
			return fmt.Errorf("failed to watch path %s: %w", path, err)
		}
	}

	go func() {
		defer watcher.Close()
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
					rl.log.Infof("File changed: %s", event.Name)
					// Simple debounce or just reload
					// Check if it's a file we care about
					ext := filepath.Ext(event.Name)
					if ext == ".yaml" || ext == ".yml" || ext == ".json" {
						if err := rl.LoadFromFile(event.Name); err != nil {
							rl.log.Errorf("Failed to reload file %s: %v", event.Name, err)
						}
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				rl.log.Errorf("Watcher error: %v", err)
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

// SaveRule saves a rule to the underlying file.
// If the rule is new, it saves to a default file based on middleware type.
func (rl *RuleLoader) SaveRule(rule *Rule) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Determine file path
	path, exists := rl.loadedFiles[rule.ID]
	if !exists {
		// New rule, default to repo directory based on middleware
		// Ensure directory exists
		repoDir := "internal/knowledge/repository"
		if _, err := os.Stat(repoDir); os.IsNotExist(err) {
			_ = os.MkdirAll(repoDir, 0755)
		}
		path = filepath.Join(repoDir, fmt.Sprintf("%s_rules.yaml", rule.MiddlewareType))
	}

	// Read existing rules
	var rules []Rule
	if _, err := os.Stat(path); err == nil {
		data, err := ioutil.ReadFile(path)
		if err == nil {
			_ = yaml.Unmarshal(data, &rules)
		}
	}

	// Update or Append
	found := false
	for i, r := range rules {
		if r.ID == rule.ID {
			rules[i] = *rule
			found = true
			break
		}
	}
	if !found {
		rules = append(rules, *rule)
	}

	// Write back
	data, err := yaml.Marshal(rules)
	if err != nil {
		return fmt.Errorf("failed to marshal rules: %w", err)
	}

	if err := ioutil.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", path, err)
	}

	rl.loadedFiles[rule.ID] = path
	return nil
}

// DeleteRule removes a rule from the underlying file.
func (rl *RuleLoader) DeleteRule(id string) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	path, exists := rl.loadedFiles[id]
	if !exists {
		return fmt.Errorf("rule %s not associated with any file", id)
	}

	// Read existing rules
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", path, err)
	}

	var rules []Rule
	if err := yaml.Unmarshal(data, &rules); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Remove
	newRules := rules[:0]
	found := false
	for _, r := range rules {
		if r.ID == id {
			found = true
			continue
		}
		newRules = append(newRules, r)
	}

	if !found {
		return fmt.Errorf("rule %s not found in file %s", id, path)
	}

	// Write back
	data, err = yaml.Marshal(newRules)
	if err != nil {
		return fmt.Errorf("failed to marshal rules: %w", err)
	}

	if err := ioutil.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", path, err)
	}

	delete(rl.loadedFiles, id)
	return nil
}
