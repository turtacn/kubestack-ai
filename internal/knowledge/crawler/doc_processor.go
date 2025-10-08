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

package crawler

import (
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/knowledge/store"
)

// Chunk represents a single, embeddable piece of text derived from a larger
// document. It contains the clean text content and metadata linking it back to
// its original source. These chunks are the units that get converted into
// vector embeddings for similarity searches.
type Chunk struct {
	// ID is the unique identifier for this specific chunk.
	ID string
	// Content is the processed, clean text content of the chunk.
	Content string
	// SourceID is the ID of the original RawDocument from which this chunk was derived.
	SourceID string
	// SourceURL is the URL or path of the original document.
	SourceURL string
	// Metadata contains any additional, chunk-specific information.
	Metadata map[string]interface{}
}

// DocProcessor defines the interface for components that process raw documents
// into clean, embeddable chunks. This allows for different chunking strategies
// (e.g., simple text splitting, markdown-aware splitting) to be used interchangeably.
type DocProcessor interface {
	// Process takes a raw document, cleans its content, and splits it into
	// one or more chunks suitable for embedding.
	//
	// Parameters:
	//   doc (*store.RawDocument): The raw document to be processed.
	//
	// Returns:
	//   []Chunk: A slice of chunks derived from the document.
	//   error: An error if processing fails.
	Process(doc *store.RawDocument) ([]Chunk, error)
}

// textSplitter is an implementation of DocProcessor that cleans and splits raw text content
// using a fixed-size window with overlap.
type textSplitter struct {
	log             logger.Logger
	chunkSize       int
	chunkOverlap    int
	spaceNormalizer *regexp.Regexp
}

// NewTextSplitter creates a new document processor that uses a simple, fixed-size
// text splitting strategy. This is a common and effective baseline for chunking documents.
//
// Parameters:
//   chunkSize (int): The target size of each chunk in characters.
//   chunkOverlap (int): The number of characters to overlap between consecutive chunks to preserve context.
//
// Returns:
//   DocProcessor: A new instance of a text-splitting document processor.
//   error: An error if initialization fails (nil in this implementation).
func NewTextSplitter(chunkSize, chunkOverlap int) (DocProcessor, error) {
	return &textSplitter{
		log:             logger.NewLogger("doc-processor"),
		chunkSize:       chunkSize,
		chunkOverlap:    chunkOverlap,
		spaceNormalizer: regexp.MustCompile(`\s+`),
	}, nil
}

// Process implements the DocProcessor interface for the textSplitter. It first
// cleans the raw text by normalizing whitespace, then splits the cleaned text
// into fixed-size chunks with a specified overlap.
//
// Parameters:
//   doc (*store.RawDocument): The raw document to be processed.
//
// Returns:
//   []Chunk: A slice of chunks derived from the document.
//   error: An error if processing fails (nil in this implementation).
func (p *textSplitter) Process(doc *store.RawDocument) ([]Chunk, error) {
	p.log.Infof("Processing document from source: %s", doc.Source)

	// 1. Clean the text content.
	// Replace multiple whitespace characters (including newlines, tabs) with a single space.
	cleanedContent := p.spaceNormalizer.ReplaceAllString(doc.Content, " ")
	cleanedContent = strings.TrimSpace(cleanedContent)

	if len(cleanedContent) == 0 {
		p.log.Warnf("Document from %s has no content after cleaning.", doc.Source)
		return []Chunk{}, nil
	}

	// 2. Chunk the text using a sliding window.
	// This is a simple but effective strategy for ensuring semantic context is not lost at chunk boundaries.
	var chunks []Chunk
	start := 0
	for start < len(cleanedContent) {
		end := start + p.chunkSize
		if end > len(cleanedContent) {
			end = len(cleanedContent)
		}

		chunkContent := cleanedContent[start:end]

		chunks = append(chunks, Chunk{
			ID:        uuid.New().String(),
			Content:   chunkContent,
			SourceID:  doc.ID,
			SourceURL: doc.Source,
			Metadata:  map[string]interface{}{"chunk_number": len(chunks) + 1},
		})

		if end == len(cleanedContent) {
			break
		}

		start += p.chunkSize - p.chunkOverlap
	}

	p.log.Infof("Split document from %s into %d chunks.", doc.Source, len(chunks))

	// TODO: Implement processing for other formats (PDF, DOCX). This would involve using libraries
	// to extract raw text first, then applying this same cleaning/chunking logic.
	// TODO: Implement more advanced, content-aware chunking strategies, such as splitting on
	// sentences or markdown headers (RecursiveCharacterTextSplitter).

	return chunks, nil
}

//Personal.AI order the ending
