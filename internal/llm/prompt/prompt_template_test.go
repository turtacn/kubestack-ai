package prompt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGoTemplate_Render(t *testing.T) {
	tmplStr := "Analyze {{.Service}} with issue: {{.Issue}}"
	tmpl, err := NewGoTemplate("test", tmplStr)
	assert.NoError(t, err)

	data := map[string]string{
		"Service": "Redis",
		"Issue":   "high memory",
	}

	output, err := tmpl.Render(data)
	assert.NoError(t, err)
	assert.Equal(t, "Analyze Redis with issue: high memory", output)
}

func TestGoTemplate_Funcs(t *testing.T) {
	tmplStr := "{{truncate .Message 5}}"
	tmpl, err := NewGoTemplate("test_funcs", tmplStr)
	assert.NoError(t, err)

	data := map[string]string{
		"Message": "HelloWorld",
	}

	output, err := tmpl.Render(data)
	assert.NoError(t, err)
	assert.Equal(t, "Hello", output)
}

func TestPromptTemplate_Validate(t *testing.T) {
    // Valid template
    tmpl, err := NewGoTemplate("valid", "{{.Name}}")
    assert.NoError(t, err)
    assert.NoError(t, tmpl.Validate())

    // We can't easily create an invalid template that passes NewGoTemplate but fails Validate
    // because NewGoTemplate does parsing.
    // But we can test that NewGoTemplate fails on bad syntax.
    _, err = NewGoTemplate("invalid", "{{.Name")
    assert.Error(t, err)
}
