package logger

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/journal/internal/models"
)

// ProcessingLogger handles logging for journal entry processing stages
type ProcessingLogger struct {
	db      *sql.DB
	buffers map[string]*LogBuffer
	mu      sync.RWMutex
}

// LogBuffer temporarily stores logs before batch insertion
type LogBuffer struct {
	logs      []models.ProcessingLog
	lastFlush time.Time
	mu        sync.Mutex
}

// NewProcessingLogger creates a new processing logger instance
func NewProcessingLogger(db *sql.DB) *ProcessingLogger {
	pl := &ProcessingLogger{
		db:      db,
		buffers: make(map[string]*LogBuffer),
	}

	// Start background flusher
	go pl.backgroundFlusher()

	return pl
}

// LogDebug logs a debug message for a specific entry and stage
func (pl *ProcessingLogger) LogDebug(entryID string, stage models.ProcessingStage, message string, details map[string]interface{}) {
	pl.log(entryID, stage, "debug", message, details)
}

// LogInfo logs an info message for a specific entry and stage
func (pl *ProcessingLogger) LogInfo(entryID string, stage models.ProcessingStage, message string, details map[string]interface{}) {
	pl.log(entryID, stage, "info", message, details)
}

// LogWarn logs a warning message for a specific entry and stage
func (pl *ProcessingLogger) LogWarn(entryID string, stage models.ProcessingStage, message string, details map[string]interface{}) {
	pl.log(entryID, stage, "warn", message, details)
}

// LogError logs an error message for a specific entry and stage
func (pl *ProcessingLogger) LogError(entryID string, stage models.ProcessingStage, message string, details map[string]interface{}) {
	pl.log(entryID, stage, "error", message, details)
}

// log adds a log entry to the buffer
func (pl *ProcessingLogger) log(entryID string, stage models.ProcessingStage, level, message string, details map[string]interface{}) {
	logEntry := models.ProcessingLog{
		EntryID:   entryID,
		Stage:     stage,
		Level:     level,
		Message:   message,
		Details:   details,
		CreatedAt: time.Now(),
	}

	// Also log to stdout for debugging
	detailsJSON, _ := json.Marshal(details)
	log.Printf("[%s] Entry %s - Stage: %s - %s: %s %s",
		level, entryID, stage, level, message, string(detailsJSON))

	// Add to buffer
	pl.mu.Lock()
	buffer, exists := pl.buffers[entryID]
	if !exists {
		buffer = &LogBuffer{
			logs:      make([]models.ProcessingLog, 0, 10),
			lastFlush: time.Now(),
		}
		pl.buffers[entryID] = buffer
	}
	pl.mu.Unlock()

	buffer.mu.Lock()
	buffer.logs = append(buffer.logs, logEntry)
	shouldFlush := len(buffer.logs) >= 10 || time.Since(buffer.lastFlush) > 5*time.Second
	buffer.mu.Unlock()

	if shouldFlush {
		pl.flushBuffer(entryID)
	}
}

// UpdateStage updates the processing stage and logs the transition
func (pl *ProcessingLogger) UpdateStage(entryID string, stage models.ProcessingStage) error {
	// Log the stage transition
	pl.LogInfo(entryID, stage, fmt.Sprintf("Transitioning to stage: %s", stage), nil)

	// Update the database
	query := `
		UPDATE journal_entries 
		SET processing_stage = $1, updated_at = $2
		WHERE id = $3`

	_, err := pl.db.Exec(query, stage, time.Now(), entryID)
	if err != nil {
		pl.LogError(entryID, stage, "Failed to update processing stage", map[string]interface{}{
			"error": err.Error(),
		})
		return fmt.Errorf("failed to update processing stage: %w", err)
	}

	// Update processing timestamps
	if stage == models.StageAnalyzing {
		// Mark processing as started
		query = `UPDATE journal_entries SET processing_started_at = $1 WHERE id = $2`
		_, err = pl.db.Exec(query, time.Now(), entryID)
	} else if stage == models.StageCompleted || stage == models.StageFailed {
		// Mark processing as completed
		query = `UPDATE journal_entries SET processing_completed_at = $1 WHERE id = $2`
		_, err = pl.db.Exec(query, time.Now(), entryID)
	}

	return err
}

// SetError sets the processing error for an entry
func (pl *ProcessingLogger) SetError(entryID string, stage models.ProcessingStage, err error) error {
	pl.LogError(entryID, stage, "Processing failed", map[string]interface{}{
		"error": err.Error(),
	})

	// Update the database
	query := `
		UPDATE journal_entries 
		SET processing_stage = $1, processing_error = $2, processing_completed_at = $3
		WHERE id = $4`

	_, dbErr := pl.db.Exec(query, models.StageFailed, err.Error(), time.Now(), entryID)
	if dbErr != nil {
		return fmt.Errorf("failed to set processing error: %w", dbErr)
	}

	return nil
}

// GetLogs retrieves all logs for a specific entry
func (pl *ProcessingLogger) GetLogs(entryID string) ([]models.ProcessingLog, error) {
	// Flush any pending logs first
	pl.flushBuffer(entryID)

	query := `
		SELECT id, entry_id, stage, level, message, details, created_at
		FROM processing_logs
		WHERE entry_id = $1
		ORDER BY created_at ASC`

	rows, err := pl.db.Query(query, entryID)
	if err != nil {
		return nil, fmt.Errorf("failed to query logs: %w", err)
	}
	defer rows.Close()

	var logs []models.ProcessingLog
	for rows.Next() {
		var log models.ProcessingLog
		var detailsJSON []byte

		err := rows.Scan(&log.ID, &log.EntryID, &log.Stage, &log.Level,
			&log.Message, &detailsJSON, &log.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan log: %w", err)
		}

		if len(detailsJSON) > 0 {
			json.Unmarshal(detailsJSON, &log.Details)
		}

		logs = append(logs, log)
	}

	return logs, nil
}

// GetLogsByStage retrieves logs for a specific entry and stage
func (pl *ProcessingLogger) GetLogsByStage(entryID string, stage models.ProcessingStage) ([]models.ProcessingLog, error) {
	// Flush any pending logs first
	pl.flushBuffer(entryID)

	query := `
		SELECT id, entry_id, stage, level, message, details, created_at
		FROM processing_logs
		WHERE entry_id = $1 AND stage = $2
		ORDER BY created_at ASC`

	rows, err := pl.db.Query(query, entryID, stage)
	if err != nil {
		return nil, fmt.Errorf("failed to query logs: %w", err)
	}
	defer rows.Close()

	var logs []models.ProcessingLog
	for rows.Next() {
		var log models.ProcessingLog
		var detailsJSON []byte

		err := rows.Scan(&log.ID, &log.EntryID, &log.Stage, &log.Level,
			&log.Message, &detailsJSON, &log.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan log: %w", err)
		}

		if len(detailsJSON) > 0 {
			json.Unmarshal(detailsJSON, &log.Details)
		}

		logs = append(logs, log)
	}

	return logs, nil
}

// flushBuffer writes buffered logs to the database
func (pl *ProcessingLogger) flushBuffer(entryID string) {
	pl.mu.RLock()
	buffer, exists := pl.buffers[entryID]
	pl.mu.RUnlock()

	if !exists {
		return
	}

	buffer.mu.Lock()
	if len(buffer.logs) == 0 {
		buffer.mu.Unlock()
		return
	}

	logs := make([]models.ProcessingLog, len(buffer.logs))
	copy(logs, buffer.logs)
	buffer.logs = buffer.logs[:0]
	buffer.lastFlush = time.Now()
	buffer.mu.Unlock()

	// Batch insert logs
	tx, err := pl.db.Begin()
	if err != nil {
		log.Printf("Failed to begin transaction for log flush: %v", err)
		return
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO processing_logs (entry_id, stage, level, message, details, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`)
	if err != nil {
		log.Printf("Failed to prepare statement for log flush: %v", err)
		return
	}
	defer stmt.Close()

	for _, logEntry := range logs {
		detailsJSON, _ := json.Marshal(logEntry.Details)
		_, err = stmt.Exec(logEntry.EntryID, logEntry.Stage, logEntry.Level,
			logEntry.Message, detailsJSON, logEntry.CreatedAt)
		if err != nil {
			log.Printf("Failed to insert log: %v", err)
		}
	}

	if err = tx.Commit(); err != nil {
		log.Printf("Failed to commit log batch: %v", err)
	}
}

// backgroundFlusher periodically flushes all buffers
func (pl *ProcessingLogger) backgroundFlusher() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		pl.mu.RLock()
		entryIDs := make([]string, 0, len(pl.buffers))
		for entryID := range pl.buffers {
			entryIDs = append(entryIDs, entryID)
		}
		pl.mu.RUnlock()

		for _, entryID := range entryIDs {
			pl.flushBuffer(entryID)
		}
	}
}
