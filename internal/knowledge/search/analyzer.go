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

package search

import (
	"github.com/blevesearch/bleve/v2/analysis"
	"github.com/blevesearch/bleve/v2/analysis/token/lowercase"
	"github.com/blevesearch/bleve/v2/registry"
)

const JiebaAnalyzerName = "jieba"

// customJiebaAnalyzer is a custom bleve analyzer that uses the jieba tokenizer.
type customJiebaAnalyzer struct {
	tokenizer analysis.Tokenizer
}

// Analyze performs the analysis of the input text.
func (a *customJiebaAnalyzer) Analyze(input []byte) analysis.TokenStream {
	tokens := a.tokenizer.Tokenize(input)
	filter := lowercase.NewLowerCaseFilter()
	return filter.Filter(tokens)
}

// newJiebaAnalyzer creates a new analyzer.
func newJiebaAnalyzer(tokenizer analysis.Tokenizer) *customJiebaAnalyzer {
	return &customJiebaAnalyzer{
		tokenizer: tokenizer,
	}
}

// analyzerConstructor is the constructor function that will be registered with bleve.
func analyzerConstructor(config map[string]interface{}, cache *registry.Cache) (analysis.Analyzer, error) {
	tokenizer, err := cache.TokenizerNamed(JiebaTokenizerName)
	if err != nil {
		return nil, err
	}
	return newJiebaAnalyzer(tokenizer), nil
}

func init() {
	registry.RegisterAnalyzer(JiebaAnalyzerName, analyzerConstructor)
}
