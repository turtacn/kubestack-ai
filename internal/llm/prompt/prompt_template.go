package prompt

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

// PromptTemplate defines the interface for rendering prompts.
type PromptTemplate interface {
	Render(data interface{}) (string, error)
	Validate() error
}

// GoTemplate implements PromptTemplate using the text/template package.
type GoTemplate struct {
	name    string
	rawTmpl string
	tmpl    *template.Template
}

// NewGoTemplate creates a new GoTemplate instance.
func NewGoTemplate(name, rawTmpl string) (*GoTemplate, error) {
	funcMap := template.FuncMap{
		"truncate": func(s string, n int) string {
			if len(s) > n {
				return s[:n]
			}
			return s
		},
		"join": strings.Join,
	}

	tmpl, err := template.New(name).Funcs(funcMap).Parse(rawTmpl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	return &GoTemplate{
		name:    name,
		rawTmpl: rawTmpl,
		tmpl:    tmpl,
	}, nil
}

// Render renders the template with the provided data.
func (t *GoTemplate) Render(data interface{}) (string, error) {
	var buf bytes.Buffer
	if err := t.tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to render template: %w", err)
	}
	return buf.String(), nil
}

// Validate checks if the template is valid.
func (t *GoTemplate) Validate() error {
	// Simple validation: try rendering with empty data (nil) might panic or fail if fields are accessed.
	// A better approach for Go templates is harder without schema, but we can check if Parse succeeded (done in New).
	// We can try to Parse again to ensure rawTmpl is valid.
	_, err := template.New(t.name).Parse(t.rawTmpl)
	return err
}
