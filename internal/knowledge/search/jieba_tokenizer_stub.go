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

//go:build !cgo
// +build !cgo

package search

import (
	"github.com/blevesearch/bleve/v2/analysis"
	"github.com/blevesearch/bleve/v2/registry"
)

const (
	JiebaTokenizerName = "jieba"
)

// JiebaTokenizer is a stub implementation when CGO is disabled
type JiebaTokenizer struct{}

func NewJiebaTokenizer() *JiebaTokenizer {
	return &JiebaTokenizer{}
}

func (t *JiebaTokenizer) Tokenize(sentence []byte) analysis.TokenStream {
	// Fallback to empty tokenization when CGO is disabled
	// Note: This is a stub implementation - full functionality requires CGO
	return analysis.TokenStream{}
}

func (t *JiebaTokenizer) Free() {
	// No-op when CGO is disabled
}

func tokenizerConstructor(config map[string]interface{}, cache *registry.Cache) (analysis.Tokenizer, error) {
	return NewJiebaTokenizer(), nil
}

func init() {
	// Register stub tokenizer when CGO is disabled
	registry.RegisterTokenizer(JiebaTokenizerName, tokenizerConstructor)
}
