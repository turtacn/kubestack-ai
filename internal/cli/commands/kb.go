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

package commands

import (
	"fmt"
	"net/url"

	"github.com/kubestack-ai/kubestack-ai/internal/core/interfaces"
	"github.com/kubestack-ai/kubestack-ai/internal/knowledge/crawler"
	"github.com/kubestack-ai/kubestack-ai/internal/knowledge/store"
	"github.com/spf13/cobra"
)

// newKbCmd creates the base `kb` command, which serves as a parent for all
// knowledge base management subcommands.
func newKbCmd(orchestrator interfaces.Orchestrator) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kb",
		Short: "Manage the knowledge base",
		Long:  `Provides commands to manage the KubeStack-AI knowledge base, such as adding new documents from URLs or files.`,
	}

	cmd.AddCommand(newKbAddCmd(orchestrator))

	return cmd
}

// newKbAddCmd creates the `kb add` command, which allows users to add new
// documents to the knowledge base from a URL or local file path.
func newKbAddCmd(orchestrator interfaces.Orchestrator) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add [uri]",
		Short: "Add a document to the knowledge base from a URL or file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			uri := args[0]
			fmt.Printf("Adding document from: %s\n", uri)

			// Determine if the URI is a URL or a local file path
			u, err := url.ParseRequestURI(uri)
			if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
				// Treat as a local file path
				// TODO: Implement file processing
				return fmt.Errorf("local file processing not yet implemented")
			}

			// Crawl the URL
			fmt.Println("Crawling document...")
			webCrawler, err := crawler.NewWebCrawler([]string{u.Host})
			if err != nil {
				return fmt.Errorf("failed to create web crawler: %w", err)
			}
			rawDoc, err := webCrawler.Crawl(uri)
			if err != nil {
				return fmt.Errorf("failed to crawl document: %w", err)
			}
			fmt.Printf("Crawled document: %s\n", rawDoc.Source)

			// Process the document into chunks
			fmt.Println("Processing document...")
			docProcessor, err := crawler.NewTextSplitter(1000, 100) // 1000 chars per chunk, 100 overlap
			if err != nil {
				return fmt.Errorf("failed to create doc processor: %w", err)
			}
			chunks, err := docProcessor.Process(rawDoc)
			if err != nil {
				return fmt.Errorf("failed to process document: %w", err)
			}
			fmt.Printf("Processed document into %d chunks\n", len(chunks))

			// Embed the chunks
			fmt.Println("Embedding document chunks...")
			embedder, err := orchestrator.GetEmbedder()
			if err != nil {
				return fmt.Errorf("failed to get embedder: %w", err)
			}

			chunkContents := make([]string, len(chunks))
			for i, chunk := range chunks {
				chunkContents[i] = chunk.Content
			}

			embeddings, err := embedder.EmbedDocuments(cmd.Context(), chunkContents)
			if err != nil {
				return fmt.Errorf("failed to embed document chunks: %w", err)
			}
			fmt.Printf("Successfully generated %d embeddings.\n", len(embeddings))

			// Ingest the data
			fmt.Println("Ingesting document and chunks into knowledge base...")
			docStore, err := orchestrator.GetDocumentStore()
			if err != nil {
				return fmt.Errorf("failed to get document store: %w", err)
			}
			vecStore, err := orchestrator.GetVectorStore()
			if err != nil {
				return fmt.Errorf("failed to get vector store: %w", err)
			}

			// Save the raw document
			docID, err := docStore.Add(cmd.Context(), rawDoc)
			if err != nil {
				return fmt.Errorf("failed to save raw document: %w", err)
			}

			// Save the vectorized chunks
			storeDocs := make([]store.StoreDocument, len(chunks))
			for i, chunk := range chunks {
				storeDocs[i] = store.StoreDocument{
					ID:       chunk.ID,
					Content:  chunk.Content,
					Vector:   embeddings[i],
					Metadata: map[string]interface{}{"SourceID": docID},
				}
			}
			if err := vecStore.AddDocuments(cmd.Context(), storeDocs); err != nil {
				return fmt.Errorf("failed to save document chunks: %w", err)
			}

			fmt.Println("Successfully added document to knowledge base.")
			return nil
		},
	}
	return cmd
}