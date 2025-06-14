package handlers

import (
	"encoding/json"
	"fmt"
	
	"github.com/journal/internal/service"
)

type JournalHandlers struct {
	service *service.JournalService
}

func NewJournalHandlers(service *service.JournalService) *JournalHandlers {
	return &JournalHandlers{service: service}
}

// CreateEntryParams for creating journal entries
type CreateEntryParams struct {
	Content string `json:"content"`
}

func (h *JournalHandlers) CreateEntry(params json.RawMessage) (interface{}, error) {
	var p CreateEntryParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	if p.Content == "" {
		return nil, fmt.Errorf("content cannot be empty")
	}
	
	return h.service.CreateEntry(p.Content)
}

// UpdateEntryParams for updating journal entries
type UpdateEntryParams struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}

func (h *JournalHandlers) UpdateEntry(params json.RawMessage) (interface{}, error) {
	var p UpdateEntryParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	if p.ID == "" || p.Content == "" {
		return nil, fmt.Errorf("id and content are required")
	}
	
	return h.service.UpdateEntry(p.ID, p.Content)
}

// GetEntryParams for retrieving a single entry
type GetEntryParams struct {
	ID string `json:"id"`
}

func (h *JournalHandlers) GetEntry(params json.RawMessage) (interface{}, error) {
	var p GetEntryParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	if p.ID == "" {
		return nil, fmt.Errorf("id is required")
	}
	
	return h.service.GetEntry(p.ID)
}

// SearchParams wrapper
type SearchParamsWrapper struct {
	service.SearchParams
	SearchType string `json:"search_type"` // "classic", "vector", "hybrid"
}

func (h *JournalHandlers) Search(params json.RawMessage) (interface{}, error) {
	var p SearchParamsWrapper
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	// Set defaults
	if p.Limit == 0 {
		p.Limit = 50
	}
	if p.SearchType == "" {
		p.SearchType = "classic"
	}
	
	switch p.SearchType {
	case "classic":
		return h.service.ClassicSearch(p.SearchParams)
	case "vector":
		if p.Query == "" {
			// Return empty array for empty query
			return []interface{}{}, nil
		}
		return h.service.VectorSearch(p.Query, p.Limit)
	case "hybrid":
		return h.service.HybridSearch(p.SearchParams)
	default:
		return nil, fmt.Errorf("invalid search_type: %s", p.SearchType)
	}
}

// ToggleFavoriteParams for toggling favorites
type ToggleFavoriteParams struct {
	ID string `json:"id"`
}

func (h *JournalHandlers) ToggleFavorite(params json.RawMessage) (interface{}, error) {
	var p ToggleFavoriteParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	if p.ID == "" {
		return nil, fmt.Errorf("id is required")
	}
	
	if err := h.service.ToggleFavorite(p.ID); err != nil {
		return nil, err
	}
	
	return map[string]string{"status": "success"}, nil
}

// Collection handlers
type CreateCollectionParams struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (h *JournalHandlers) CreateCollection(params json.RawMessage) (interface{}, error) {
	var p CreateCollectionParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	if p.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	
	return h.service.CreateCollection(p.Name, p.Description)
}

func (h *JournalHandlers) GetCollections(params json.RawMessage) (interface{}, error) {
	return h.service.GetCollections()
}

type CollectionOperationParams struct {
	EntryID      string `json:"entry_id"`
	CollectionID string `json:"collection_id"`
}

func (h *JournalHandlers) AddToCollection(params json.RawMessage) (interface{}, error) {
	var p CollectionOperationParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	if p.EntryID == "" || p.CollectionID == "" {
		return nil, fmt.Errorf("entry_id and collection_id are required")
	}
	
	if err := h.service.AddToCollection(p.EntryID, p.CollectionID); err != nil {
		return nil, err
	}
	
	return map[string]string{"status": "success"}, nil
}

func (h *JournalHandlers) RemoveFromCollection(params json.RawMessage) (interface{}, error) {
	var p CollectionOperationParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	if p.EntryID == "" || p.CollectionID == "" {
		return nil, fmt.Errorf("entry_id and collection_id are required")
	}
	
	if err := h.service.RemoveFromCollection(p.EntryID, p.CollectionID); err != nil {
		return nil, err
	}
	
	return map[string]string{"status": "success"}, nil
}

// GetProcessingLogsParams for retrieving processing logs
type GetProcessingLogsParams struct {
	EntryID string `json:"entry_id"`
}

func (h *JournalHandlers) GetProcessingLogs(params json.RawMessage) (interface{}, error) {
	var p GetProcessingLogsParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	if p.EntryID == "" {
		return nil, fmt.Errorf("entry_id is required")
	}
	
	logs, err := h.service.GetProcessingLogs(p.EntryID)
	if err != nil {
		return nil, err
	}
	
	return logs, nil
}

// AnalyzeFailureParams for analyzing processing failures
type AnalyzeFailureParams struct {
	EntryID string `json:"entry_id"`
}

func (h *JournalHandlers) AnalyzeFailure(params json.RawMessage) (interface{}, error) {
	var p AnalyzeFailureParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	if p.EntryID == "" {
		return nil, fmt.Errorf("entry_id is required")
	}
	
	analysis, err := h.service.AnalyzeFailure(p.EntryID)
	if err != nil {
		return nil, err
	}
	
	return analysis, nil
}