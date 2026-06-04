// wiki_engine.go provides the high-level Wiki layer wrapper.
package memory

// WikiManager wraps WikiFS with LLM-driven Ingest/Query/Lint.
type WikiManager struct{}

// NewWikiManager creates a manager.
func NewWikiManager() *WikiManager { return &WikiManager{} }

// Close is a no-op for now.
func (w *WikiManager) Close() error { return nil }

// IngestAsync queues a document for background wiki Ingest (Phase 3).
func (w *WikiManager) IngestAsync(kbID string, doc Document) {
	// Phase 3: enqueue to job channel, LLM-driven extraction
}
