package intent

// IntentType defines the type of intent.
type IntentType string

const (
	IntentDiagnose IntentType = "diagnose" // Diagnostic intent
	IntentQuery    IntentType = "query"    // Query intent
	IntentFix      IntentType = "fix"      // Remediation intent
	IntentAlert    IntentType = "alert"    // Alerting intent
	IntentConfig   IntentType = "config"   // Configuration intent
	IntentExplain  IntentType = "explain"  // Explanation intent
	IntentCompare  IntentType = "compare"  // Comparison intent
	IntentHelp     IntentType = "help"     // Help intent
	IntentUnknown  IntentType = "unknown"  // Unknown intent
)

// Intent represents the result of intent recognition.
type Intent struct {
	Type       IntentType        `json:"type"`
	Confidence float64           `json:"confidence"`
	Slots      map[string]string `json:"slots"`
	RawText    string            `json:"raw_text"`
	Reason     string            `json:"reason,omitempty"`
}

// IntentPattern represents a pattern for matching intents.
type IntentPattern struct {
	Type     IntentType
	Patterns []string // Regex patterns
	Keywords []string // Keywords
	Priority int      // Priority (higher value means higher priority)
}
