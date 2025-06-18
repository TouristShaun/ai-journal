package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
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
	db              *db.DB
	processor       *ollama.Processor
	mcpClient       *mcp.Client
	broadcaster     *events.Broadcaster
	logger          *logger.ProcessingLogger
	failureAnalyzer *FailureAnalyzer
}

func NewJournalService(database *db.DB, processor *ollama.Processor, mcpClient *mcp.Client, broadcaster *events.Broadcaster, logger *logger.ProcessingLogger) *JournalService {
	failureAnalyzer := NewFailureAnalyzer(processor, logger)

	return &JournalService{
		db:              database,
		processor:       processor,
		mcpClient:       mcpClient,
		broadcaster:     broadcaster,
		logger:          logger,
		failureAnalyzer: failureAnalyzer,
	}
}

// CreateEntry creates a new journal entry with processing and embedding
func (s *JournalService) CreateEntry(content string) (*models.JournalEntry, error) {
	log.Printf("Creating new journal entry, content length: %d", len(content))

	// Create initial entry with minimal processing
	now := time.Now()
	entry := models.JournalEntry{
		Content: content,
		ProcessedData: models.ProcessedData{
			Summary:       "Processing...",
			Entities:      []string{},
			Topics:        []string{},
			Sentiment:     "neutral",
			Metadata:      make(map[string]any),
			ExtractedURLs: []models.ExtractedURL{},
		},
		CreatedAt:           now,
		UpdatedAt:           now,
		ProcessingStage:     models.StageCreated,
		ProcessingStartedAt: &now,
	}

	// Convert minimal processed data to JSON
	processedJSON, err := json.Marshal(entry.ProcessedData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal processed data: %w", err)
	}

	// Insert into database immediately
	query := `
		INSERT INTO journal_entries (content, processed_data, created_at, updated_at, processing_stage, processing_started_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`

	err = s.db.QueryRow(query,
		entry.Content,
		processedJSON,
		entry.CreatedAt,
		entry.UpdatedAt,
		entry.ProcessingStage,
		entry.ProcessingStartedAt,
	).Scan(&entry.ID)

	if err != nil {
		return nil, fmt.Errorf("failed to insert entry: %w", err)
	}

	log.Printf("Created journal entry with ID: %s", entry.ID)

	// Log initial creation
	s.logger.LogInfo(entry.ID, models.StageCreated, "Journal entry created", map[string]interface{}{
		"content_length": len(content),
	})

	// Send created event to all connected clients
	s.broadcaster.SendEvent(events.EventEntryCreated, entry.ID, map[string]interface{}{
		"entry": entry,
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
			"stage":   models.StageAnalyzing,
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

		// Fetch URLs if any
		if s.mcpClient != nil && len(processedData.ExtractedURLs) > 0 {
			// Transition to fetching URLs stage
			s.logger.UpdateStage(entryID, models.StageFetchingURLs)
			s.broadcaster.SendEvent(events.EventEntryProcessing, entryID, map[string]interface{}{
				"stage":   models.StageFetchingURLs,
				"message": fmt.Sprintf("Fetching %d URLs", len(processedData.ExtractedURLs)),
			})

			s.logger.LogInfo(entryID, models.StageFetchingURLs, "Starting URL fetching", map[string]interface{}{
				"urls_count": len(processedData.ExtractedURLs),
			})

			// Fetch each URL
			for i, urlInfo := range processedData.ExtractedURLs {
				s.logger.LogInfo(entryID, models.StageFetchingURLs, fmt.Sprintf("Fetching URL %d/%d", i+1, len(processedData.ExtractedURLs)), map[string]interface{}{
					"url": urlInfo.URL,
				})

				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				fetchedContent, err := s.mcpClient.FetchURL(ctx, urlInfo.URL, urlInfo.Title) // Using Title as reason
				cancel()

				if err != nil {
					log.Printf("Failed to fetch URL %s: %v", urlInfo.URL, err)
					s.logger.LogInfo(entryID, models.StageFetchingURLs, fmt.Sprintf("Failed to fetch URL: %v", err), map[string]interface{}{
						"url": urlInfo.URL,
					})
					continue
				}

				// Update the entry with fetched content
				tempEntry.ProcessedData.ExtractedURLs[i].Title = fetchedContent.Title
				tempEntry.ProcessedData.ExtractedURLs[i].Content = fetchedContent.Content
			}

			s.logger.LogInfo(entryID, models.StageFetchingURLs, "URL fetching completed", map[string]interface{}{
				"fetched_count": len(tempEntry.ProcessedData.ExtractedURLs),
			})
		}

		// Transition to embedding generation stage
		s.logger.UpdateStage(entryID, models.StageGeneratingEmbeddings)
		s.broadcaster.SendEvent(events.EventEntryProcessing, entryID, map[string]interface{}{
			"stage":   models.StageGeneratingEmbeddings,
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

		// Fetch the complete updated entry to send in the event
		updatedEntry, err := s.GetEntry(entryID)
		if err != nil {
			log.Printf("Failed to fetch updated entry for event: %v", err)
			// Create a complete entry structure even if fetch fails
			tempEntry.UpdatedAt = time.Now()
			tempEntry.ProcessingStage = models.StageCompleted
			tempEntry.ProcessingCompletedAt = &tempEntry.UpdatedAt

			// Send event with reconstructed entry data
			s.broadcaster.SendEvent(events.EventEntryProcessed, entryID, map[string]interface{}{
				"entry": &tempEntry,
				"stage": models.StageCompleted,
			})
		} else {
			// Send success event with full updated entry
			s.broadcaster.SendEvent(events.EventEntryProcessed, entryID, map[string]interface{}{
				"entry": updatedEntry,
				"stage": models.StageCompleted,
			})
		}
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

	// Send updated event
	s.broadcaster.SendEvent(events.EventEntryUpdated, newEntry.ID, map[string]interface{}{
		"entry":       &newEntry,
		"original_id": id,
	})

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
	Query         string     `json:"query"`
	IsFavorite    *bool      `json:"is_favorite"`
	CollectionIDs []string   `json:"collection_ids"`
	StartDate     *time.Time `json:"start_date"`
	EndDate       *time.Time `json:"end_date"`
	Limit         int        `json:"limit"`
	Offset        int        `json:"offset"`
	SemanticMode  string     `json:"semantic_mode"` // similar, explore, contrast
	HybridMode    string     `json:"hybrid_mode"`   // balanced, semantic_boost, precision, discovery
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

// VectorSearch performs semantic search using embeddings with different modes
func (s *JournalService) VectorSearch(params SearchParams) ([]models.JournalEntry, error) {
	// Validate query is not empty
	if params.Query == "" {
		return nil, fmt.Errorf("query cannot be empty for vector search")
	}

	// Generate embedding for query
	embedding, err := s.processor.CreateEmbedding(models.JournalEntry{
		Content:       params.Query,
		ProcessedData: models.ProcessedData{},
	})
	if err != nil {
		// Check if Ollama is running
		if strings.Contains(err.Error(), "connection refused") {
			return nil, fmt.Errorf("Ollama service is not running. Please start it with 'ollama serve'")
		}
		return nil, fmt.Errorf("failed to create query embedding: %w", err)
	}

	// Default semantic mode
	if params.SemanticMode == "" {
		params.SemanticMode = "similar"
	}

	// Default limit
	if params.Limit == 0 {
		params.Limit = 20
	}

	// Build base query
	baseQuery := `
		SELECT 
			je.id, je.content, je.processed_data, je.created_at, je.updated_at,
			je.is_favorite, je.original_entry_id,
			je.processing_stage, je.processing_started_at, je.processing_completed_at, je.processing_error,
			COALESCE(array_agg(jc.collection_id) FILTER (WHERE jc.collection_id IS NOT NULL), '{}') as collection_ids,
			1 - (je.embedding <=> $1) as similarity
		FROM journal_entries je
		LEFT JOIN journal_collection jc ON je.id = jc.journal_id
		WHERE je.embedding IS NOT NULL`

	// Add filters
	args := []interface{}{pgvector.NewVector(embedding)}
	argCount := 1

	// Collection filter
	if len(params.CollectionIDs) > 0 {
		argCount++
		baseQuery += fmt.Sprintf(" AND je.id IN (SELECT journal_id FROM journal_collection WHERE collection_id = ANY($%d))", argCount)
		args = append(args, pq.Array(params.CollectionIDs))
	}

	// Favorite filter
	if params.IsFavorite != nil {
		argCount++
		baseQuery += fmt.Sprintf(" AND je.is_favorite = $%d", argCount)
		args = append(args, *params.IsFavorite)
	}

	// Date filters
	if params.StartDate != nil {
		argCount++
		baseQuery += fmt.Sprintf(" AND je.created_at >= $%d", argCount)
		args = append(args, *params.StartDate)
	}

	if params.EndDate != nil {
		argCount++
		baseQuery += fmt.Sprintf(" AND je.created_at <= $%d", argCount)
		args = append(args, *params.EndDate)
	}

	baseQuery += " GROUP BY je.id"

	// Apply semantic mode
	var searchQuery string
	switch params.SemanticMode {
	case "contrast":
		// Find contrasting/opposite entries by using inverse similarity
		searchQuery = baseQuery + " ORDER BY je.embedding <=> $1 DESC"
	case "explore":
		// Find conceptually related entries with medium similarity
		searchQuery = baseQuery + " HAVING 1 - (je.embedding <=> $1) BETWEEN 0.3 AND 0.7 ORDER BY RANDOM()"
	default: // "similar"
		// Standard similarity search
		searchQuery = baseQuery + " ORDER BY je.embedding <=> $1"
	}

	// Add limit
	argCount++
	searchQuery += fmt.Sprintf(" LIMIT $%d", argCount)
	args = append(args, params.Limit)

	rows, err := s.db.Query(searchQuery, args...)
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
	// Default hybrid mode
	if params.HybridMode == "" {
		params.HybridMode = "balanced"
	}

	// If there's a query, perform vector search first
	vectorResults := []models.JournalEntry{}
	if params.Query != "" {
		// Create vector params with a higher limit for hybrid
		vectorParams := params
		vectorParams.Limit = 30

		var err error
		vectorResults, err = s.VectorSearch(vectorParams)
		if err != nil {
			log.Printf("Vector search failed, falling back to classic: %v", err)
		}
	}

	// Also perform classic search
	classicResults, err := s.ClassicSearch(params)
	if err != nil {
		return nil, err
	}

	// Merge results intelligently based on hybrid mode
	type scoredEntry struct {
		entry models.JournalEntry
		score float32
	}

	resultScores := make(map[string]scoredEntry)

	// Calculate weights based on hybrid mode
	var vectorWeight, classicWeight float32
	switch params.HybridMode {
	case "semantic_boost":
		vectorWeight, classicWeight = 0.7, 0.3
	case "precision":
		vectorWeight, classicWeight = 0.3, 0.7
	case "discovery":
		vectorWeight, classicWeight = 0.8, 0.2
	default: // "balanced"
		vectorWeight, classicWeight = 0.5, 0.5
	}

	// Add vector results with their similarity scores
	for i, entry := range vectorResults {
		similarity := float32(0.0)
		if score, ok := entry.ProcessedData.Metadata["similarity_score"].(float32); ok {
			similarity = score
		}
		// Normalize rank to score (higher rank = lower score)
		rankScore := 1.0 - (float32(i) / float32(len(vectorResults)))
		finalScore := (similarity*0.7 + rankScore*0.3) * vectorWeight

		resultScores[entry.ID] = scoredEntry{
			entry: entry,
			score: finalScore,
		}
	}

	// Add classic results with rank-based scoring
	for i, entry := range classicResults {
		rankScore := 1.0 - (float32(i) / float32(len(classicResults)))
		finalScore := rankScore * classicWeight

		if existing, exists := resultScores[entry.ID]; exists {
			// Entry exists in both results, combine scores
			resultScores[entry.ID] = scoredEntry{
				entry: existing.entry,
				score: existing.score + finalScore,
			}
		} else {
			// New entry from classic search
			resultScores[entry.ID] = scoredEntry{
				entry: entry,
				score: finalScore,
			}
		}
	}

	// Sort by final score
	scored := make([]scoredEntry, 0, len(resultScores))
	for _, se := range resultScores {
		scored = append(scored, se)
	}

	// Sort by score descending
	for i := 0; i < len(scored); i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score > scored[i].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	// For discovery mode, add some randomization
	if params.HybridMode == "discovery" && len(scored) > 10 {
		// Shuffle top 10 results slightly
		for i := 0; i < 5; i++ {
			j := i + 1 + (i % 5)
			if j < 10 {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	// Convert back to entries slice
	results := make([]models.JournalEntry, 0, len(scored))
	limit := params.Limit
	if limit == 0 {
		limit = 20
	}

	for i, se := range scored {
		if i >= limit {
			break
		}
		// Add hybrid score to metadata
		if se.entry.ProcessedData.Metadata == nil {
			se.entry.ProcessedData.Metadata = make(map[string]any)
		}
		se.entry.ProcessedData.Metadata["hybrid_score"] = se.score
		results = append(results, se.entry)
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
	if err != nil {
		return err
	}

	// Get the updated entry to send in the event
	entry, err := s.GetEntry(entryID)
	if err != nil {
		log.Printf("Failed to get entry after adding to collection: %v", err)
		// Still return success since the collection was added
		return nil
	}

	// Send updated event with the full entry data
	s.broadcaster.SendEvent(events.EventEntryUpdated, entryID, map[string]interface{}{
		"entry":             entry,
		"collection_action": "added",
		"collection_id":     collectionID,
	})

	return nil
}

func (s *JournalService) RemoveFromCollection(entryID, collectionID string) error {
	_, err := s.db.Exec(
		"DELETE FROM journal_collection WHERE journal_id = $1 AND collection_id = $2",
		entryID, collectionID,
	)
	if err != nil {
		return err
	}

	// Get the updated entry to send in the event
	entry, err := s.GetEntry(entryID)
	if err != nil {
		log.Printf("Failed to get entry after removing from collection: %v", err)
		// Still return success since the collection was removed
		return nil
	}

	// Send updated event with the full entry data
	s.broadcaster.SendEvent(events.EventEntryUpdated, entryID, map[string]interface{}{
		"entry":             entry,
		"collection_action": "removed",
		"collection_id":     collectionID,
	})

	return nil
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

// RetryProcessing retries processing for a failed entry
func (s *JournalService) RetryProcessing(entryID string) error {
	// Get the entry
	entry, err := s.GetEntry(entryID)
	if err != nil {
		return fmt.Errorf("failed to get entry: %w", err)
	}

	// Check if entry is in a failed state or stuck in processing
	if entry.ProcessingStage != models.StageFailed && entry.ProcessingStage != models.StageCompleted {
		// Check if it's been stuck for more than 5 minutes
		if entry.ProcessingStartedAt != nil {
			elapsed := time.Since(*entry.ProcessingStartedAt)
			if elapsed < 5*time.Minute {
				return fmt.Errorf("entry %s is currently being processed (stage: %s)", entryID, entry.ProcessingStage)
			}
		}
	}

	// Reset processing state
	now := time.Now()
	_, err = s.db.Exec(`
		UPDATE journal_entries 
		SET processing_stage = $1, 
		    processing_started_at = $2,
		    processing_completed_at = NULL,
		    processing_error = NULL
		WHERE id = $3`,
		models.StageCreated,
		now,
		entryID,
	)
	if err != nil {
		return fmt.Errorf("failed to reset processing state: %w", err)
	}

	// Log retry attempt
	s.logger.LogInfo(entryID, models.StageCreated, "Retrying processing", map[string]interface{}{
		"previous_stage": entry.ProcessingStage,
		"previous_error": entry.ProcessingError,
	})

	// Send processing event
	s.broadcaster.SendEvent(events.EventEntryProcessing, entryID, map[string]interface{}{
		"stage":   models.StageCreated,
		"message": "Retrying processing",
	})

	// Process asynchronously in background
	go func(entryID string, content string) {
		// Recover from panics in goroutine
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC in retry processing for entry %s: %v", entryID, r)
				s.logger.SetError(entryID, models.StageAnalyzing, fmt.Errorf("panic: %v", r))
				// Send failure event
				s.broadcaster.SendEvent(events.EventEntryFailed, entryID, map[string]interface{}{
					"error": fmt.Sprintf("%v", r),
					"stage": models.StageFailed,
				})
			}
		}()

		log.Printf("Starting retry processing for entry %s", entryID)

		// Use the same processing logic as CreateEntry
		// Transition to analyzing stage
		s.logger.UpdateStage(entryID, models.StageAnalyzing)
		s.broadcaster.SendEvent(events.EventEntryProcessing, entryID, map[string]interface{}{
			"stage":   models.StageAnalyzing,
			"message": "Analyzing content with AI",
		})

		// Process content with Qwen
		s.logger.LogInfo(entryID, models.StageAnalyzing, "Starting AI analysis (retry)", nil)
		processedData, err := s.processor.ProcessJournalEntry(content)
		if err != nil {
			log.Printf("Failed to process entry %s on retry: %v", entryID, err)
			s.logger.SetError(entryID, models.StageAnalyzing, err)
			// Send failure event
			s.broadcaster.SendEvent(events.EventEntryFailed, entryID, map[string]interface{}{
				"error": err.Error(),
				"stage": models.StageAnalyzing,
			})
			return
		}

		// Continue with the rest of the processing pipeline...
		// (URL fetching, embedding generation, etc.)

		// Create temporary entry for embedding generation
		tempEntry := models.JournalEntry{
			ID:            entryID,
			Content:       content,
			ProcessedData: *processedData,
		}

		// Transition to fetching URLs stage
		s.logger.UpdateStage(entryID, models.StageFetchingURLs)
		s.broadcaster.SendEvent(events.EventEntryProcessing, entryID, map[string]interface{}{
			"stage":   models.StageFetchingURLs,
			"message": "Fetching linked content",
		})

		// Fetch URLs if any
		if len(processedData.ExtractedURLs) > 0 {
			s.logger.LogInfo(entryID, models.StageFetchingURLs, "Fetching URLs", map[string]interface{}{
				"urls_count": len(processedData.ExtractedURLs),
			})

			for i, urlInfo := range processedData.ExtractedURLs {
				s.logger.LogInfo(entryID, models.StageFetchingURLs, fmt.Sprintf("Fetching URL %d/%d", i+1, len(processedData.ExtractedURLs)), map[string]interface{}{
					"url": urlInfo.URL,
				})

				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				fetchedContent, err := s.mcpClient.FetchURL(ctx, urlInfo.URL, urlInfo.Title) // Using Title as reason
				cancel()

				if err != nil {
					s.logger.LogInfo(entryID, models.StageFetchingURLs, fmt.Sprintf("Failed to fetch URL: %v", err), map[string]interface{}{
						"url": urlInfo.URL,
					})
					continue
				}

				// Add fetched content to the URL info
				tempEntry.ProcessedData.ExtractedURLs[i].Title = fetchedContent.Title
				tempEntry.ProcessedData.ExtractedURLs[i].Content = fetchedContent.Content
			}
		}

		// Transition to embedding generation stage
		s.logger.UpdateStage(entryID, models.StageGeneratingEmbeddings)
		s.broadcaster.SendEvent(events.EventEntryProcessing, entryID, map[string]interface{}{
			"stage":   models.StageGeneratingEmbeddings,
			"message": "Creating semantic embeddings",
		})

		// Generate embeddings
		s.logger.LogInfo(entryID, models.StageGeneratingEmbeddings, "Generating embeddings", nil)
		embedding, err := s.processor.CreateEmbedding(tempEntry)
		if err != nil {
			log.Printf("Failed to create embeddings for entry %s: %v", entryID, err)
			s.logger.SetError(entryID, models.StageGeneratingEmbeddings, err)
			// Send failure event
			s.broadcaster.SendEvent(events.EventEntryFailed, entryID, map[string]interface{}{
				"error": err.Error(),
				"stage": models.StageGeneratingEmbeddings,
			})
			return
		}

		// Update entry with processed data
		processedJSON, err := json.Marshal(tempEntry.ProcessedData)
		if err != nil {
			s.logger.SetError(entryID, models.StageGeneratingEmbeddings, fmt.Errorf("failed to marshal processed data: %w", err))
			return
		}

		completedAt := time.Now()
		_, err = s.db.Exec(`
			UPDATE journal_entries 
			SET processed_data = $1, 
			    embedding = $2, 
			    updated_at = $3,
			    processing_stage = $4,
			    processing_completed_at = $5,
			    processing_error = NULL
			WHERE id = $6`,
			processedJSON,
			pgvector.NewVector(embedding),
			completedAt,
			models.StageCompleted,
			completedAt,
			entryID,
		)

		if err != nil {
			log.Printf("Failed to update entry %s with processed data: %v", entryID, err)
			s.logger.SetError(entryID, models.StageGeneratingEmbeddings, err)
			return
		}

		// Mark as completed
		s.logger.UpdateStage(entryID, models.StageCompleted)
		s.logger.LogInfo(entryID, models.StageCompleted, "Processing completed successfully", map[string]interface{}{
			"processing_time": completedAt.Sub(*entry.ProcessingStartedAt).Seconds(),
		})

		// Get updated entry to send in event
		updatedEntry, err := s.GetEntry(entryID)
		if err != nil {
			log.Printf("Failed to get updated entry %s: %v", entryID, err)
			// Create a complete entry structure even if fetch fails
			tempEntry.UpdatedAt = completedAt
			tempEntry.ProcessingStage = models.StageCompleted
			tempEntry.ProcessingCompletedAt = &completedAt

			// Send event with reconstructed entry data
			s.broadcaster.SendEvent(events.EventEntryProcessed, entryID, map[string]interface{}{
				"entry": &tempEntry,
				"stage": models.StageCompleted,
			})
		} else {
			// Send processed event with full entry
			s.broadcaster.SendEvent(events.EventEntryProcessed, entryID, map[string]interface{}{
				"entry": updatedEntry,
				"stage": models.StageCompleted,
			})
		}

		log.Printf("Successfully completed retry processing for entry %s", entryID)
	}(entryID, entry.Content)

	return nil
}

// GetSearchSuggestions returns popular topics and entities for search suggestions
func (s *JournalService) GetSearchSuggestions() (map[string]interface{}, error) {
	// Get top topics
	topicsQuery := `
		SELECT topic, COUNT(*) as count
		FROM journal_entries,
		LATERAL jsonb_array_elements_text(processed_data->'topics') as topic
		WHERE processing_stage = 'completed'
		GROUP BY topic
		ORDER BY count DESC
		LIMIT 10`

	topicsRows, err := s.db.Query(topicsQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get top topics: %w", err)
	}
	defer topicsRows.Close()

	topics := []map[string]interface{}{}
	for topicsRows.Next() {
		var topic string
		var count int
		if err := topicsRows.Scan(&topic, &count); err != nil {
			continue
		}
		topics = append(topics, map[string]interface{}{
			"text":  topic,
			"count": count,
		})
	}

	// Get top entities
	entitiesQuery := `
		SELECT entity, COUNT(*) as count
		FROM journal_entries,
		LATERAL jsonb_array_elements_text(processed_data->'entities') as entity
		WHERE processing_stage = 'completed'
		GROUP BY entity
		ORDER BY count DESC
		LIMIT 10`

	entitiesRows, err := s.db.Query(entitiesQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get top entities: %w", err)
	}
	defer entitiesRows.Close()

	entities := []map[string]interface{}{}
	for entitiesRows.Next() {
		var entity string
		var count int
		if err := entitiesRows.Scan(&entity, &count); err != nil {
			continue
		}
		entities = append(entities, map[string]interface{}{
			"text":  entity,
			"count": count,
		})
	}

	// Get recent searches from metadata
	recentQuery := `
		SELECT DISTINCT content
		FROM journal_entries
		WHERE processing_stage = 'completed'
		ORDER BY created_at DESC
		LIMIT 5`

	recentRows, err := s.db.Query(recentQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent entries: %w", err)
	}
	defer recentRows.Close()

	recentPhrases := []string{}
	for recentRows.Next() {
		var content string
		if err := recentRows.Scan(&content); err != nil {
			continue
		}
		// Extract first meaningful phrase (up to 50 chars)
		if len(content) > 50 {
			content = content[:50] + "..."
		}
		recentPhrases = append(recentPhrases, content)
	}

	return map[string]interface{}{
		"topics":   topics,
		"entities": entities,
		"recent":   recentPhrases,
	}, nil
}

// ExportEntries exports journal entries in various formats
func (s *JournalService) ExportEntries(params SearchParams, format string) ([]byte, string, error) {
	// Get entries using existing search
	entries, err := s.ClassicSearch(params)
	if err != nil {
		return nil, "", fmt.Errorf("failed to search entries: %w", err)
	}

	switch format {
	case "json":
		data, err := json.MarshalIndent(entries, "", "  ")
		if err != nil {
			return nil, "", fmt.Errorf("failed to marshal JSON: %w", err)
		}
		return data, "application/json", nil

	case "markdown":
		var md strings.Builder
		md.WriteString("# Journal Export\n\n")
		md.WriteString(fmt.Sprintf("*Exported on %s*\n\n", time.Now().Format("January 2, 2006")))

		for _, entry := range entries {
			md.WriteString(fmt.Sprintf("## %s\n\n", entry.CreatedAt.Format("January 2, 2006 - 3:04 PM")))

			if entry.ProcessedData.Summary != "" {
				md.WriteString(fmt.Sprintf("**Summary:** %s\n\n", entry.ProcessedData.Summary))
			}

			md.WriteString(entry.Content + "\n\n")

			if len(entry.ProcessedData.Topics) > 0 {
				md.WriteString("**Topics:** " + strings.Join(entry.ProcessedData.Topics, ", ") + "\n\n")
			}

			if len(entry.ProcessedData.Entities) > 0 {
				md.WriteString("**Entities:** " + strings.Join(entry.ProcessedData.Entities, ", ") + "\n\n")
			}

			if entry.ProcessedData.Sentiment != "" {
				md.WriteString(fmt.Sprintf("**Sentiment:** %s\n\n", entry.ProcessedData.Sentiment))
			}

			md.WriteString("---\n\n")
		}

		return []byte(md.String()), "text/markdown", nil

	case "csv":
		var csv strings.Builder
		csv.WriteString("Date,Time,Summary,Content,Topics,Entities,Sentiment,Is Favorite\n")

		for _, entry := range entries {
			csv.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s,%v\n",
				entry.CreatedAt.Format("2006-01-02"),
				entry.CreatedAt.Format("15:04:05"),
				escapeCSV(entry.ProcessedData.Summary),
				escapeCSV(entry.Content),
				escapeCSV(strings.Join(entry.ProcessedData.Topics, "; ")),
				escapeCSV(strings.Join(entry.ProcessedData.Entities, "; ")),
				entry.ProcessedData.Sentiment,
				entry.IsFavorite,
			))
		}

		return []byte(csv.String()), "text/csv", nil

	default:
		return nil, "", fmt.Errorf("unsupported export format: %s", format)
	}
}

// escapeCSV escapes special characters in CSV fields
func escapeCSV(s string) string {
	if strings.ContainsAny(s, ",\"\n\r") {
		s = strings.ReplaceAll(s, `"`, `""`)
		return `"` + s + `"`
	}
	return s
}
