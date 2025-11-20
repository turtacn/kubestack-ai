package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPromptRenderer_Render(t *testing.T) {
	renderer, err := NewPromptRenderer()
	assert.NoError(t, err)

	testCases := []struct {
		name         string
		templateName string
		data         interface{}
		expectErr    bool
		expected     string
	}{
		{
			name:         "Valid Diagnosis Template",
			templateName: "diagnosis",
			data: map[string]interface{}{
				"PluginName": "Redis",
				"Timestamp":  "2024-01-01",
				"UserQuery":  "it's slow",
				"SystemLogs": "OOM killer",
				"MetricData": "memory > 90%",
			},
			expectErr: false,
			expected:  "Plugin: Redis",
		},
		{
			name:         "Template Not Found",
			templateName: "non_existent",
			data:         nil,
			expectErr:    true,
			expected:     "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := renderer.Render(tc.templateName, tc.data)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, result, tc.expected)
			}
		})
	}
}
