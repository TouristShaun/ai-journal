package models

import (
	"database/sql/driver"
	"fmt"
	"time"
	"github.com/pgvector/pgvector-go"
)

type JournalEntry struct {
	ID                     string            `json:"id" db:"id"`
	Content                string            `json:"content" db:"content"`
	ProcessedData          ProcessedData     `json:"processed_data" db:"processed_data"`
	Embedding              pgvector.Vector   `json:"-" db:"embedding"`
	CreatedAt              time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time         `json:"updated_at" db:"updated_at"`
	IsFavorite             bool              `json:"is_favorite" db:"is_favorite"`
	CollectionIDs          []string          `json:"collection_ids" db:"collection_ids"`
	OriginalEntryID        *string           `json:"original_entry_id,omitempty" db:"original_entry_id"`
	ProcessingStage        ProcessingStage   `json:"processing_stage" db:"processing_stage"`
	ProcessingStartedAt    *time.Time        `json:"processing_started_at,omitempty" db:"processing_started_at"`
	ProcessingCompletedAt  *time.Time        `json:"processing_completed_at,omitempty" db:"processing_completed_at"`
	ProcessingError        *string           `json:"processing_error,omitempty" db:"processing_error"`
}

type ProcessedData struct {
	Summary         string          `json:"summary"`
	ExtractedURLs   []ExtractedURL  `json:"extracted_urls"`
	Entities        []string        `json:"entities"`
	Topics          []string        `json:"topics"`
	Sentiment       string          `json:"sentiment"`
	Metadata        map[string]any  `json:"metadata"`
}

type ExtractedURL struct {
	URL             string          `json:"url"`
	Title           string          `json:"title"`
	Content         string          `json:"content"`
	ExtractedAt     time.Time       `json:"extracted_at"`
	Source          string          `json:"source"`
}

type Collection struct {
	ID              string          `json:"id" db:"id"`
	Name            string          `json:"name" db:"name"`
	Description     string          `json:"description" db:"description"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at" db:"updated_at"`
}

// ProcessingStage represents the current stage of journal entry processing
type ProcessingStage string

const (
	StageCreated             ProcessingStage = "created"
	StageAnalyzing           ProcessingStage = "analyzing"
	StageFetchingURLs        ProcessingStage = "fetching_urls"
	StageGeneratingEmbeddings ProcessingStage = "generating_embeddings"
	StageCompleted           ProcessingStage = "completed"
	StageFailed              ProcessingStage = "failed"
)

// Scan implements the sql.Scanner interface
func (s *ProcessingStage) Scan(value interface{}) error {
	if value == nil {
		*s = StageCreated
		return nil
	}
	switch v := value.(type) {
	case string:
		*s = ProcessingStage(v)
	case []byte:
		*s = ProcessingStage(v)
	default:
		return fmt.Errorf("cannot scan type %T into ProcessingStage", value)
	}
	return nil
}

// Value implements the driver.Valuer interface
func (s ProcessingStage) Value() (driver.Value, error) {
	return string(s), nil
}

// ProcessingLog represents a log entry for a specific processing stage
type ProcessingLog struct {
	ID        string                 `json:"id" db:"id"`
	EntryID   string                 `json:"entry_id" db:"entry_id"`
	Stage     ProcessingStage        `json:"stage" db:"stage"`
	Level     string                 `json:"level" db:"level"` // debug, info, warn, error
	Message   string                 `json:"message" db:"message"`
	Details   map[string]interface{} `json:"details" db:"details"`
	CreatedAt time.Time              `json:"created_at" db:"created_at"`
}