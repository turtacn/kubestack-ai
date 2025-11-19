// Copyright © 2024 KubeStack-AI Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package crawler

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
	md "github.com/JohannesKaufmann/html-to-markdown"
)

// ContentCleaner defines the interface for cleaning and processing crawled content.
type ContentCleaner interface {
	Clean(rawHTML string) (string, error)
}

// HTMLCleaner is an implementation of ContentCleaner that cleans HTML content.
type HTMLCleaner struct {
	converter *md.Converter
}

// NewHTMLCleaner creates a new HTMLCleaner.
func NewHTMLCleaner() *HTMLCleaner {
	return &HTMLCleaner{
		converter: md.NewConverter("", true, nil),
	}
}

// Clean removes unwanted HTML elements and converts the content to Markdown.
func (c *HTMLCleaner) Clean(rawHTML string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(rawHTML))
	if err != nil {
		return "", err
	}

	// Remove unwanted elements
	doc.Find("nav, footer, script, style, .ads").Remove()

	// Extract the main content
	mainContent, err := doc.Find("article, main").First().Html()
	if err != nil {
		mainContent, err = doc.Find("body").Html()
		if err != nil {
			return "", err
		}
	}

	// Convert to Markdown
	markdown, err := c.converter.ConvertString(mainContent)
	if err != nil {
		return "", err
	}

	return markdown, nil
}
