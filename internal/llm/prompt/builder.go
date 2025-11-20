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

package prompt

import (
	"bytes"
	"text/template"
)

// Builder helps construct prompts for the language model by combining a
// template with dynamic data.
type Builder struct {
	template *template.Template
	data     map[string]interface{}
}

// NewBuilder creates a new prompt builder from a template.
func NewBuilder(tmpl Template) (*Builder, error) {
	t, err := template.New(tmpl.Name).Parse(tmpl.Content)
	if err != nil {
		return nil, err
	}
	return &Builder{
		template: t,
		data:     make(map[string]interface{}),
	}, nil
}

// WithData adds data to the prompt.
func (b *Builder) WithData(key string, value interface{}) *Builder {
	b.data[key] = value
	return b
}

// Build constructs the final prompt string.
func (b *Builder) Build() (string, error) {
	var buf bytes.Buffer
	if err := b.template.Execute(&buf, b.data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
