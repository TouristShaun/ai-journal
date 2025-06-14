package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
	
	"github.com/journal/internal/db"
	"github.com/journal/internal/events"
	"github.com/journal/internal/logger"
	"github.com/journal/internal/mcp"
	"github.com/journal/internal/models"
	"github.com/journal/internal/ollama"
	"github.com/lib/pq"
	"github.com/pgvector/pgvector-go"
)

type JournalService struct {
	db               *db.DB
	processor        *ollama.Processor
	mcpClient        *mcp.Client
	broadcaster      *events.Broadcaster
	logger           *logger.ProcessingLogger
	failureAnalyzer  *FailureAnalyzer
}

func NewJournalService(database *db.DB, processor *ollama.Processor, mcpClient *mcp.Client, broadcaster *events.Broadcaster, logger *logger.ProcessingLogger) *JournalService {
	failureAnalyzer := NewFailureAnalyzer(processor, logger)
	
	return &JournalService{
		db:               database,
		processor:        processor,
		mcpClient:        mcpClient,
		broadcaster:      broadcaster,
		logger:           logger,
		failureAnalyzer:  failureAnalyzer,
	}
}

// CreateEntry creates a new journal entry with processing and embedding
func (s *JournalService) CreateEntry(content string) (*models.JournalEntry, error) {
	log.Printf("Creating new journal entry, content length: %d", len(content))
	
	// Create initial entry with minimal processing
	entry := models.JournalEntry{
		Content: content,
		ProcessedData: models.ProcessedData{
			Summary:   "Processing...",
			Entities:  []string{},
			Topics:    []string{},
			Sentiment: "neutral",
			Metadata:  make(map[string]any),
			ExtractedURLs: []models.ExtractedURL{},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		ProcessingStage: models.StageCreated,
	}
	
	// Convert minimal processed data to JSON
	processedJSON, err := json.Marshal(entry.ProcessedData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal processed data: %w", err)
	}
	
	// Insert into database immediately
	query := `
		INSERT INTO journal_entries (content, processed_data, created_at, updated_at, processing_stage)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`
	
	err = s.db.QueryRow(query, 
		entry.Content,
		processedJSON,
		entry.CreatedAt,
		entry.UpdatedAt,
		entry.ProcessingStage,
	).Scan(&entry.ID)
	
	if err != nil {
		return nil, fmt.Errorf("failed to insert entry: %w", err)
	}
	
	log.Printf("Created journal entry with ID: %s", entry.ID)
	
	// Log initial creation
	s.logger.LogInfo(entry.ID, models.StageCreated, "Journal entry created", map[string]interface{}{
		"content_length": len(content),
	})
	
	// Process asynchronously in background
	go func(entryID string, content string) {
		// Recover from panics in goroutine
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC in background processing for entry %s: %v", entryID, r)
				s.logger.SetError(entryID, models.StageAnalyzing, fmt.Errorf("panic: %v", r))
				// Send failure event
				s.broadcaster.SendEvent(events.EventEntryFailed, entryID, map[string]interface{}{
					"error": fmt.Sprintf("%v", r),
					"stage": models.StageFailed,
				})
			}
		}()
		
		log.Printf("Starting background processing for entry %s", entryID)
		
		// Transition to analyzing stage
		s.logger.UpdateStage(entryID, models.StageAnalyzing)
		s.broadcaster.SendEvent(events.EventEntryProcessing, entryID, map[string]interface{}{
			"stage": models.StageAnalyzing,
			"message": "Analyzing content with AI",
		})
		
		// Process content with Qwen
		s.logger.LogInfo(entryID, models.StageAnalyzing, "Starting AI analysis", nil)
		processedData, err := s.processor.ProcessJournalEntry(content)
		if err != nil {
			log.Printf("Failed to process entry %s: %v", entryID, err)
			s.logger.SetError(entryID, models.StageAnalyzing, err)
			// Send failure event
			s.broadcaster.SendEvent(events.EventEntryFailed, entryID, map[string]interface{}{
				"error": err.Error(),
				"stage": models.StageAnalyzing,
			})
			return
		}
		
		s.logger.LogInfo(entryID, models.StageAnalyzing, "AI analysis completed", map[string]interface{}{
			"entities_count": len(processedData.Entities),
			"topics_count":   len(processedData.Topics),
			"sentiment":      processedData.Sentiment,
		})
		
		// Create temporary entry for embedding generation
		tempEntry := models.JournalEntry{
			ID:            entryID,
			Content:       content,
			ProcessedData: *processedData,
		}
		
		// Fetch URLs if MCP client is available
		if s.mcpClient != nil && len(processedData.Metadata) > 0 {
			// Check if we have URLs to fetch
			urlsToFetch := 0
			for _, value := range processedData.Metadata {
				if urlData, ok := value.(map[string]interface{}); ok {
					if status, exists := urlData["status"]; exists && status == "pending_fetch" {
						urlsToFetch++
					}
				}
			}
			
			if urlsToFetch > 0 {
				// Transition to fetching URLs stage
				s.logger.UpdateStage(entryID, models.StageFetchingURLs)
				s.broadcaster.SendEvent(events.EventEntryProcessing, entryID, map[string]interface{}{
					"stage": models.StageFetchingURLs,
					"message": fmt.Sprintf("Fetching %d URLs", urlsToFetch),
				})
				
				s.logger.LogInfo(entryID, models.StageFetchingURLs, "Starting URL fetching", map[string]interface{}{
					"urls_count": urlsToFetch,
				})
				
				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				defer cancel()
				
				if err := s.mcpClient.FetchURLsForEntry(ctx, &tempEntry); err != nil {
					log.Printf("Failed to fetch URLs for entry %s: %v", entryID, err)
					s.logger.LogWarn(entryID, models.StageFetchingURLs, "URL fetching failed", map[string]interface{}{
						"error": err.Error(),
					})
				} else {
					s.logger.LogInfo(entryID, models.StageFetchingURLs, "URL fetching completed", map[string]interface{}{
						"fetched_count": len(tempEntry.ProcessedData.ExtractedURLs),
					})
				}
			}
		}
		
		// Transition to embedding generation stage
		s.logger.UpdateStage(entryID, models.StageGeneratingEmbeddings)
		s.broadcaster.SendEvent(events.EventEntryProcessing, entryID, map[string]interface{}{
			"stage": models.StageGeneratingEmbeddings,
			"message": "Generating semantic embeddings",
		})
		
		// Generate embedding
		s.logger.LogInfo(entryID, models.StageGeneratingEmbeddings, "Starting embedding generation", nil)
		embedding, err := s.processor.CreateEmbedding(tempEntry)
		if err != nil {
			log.Printf("Failed to create embedding for entry %s: %v", entryID, err)
			s.logger.SetError(entryID, models.StageGeneratingEmbeddings, err)
			// Send failure event
			s.broadcaster.SendEvent(events.EventEntryFailed, entryID, map[string]interface{}{
				"error": err.Error(),
				"stage": models.StageGeneratingEmbeddings,
			})
			return
		}
		
		s.logger.LogInfo(entryID, models.StageGeneratingEmbeddings, "Embedding generated", map[string]interface{}{
			"embedding_dims": len(embedding),
		})
		
		// Update processed data JSON
		processedJSON, err := json.Marshal(tempEntry.ProcessedData)
		if err != nil {
			log.Printf("Failed to marshal processed data for entry %s: %v", entryID, err)
			return
		}
		
		// Update the entry with processed data and embedding
		updateQuery := `
			UPDATE journal_entries 
			SET processed_data = $1, embedding = $2, updated_at = $3, 
			    processing_stage = $4, processing_completed_at = $5
			WHERE id = $6`
		
		_, err = s.db.Exec(updateQuery,
			processedJSON,
			pgvector.NewVector(embedding),
			time.Now(),
			models.StageCompleted,
			time.Now(),
			entryID,
		)
		
		if err != nil {
			log.Printf("Failed to update entry %s with processed data: %v", entryID, err)
			s.logger.SetError(entryID, models.StageGeneratingEmbeddings, err)
			// Send failure event
			s.broadcaster.SendEvent(events.EventEntryFailed, entryID, map[string]interface{}{
				"error": err.Error(),
				"stage": models.StageGeneratingEmbeddings,
			})
			return
		}
		
		// Update to completed stage
		s.logger.UpdateStage(entryID, models.StageCompleted)
		s.logger.LogInfo(entryID, models.StageCompleted, "Processing completed successfully", map[string]interface{}{
			"total_entities": len(tempEntry.ProcessedData.Entities),
			"total_topics":   len(tempEntry.ProcessedData.Topics),
			"total_urls":     len(tempEntry.ProcessedData.ExtractedURLs),
		})
		
		log.Printf("Successfully processed entry %s", entryID)
		
		// Send success event with updated data
		s.broadcaster.SendEvent(events.EventEntryProcessed, entryID, map[string]interface{}{
			"processed_data": tempEntry.ProcessedData,
			"updated_at":     time.Now(),
			"stage":          models.StageCompleted,
		})
	}(entry.ID, content)
	
	return &entry, nil
}

// UpdateEntry updates an existing entry and preserves the original
func (s *JournalService) UpdateEntry(id string, content string) (*models.JournalEntry, error) {
	// First, get the original entry
	original, err := s.GetEntry(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get original entry: %w", err)
	}
	
	// Process new content
	processedData, err := s.processor.ProcessJournalEntry(content)
	if err != nil {
		return nil, fmt.Errorf("failed to process updated entry: %w", err)
	}
	
	// Create new entry as an update
	newEntry := models.JournalEntry{
		Content:         content,
		ProcessedData:   *processedData,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		IsFavorite:      original.IsFavorite,
		CollectionIDs:   original.CollectionIDs,
		OriginalEntryID: &id,
	}
	
	// Generate new embedding
	embedding, err := s.processor.CreateEmbedding(newEntry)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding: %w", err)
	}
	newEntry.Embedding = pgvector.NewVector(embedding)
	
	// Convert processed data to JSON
	processedJSON, err := json.Marshal(processedData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal processed data: %w", err)
	}
	
	// Insert new version
	query := `
		INSERT INTO journal_entries (content, processed_data, embedding, created_at, updated_at, is_favorite, original_entry_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`
	
	err = s.db.QueryRow(query,
		newEntry.Content,
		processedJSON,
		newEntry.Embedding,
		newEntry.CreatedAt,
		newEntry.UpdatedAt,
		newEntry.IsFavorite,
		newEntry.OriginalEntryID,
	).Scan(&newEntry.ID)
	
	if err != nil {
		return nil, fmt.Errorf("failed to insert updated entry: %w", err)
	}
	
	// Copy collection associations
	if len(original.CollectionIDs) > 0 {
		for _, collID := range original.CollectionIDs {
			_, err = s.db.Exec(
				"INSERT INTO journal_collection (journal_id, collection_id) VALUES ($1, $2)",
				newEntry.ID, collID,
			)
			if err != nil {
				log.Printf("Failed to copy collection association: %v", err)
			}
		}
	}
	
	return &newEntry, nil
}

// GetEntry retrieves a single journal entry
func (s *JournalService) GetEntry(id string) (*models.JournalEntry, error) {
	query := `
		SELECT 
			je.id, je.content, je.processed_data, je.created_at, je.updated_at,
			je.is_favorite, je.original_entry_id,
			je.processing_stage, je.processing_started_at, je.processing_completed_at, je.processing_error,
			COALESCE(array_agg(jc.collection_id) FILTER (WHERE jc.collection_id IS NOT NULL), '{}') as collection_ids
		FROM journal_entries je
		LEFT JOIN journal_collection jc ON je.id = jc.journal_id
		WHERE je.id = $1
		GROUP BY je.id`
	
	var entry models.JournalEntry
	var processedJSON []byte
	
	err := s.db.QueryRow(query, id).Scan(
		&entry.ID,
		&entry.Content,
		&processedJSON,
		&entry.CreatedAt,
		&entry.UpdatedAt,
		&entry.IsFavorite,
		&entry.OriginalEntryID,
		&entry.ProcessingStage,
		&entry.ProcessingStartedAt,
		&entry.ProcessingCompletedAt,
		&entry.ProcessingError,
		pq.Array(&entry.CollectionIDs),
	)
	
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("entry not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get entry: %w", err)
	}
	
	if err := json.Unmarshal(processedJSON, &entry.ProcessedData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal processed data: %w", err)
	}
	
	return &entry, nil
}

// SearchParams holds search parameters
type SearchParams struct {
	Query         string   `json:"query"`
	IsFavorite    *bool    `json:"is_favorite"`
	CollectionIDs []string `json:"collection_ids"`
	StartDate     *time.Time `json:"start_date"`
	EndDate       *time.Time `json:"end_date"`
	Limit         int      `json:"limit"`
	Offset        int      `json:"offset"`
}

// ClassicSearch performs traditional keyword and filter based search
func (s *JournalService) ClassicSearch(params SearchParams) ([]models.JournalEntry, error) {
	query := `
		SELECT DISTINCT
			je.id, je.content, je.processed_data, je.created_at, je.updated_at,
			je.is_favorite, je.original_entry_id,
			je.processing_stage, je.processing_started_at, je.processing_completed_at, je.processing_error,
			COALESCE(array_agg(jc.collection_id) FILTER (WHERE jc.collection_id IS NOT NULL), '{}') as collection_ids
		FROM journal_entries je
		LEFT JOIN journal_collection jc ON je.id = jc.journal_id
		WHERE 1=1`
	
	args := []interface{}{}
	argCount := 0
	
	// Add text search
	if params.Query != "" {
		argCount++
		query += fmt.Sprintf(" AND je.tsv @@ plainto_tsquery('english', $%d)", argCount)
		args = append(args, params.Query)
	}
	
	// Add favorite filter
	if params.IsFavorite != nil {
		argCount++
		query += fmt.Sprintf(" AND je.is_favorite = $%d", argCount)
		args = append(args, *params.IsFavorite)
	}
	
	// Add collection filter
	if len(params.CollectionIDs) > 0 {
		argCount++
		query += fmt.Sprintf(" AND je.id IN (SELECT journal_id FROM journal_collection WHERE collection_id = ANY($%d))", argCount)
		args = append(args, pq.Array(params.CollectionIDs))
	}
	
	// Add date filters
	if params.StartDate != nil {
		argCount++
		query += fmt.Sprintf(" AND je.created_at >= $%d", argCount)
		args = append(args, *params.StartDate)
	}
	
	if params.EndDate != nil {
		argCount++
		query += fmt.Sprintf(" AND je.created_at <= $%d", argCount)
		args = append(args, *params.EndDate)
	}
	
	// Add grouping and ordering
	query += " GROUP BY je.id ORDER BY je.created_at DESC"
	
	// Add pagination
	if params.Limit > 0 {
		argCount++
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, params.Limit)
	}
	
	if params.Offset > 0 {
		argCount++
		query += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, params.Offset)
	}
	
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search entries: %w", err)
	}
	defer rows.Close()
	
	return s.scanEntries(rows)
}

// VectorSearch performs semantic search using embeddings
func (s *JournalService) VectorSearch(query string, limit int) ([]models.JournalEntry, error) {
	// Generate embedding for query
	embedding, err := s.processor.CreateEmbedding(models.JournalEntry{
		Content: query,
		ProcessedData: models.ProcessedData{},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create query embedding: %w", err)
	}
	
	// Perform vector similarity search
	searchQuery := `
		SELECT 
			je.id, je.content, je.processed_data, je.created_at, je.updated_at,
			je.is_favorite, je.original_entry_id,
			je.processing_stage, je.processing_started_at, je.processing_completed_at, je.processing_error,
			COALESCE(array_agg(jc.collection_id) FILTER (WHERE jc.collection_id IS NOT NULL), '{}') as collection_ids,
			1 - (je.embedding <=> $1) as similarity
		FROM journal_entries je
		LEFT JOIN journal_collection jc ON je.id = jc.journal_id
		WHERE je.embedding IS NOT NULL
		GROUP BY je.id
		ORDER BY je.embedding <=> $1
		LIMIT $2`
	
	rows, err := s.db.Query(searchQuery, pgvector.NewVector(embedding), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to perform vector search: %w", err)
	}
	defer rows.Close()
	
	entries := []models.JournalEntry{}
	for rows.Next() {
		var entry models.JournalEntry
		var processedJSON []byte
		var similarity float32
		
		err := rows.Scan(
			&entry.ID,
			&entry.Content,
			&processedJSON,
			&entry.CreatedAt,
			&entry.UpdatedAt,
			&entry.IsFavorite,
			&entry.OriginalEntryID,
			&entry.ProcessingStage,
			&entry.ProcessingStartedAt,
			&entry.ProcessingCompletedAt,
			&entry.ProcessingError,
			pq.Array(&entry.CollectionIDs),
			&similarity,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan entry: %w", err)
		}
		
		if err := json.Unmarshal(processedJSON, &entry.ProcessedData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal processed data: %w", err)
		}
		
		// Add similarity score to metadata
		if entry.ProcessedData.Metadata == nil {
			entry.ProcessedData.Metadata = make(map[string]any)
		}
		entry.ProcessedData.Metadata["similarity_score"] = similarity
		
		entries = append(entries, entry)
	}
	
	return entries, nil
}

// HybridSearch combines vector and traditional search
func (s *JournalService) HybridSearch(params SearchParams) ([]models.JournalEntry, error) {
	// If there's a query, perform vector search first
	vectorResults := []models.JournalEntry{}
	if params.Query != "" {
		var err error
		vectorResults, err = s.VectorSearch(params.Query, 20)
		if err != nil {
			log.Printf("Vector search failed, falling back to classic: %v", err)
		}
	}
	
	// Also perform classic search
	classicResults, err := s.ClassicSearch(params)
	if err != nil {
		return nil, err
	}
	
	// Merge results intelligently
	resultMap := make(map[string]models.JournalEntry)
	
	// Add vector results with higher priority
	for _, entry := range vectorResults {
		resultMap[entry.ID] = entry
	}
	
	// Add classic results
	for _, entry := range classicResults {
		if _, exists := resultMap[entry.ID]; !exists {
			resultMap[entry.ID] = entry
		}
	}
	
	// Convert back to slice
	results := make([]models.JournalEntry, 0, len(resultMap))
	for _, entry := range resultMap {
		results = append(results, entry)
	}
	
	return results, nil
}

// Helper function to scan multiple entries
func (s *JournalService) scanEntries(rows *sql.Rows) ([]models.JournalEntry, error) {
	entries := []models.JournalEntry{}
	
	for rows.Next() {
		var entry models.JournalEntry
		var processedJSON []byte
		
		err := rows.Scan(
			&entry.ID,
			&entry.Content,
			&processedJSON,
			&entry.CreatedAt,
			&entry.UpdatedAt,
			&entry.IsFavorite,
			&entry.OriginalEntryID,
			&entry.ProcessingStage,
			&entry.ProcessingStartedAt,
			&entry.ProcessingCompletedAt,
			&entry.ProcessingError,
			pq.Array(&entry.CollectionIDs),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan entry: %w", err)
		}
		
		if err := json.Unmarshal(processedJSON, &entry.ProcessedData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal processed data: %w", err)
		}
		
		entries = append(entries, entry)
	}
	
	return entries, nil
}

// ToggleFavorite toggles the favorite status of an entry
func (s *JournalService) ToggleFavorite(id string) error {
	_, err := s.db.Exec(
		"UPDATE journal_entries SET is_favorite = NOT is_favorite WHERE id = $1",
		id,
	)
	return err
}

// Collection management methods
func (s *JournalService) CreateCollection(name, description string) (*models.Collection, error) {
	collection := &models.Collection{
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	err := s.db.QueryRow(
		"INSERT INTO collections (name, description) VALUES ($1, $2) RETURNING id",
		name, description,
	).Scan(&collection.ID)
	
	if err != nil {
		return nil, fmt.Errorf("failed to create collection: %w", err)
	}
	
	return collection, nil
}

func (s *JournalService) GetCollections() ([]models.Collection, error) {
	rows, err := s.db.Query("SELECT id, name, description, created_at, updated_at FROM collections ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	collections := []models.Collection{}
	for rows.Next() {
		var c models.Collection
		if err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		collections = append(collections, c)
	}
	
	return collections, nil
}

func (s *JournalService) AddToCollection(entryID, collectionID string) error {
	_, err := s.db.Exec(
		"INSERT INTO journal_collection (journal_id, collection_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
		entryID, collectionID,
	)
	return err
}

func (s *JournalService) RemoveFromCollection(entryID, collectionID string) error {
	_, err := s.db.Exec(
		"DELETE FROM journal_collection WHERE journal_id = $1 AND collection_id = $2",
		entryID, collectionID,
	)
	return err
}

// GetProcessingLogs retrieves processing logs for a specific entry
func (s *JournalService) GetProcessingLogs(entryID string) ([]models.ProcessingLog, error) {
	return s.logger.GetLogs(entryID)
}

// AnalyzeFailure analyzes why a journal entry processing failed
func (s *JournalService) AnalyzeFailure(entryID string) (*FailureAnalysis, error) {
	// Get the entry
	entry, err := s.GetEntry(entryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get entry: %w", err)
	}
	
	// Check if entry actually failed
	if entry.ProcessingStage != models.StageFailed {
		return nil, fmt.Errorf("entry %s has not failed (current stage: %s)", entryID, entry.ProcessingStage)
	}
	
	// Analyze the failure
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	return s.failureAnalyzer.AnalyzeFailure(ctx, entryID, entry)
}