package evaluation

import (
	"fmt"

	"github.com/journal/internal/db"
	"github.com/journal/internal/models"
	"github.com/journal/internal/service"
)

// SearchService wraps the journal service for evaluation
type SearchService struct {
	journalService *service.JournalService
}

// NewSearchService creates a new search service for evaluation
func NewSearchService(database *db.DB) *SearchService {
	return &SearchService{
		journalService: service.NewJournalService(database, nil, nil, nil, nil),
	}
}

// ExecuteSearch runs a search based on the parameters
func (s *SearchService) ExecuteSearch(params service.SearchParams) ([]models.JournalEntry, error) {
	// Default search type is classic if not specified
	searchType := "classic"

	// Check if this is a vector search based on semantic mode
	if params.SemanticMode != "" {
		searchType = "vector"
	}

	// Check if this is a hybrid search based on hybrid mode
	if params.HybridMode != "" {
		searchType = "hybrid"
	}

	switch searchType {
	case "classic":
		return s.journalService.ClassicSearch(params)
	case "vector":
		return s.journalService.VectorSearch(params)
	case "hybrid":
		return s.journalService.HybridSearch(params)
	default:
		return nil, fmt.Errorf("invalid search type: %s", searchType)
	}
}
