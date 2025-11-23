package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStructuredOutputParser_Parse(t *testing.T) {
	parser := NewStructuredOutputParser()

	tests := []struct {
		name        string
		input       string
		wantErr     bool
		checkResult func(*DiagnosisResult)
	}{
		{
			name: "Valid JSON",
			input: `{
				"root_cause": "OOM",
				"severity": "high",
				"confidence": 0.9,
				"affected_components": ["redis"]
			}`,
			wantErr: false,
			checkResult: func(res *DiagnosisResult) {
				assert.Equal(t, "OOM", res.RootCause)
				assert.Equal(t, "high", res.Severity)
				assert.Equal(t, 0.9, res.Confidence)
			},
		},
		{
			name: "Valid JSON with Markdown",
			input: "```json\n" + `{
				"root_cause": "OOM",
				"severity": "critical",
				"confidence": 1.0,
				"affected_components": ["redis"]
			}` + "\n```",
			wantErr: false,
			checkResult: func(res *DiagnosisResult) {
				assert.Equal(t, "critical", res.Severity)
			},
		},
		{
			name: "Missing Required Field",
			input: `{
				"severity": "high"
			}`,
			wantErr: true,
			checkResult: nil,
		},
		{
			name: "Invalid Enum",
			input: `{
				"root_cause": "bug",
				"severity": "unknown",
				"confidence": 0.5,
				"affected_components": ["app"]
			}`,
			wantErr: true,
			checkResult: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := parser.Parse(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.checkResult != nil {
					tt.checkResult(res)
				}
			}
		})
	}
}
