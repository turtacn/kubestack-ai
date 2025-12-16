package rules

import (
	"encoding/json"
	"os"
)

// RuleSet defines a collection of rules
type RuleSet struct {
	Name        string
	Version     string
	Description string
	Rules       []Rule
}

// Rule defines a single rule structure (internal representation)
// Note: This maps to DiagnosisRule in interface.go, but for loading from file we might have slightly different structure or use the same.
// Let's reuse basic structure but keep it decoupled if needed.
type Rule struct {
	ID          string   `json:"id" yaml:"id"`
	Name        string   `json:"name" yaml:"name"`
	Description string   `json:"description" yaml:"description"`
	Severity    string   `json:"severity" yaml:"severity"`
	Condition   Condition `json:"condition" yaml:"condition"`
	Message     string   `json:"message" yaml:"message"`
	Suggestion  string   `json:"suggestion" yaml:"suggestion"`
	Tags        []string `json:"tags" yaml:"tags"`
	Enabled     bool     `json:"enabled" yaml:"enabled"`
}

// Condition defines the logic condition
type Condition struct {
	Expression string `json:"expression" yaml:"expression"`
}

// RuleLoader interface
type RuleLoader interface {
	Load(path string) (*RuleSet, error)
	LoadFromJSON(data []byte) (*RuleSet, error)
}

// JSONRuleLoader implementation
type JSONRuleLoader struct{}

func (l *JSONRuleLoader) Load(path string) (*RuleSet, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return l.LoadFromJSON(data)
}

func (l *JSONRuleLoader) LoadFromJSON(data []byte) (*RuleSet, error) {
	var ruleSet RuleSet
	if err := json.Unmarshal(data, &ruleSet); err != nil {
		return nil, err
	}
	return &ruleSet, nil
}
