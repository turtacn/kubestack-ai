package knowledge

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Rule represents a diagnostic rule in the knowledge base.
type Rule struct {
	ID             string   `json:"id" yaml:"id"`
	Name           string   `json:"name" yaml:"name"`
	MiddlewareType string   `json:"middleware_type" yaml:"middleware_type"`
	Category       string   `json:"category" yaml:"category"` // e.g., performance, stability, security
	Severity       string   `json:"severity" yaml:"severity"`
	Condition      string   `json:"condition" yaml:"condition"`
	Recommendation string   `json:"recommendation" yaml:"recommendation"`
	Priority       int      `json:"priority" yaml:"priority"`
	Tags           []string `json:"tags" yaml:"tags"`
	Version        string   `json:"version" yaml:"version"`
	CreatedAt      time.Time `json:"created_at" yaml:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" yaml:"updated_at"`
}

// QueryOptions defines criteria for querying rules.
type QueryOptions struct {
	MiddlewareType string
	Severity       []string
	Tags           []string
}

// KnowledgeBase manages the storage and retrieval of rules.
type KnowledgeBase struct {
	rules       map[string]*Rule
	indexByType map[string][]*Rule
	indexByTag  map[string][]*Rule
	mu          sync.RWMutex
}

// NewKnowledgeBase creates a new instance of KnowledgeBase.
func NewKnowledgeBase() *KnowledgeBase {
	return &KnowledgeBase{
		rules:       make(map[string]*Rule),
		indexByType: make(map[string][]*Rule),
		indexByTag:  make(map[string][]*Rule),
	}
}

// AddRule adds a new rule to the knowledge base.
func (kb *KnowledgeBase) AddRule(rule *Rule) error {
	kb.mu.Lock()
	defer kb.mu.Unlock()

	if err := kb.validateRule(rule); err != nil {
		return err
	}

	if rule.ID == "" {
		rule.ID = uuid.New().String()
	}
	if rule.CreatedAt.IsZero() {
		rule.CreatedAt = time.Now()
	}
	rule.UpdatedAt = time.Now()

	kb.rules[rule.ID] = rule
	kb.updateIndexes(rule)

	return nil
}

// UpdateRule updates an existing rule.
func (kb *KnowledgeBase) UpdateRule(rule *Rule) error {
	kb.mu.Lock()
	defer kb.mu.Unlock()

	if _, exists := kb.rules[rule.ID]; !exists {
		return fmt.Errorf("rule with ID %s not found", rule.ID)
	}

	if err := kb.validateRule(rule); err != nil {
		return err
	}

	rule.UpdatedAt = time.Now()
	kb.rules[rule.ID] = rule

	// Rebuild indexes (simple but effective for now; optimization possible)
	kb.rebuildIndexes()

	return nil
}

// DeleteRule removes a rule by ID.
func (kb *KnowledgeBase) DeleteRule(id string) error {
	kb.mu.Lock()
	defer kb.mu.Unlock()

	if _, exists := kb.rules[id]; !exists {
		return fmt.Errorf("rule with ID %s not found", id)
	}

	delete(kb.rules, id)
	kb.rebuildIndexes()

	return nil
}

// GetRule retrieves a rule by ID.
func (kb *KnowledgeBase) GetRule(id string) (*Rule, error) {
	kb.mu.RLock()
	defer kb.mu.RUnlock()

	rule, exists := kb.rules[id]
	if !exists {
		return nil, fmt.Errorf("rule with ID %s not found", id)
	}
	return rule, nil
}

// GetAllRules returns all rules in the knowledge base.
func (kb *KnowledgeBase) GetAllRules() ([]*Rule, error) {
	kb.mu.RLock()
	defer kb.mu.RUnlock()

	rules := make([]*Rule, 0, len(kb.rules))
	for _, r := range kb.rules {
		rules = append(rules, r)
	}
	return rules, nil
}

// QueryRules retrieves rules matching the given options.
func (kb *KnowledgeBase) QueryRules(opts QueryOptions) ([]*Rule, error) {
	kb.mu.RLock()
	defer kb.mu.RUnlock()

	var candidates []*Rule

	// Filter by MiddlewareType
	if opts.MiddlewareType != "" {
		candidates = kb.indexByType[opts.MiddlewareType]
	} else {
		for _, r := range kb.rules {
			candidates = append(candidates, r)
		}
	}

	var results []*Rule
	for _, rule := range candidates {
		if len(opts.Severity) > 0 && !contains(opts.Severity, rule.Severity) {
			continue
		}
		if len(opts.Tags) > 0 && !hasAnyTag(rule.Tags, opts.Tags) {
			continue
		}
		results = append(results, rule)
	}

	// Sort by Priority descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Priority > results[j].Priority
	})

	return results, nil
}

// ValidateRule checks if the rule is valid.
func (kb *KnowledgeBase) ValidateRule(rule *Rule) error {
	return kb.validateRule(rule)
}

func (kb *KnowledgeBase) validateRule(rule *Rule) error {
	if rule.Name == "" {
		return fmt.Errorf("rule name is required")
	}
	if rule.MiddlewareType == "" {
		return fmt.Errorf("middleware type is required")
	}
	if rule.Condition == "" {
		return fmt.Errorf("condition is required")
	}
	return nil
}

func (kb *KnowledgeBase) updateIndexes(rule *Rule) {
	kb.indexByType[rule.MiddlewareType] = append(kb.indexByType[rule.MiddlewareType], rule)
	for _, tag := range rule.Tags {
		kb.indexByTag[tag] = append(kb.indexByTag[tag], rule)
	}
}

func (kb *KnowledgeBase) rebuildIndexes() {
	kb.indexByType = make(map[string][]*Rule)
	kb.indexByTag = make(map[string][]*Rule)
	for _, rule := range kb.rules {
		kb.updateIndexes(rule)
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func hasAnyTag(ruleTags []string, queryTags []string) bool {
	for _, qt := range queryTags {
		if contains(ruleTags, qt) {
			return true
		}
	}
	return false
}
