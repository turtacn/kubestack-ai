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
// limitations under the 'License'.
package crawler

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTMLCleaner_Clean(t *testing.T) {
	cleaner := NewHTMLCleaner()

	html := `
		<html>
			<head><title>Test</title></head>
			<body>
				<nav>Menu</nav>
				<main>
					<h1>Title</h1>
					<p>This is a paragraph.</p>
					<pre><code>code block</code></pre>
				</main>
				<footer>Footer</footer>
				<script>alert("hello")</script>
			</body>
		</html>
	`

	expectedMarkdown := "# Title\n\nThis is a paragraph.\n\n```\ncode block\n```"

	markdown, err := cleaner.Clean(html)
	assert.NoError(t, err)
	assert.Equal(t, expectedMarkdown, markdown)
}
