package models

type RetrievalResult struct {
	DocID      string                 `json:"doc_id"`
	Content    string                 `json:"content"`
	Score      float64                `json:"score"`
	Source     string                 `json:"source"`
	Metadata   map[string]interface{} `json:"metadata"`
	ChunkIndex int                    `json:"chunk_index"`
	GraphCtx   string                 `json:"graph_ctx,omitempty"` // Context from Knowledge Graph
}

type Document struct {
	ID         string                 `json:"id"`
	Content    string                 `json:"content"`
	Metadata   map[string]interface{} `json:"metadata"`
	ChunkIndex int                    `json:"chunk_index,omitempty"`
}
