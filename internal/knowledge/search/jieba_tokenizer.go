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
	"github.com/blevesearch/bleve/v2/registry"
	"github.com/yanyiwu/gojieba"
)

const (
	JiebaTokenizerName = "jieba"
)

type JiebaTokenizer struct {
	jieba *gojieba.Jieba
}

func NewJiebaTokenizer() *JiebaTokenizer {
	return &JiebaTokenizer{
		jieba: gojieba.NewJieba(),
	}
}

func (t *JiebaTokenizer) Tokenize(sentence []byte) analysis.TokenStream {
	result := make(analysis.TokenStream, 0)
	words := t.jieba.CutForSearch(string(sentence), true)
	for _, word := range words {
		token := analysis.Token{
			Term:     []byte(word),
			Start:    0,
			End:      0,
			Position: 0,
			Type:     analysis.Ideographic,
		}
		result = append(result, &token)
	}
	return result
}

func (t *JiebaTokenizer) Free() {
	t.jieba.Free()
}

func tokenizerConstructor(config map[string]interface{}, cache *registry.Cache) (analysis.Tokenizer, error) {
	return NewJiebaTokenizer(), nil
}

func init() {
	registry.RegisterTokenizer(JiebaTokenizerName, tokenizerConstructor)
}
