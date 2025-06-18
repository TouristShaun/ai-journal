package service

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/journal/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMockDB(t *testing.T) (*db.DB, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	return &db.DB{DB: mockDB}, mock
}

func TestGetSearchSuggestions(t *testing.T) {
	database, mock := setupMockDB(t)
	defer database.Close()

	service := &JournalService{
		db: database,
	}

	// Mock the topics query
	topicsRows := sqlmock.NewRows([]string{"topic", "count"}).
		AddRow("golang", 5).
		AddRow("testing", 3).
		AddRow("database", 2)

	mock.ExpectQuery(`SELECT topic, COUNT\(\*\) as count
		FROM journal_entries,
		LATERAL jsonb_array_elements_text\(processed_data->'topics'\) as topic
		WHERE processing_stage = 'completed'
		GROUP BY topic
		ORDER BY count DESC
		LIMIT 10`).
		WillReturnRows(topicsRows)

	// Mock the entities query
	entitiesRows := sqlmock.NewRows([]string{"entity", "count"}).
		AddRow("John Doe", 4).
		AddRow("New York", 2)

	mock.ExpectQuery(`SELECT entity, COUNT\(\*\) as count
		FROM journal_entries,
		LATERAL jsonb_array_elements_text\(processed_data->'entities'\) as entity
		WHERE processing_stage = 'completed'
		GROUP BY entity
		ORDER BY count DESC
		LIMIT 10`).
		WillReturnRows(entitiesRows)

	// Mock the recent entries query
	recentRows := sqlmock.NewRows([]string{"content"}).
		AddRow("Today was a productive day working on the new feature...").
		AddRow("Had a great meeting with the team about...")

	mock.ExpectQuery(`SELECT DISTINCT content
		FROM journal_entries
		WHERE processing_stage = 'completed'
		ORDER BY created_at DESC
		LIMIT 5`).
		WillReturnRows(recentRows)

	// Call the method
	suggestions, err := service.GetSearchSuggestions()
	require.NoError(t, err)

	// Verify the results
	assert.NotNil(t, suggestions)

	topics, ok := suggestions["topics"].([]map[string]interface{})
	assert.True(t, ok)
	assert.Len(t, topics, 3)
	assert.Equal(t, "golang", topics[0]["text"])
	assert.Equal(t, 5, topics[0]["count"])

	entities, ok := suggestions["entities"].([]map[string]interface{})
	assert.True(t, ok)
	assert.Len(t, entities, 2)
	assert.Equal(t, "John Doe", entities[0]["text"])
	assert.Equal(t, 4, entities[0]["count"])

	recent, ok := suggestions["recent"].([]string)
	assert.True(t, ok)
	assert.Len(t, recent, 2)
	assert.Equal(t, "Today was a productive day working on the new feat...", recent[0])

	// Ensure all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestClassicSearch(t *testing.T) {
	database, mock := setupMockDB(t)
	defer database.Close()

	service := &JournalService{
		db: database,
	}

	// Test search with query
	params := SearchParams{
		Query: "golang",
		Limit: 10,
	}

	rows := sqlmock.NewRows([]string{
		"id", "content", "processed_data", "created_at", "updated_at",
		"is_favorite", "original_entry_id", "processing_stage",
		"processing_started_at", "processing_completed_at", "processing_error",
		"collection_ids",
	}).AddRow(
		"123", "Learning golang today", `{"summary": "test", "topics": [], "entities": [], "sentiment": "positive"}`,
		time.Now(), time.Now(), false, nil, "completed",
		time.Now(), time.Now(), nil, "{}")

	mock.ExpectQuery(`SELECT DISTINCT(.*)FROM journal_entries(.*)WHERE(.*)plainto_tsquery`).
		WithArgs("golang", 10).
		WillReturnRows(rows)

	entries, err := service.ClassicSearch(params)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "123", entries[0].ID)
	assert.Equal(t, "Learning golang today", entries[0].Content)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestVectorSearch(t *testing.T) {
	database, mock := setupMockDB(t)
	defer database.Close()

	// Mock processor to avoid actual embedding generation
	service := &JournalService{
		db:        database,
		processor: nil, // Will need to mock this properly in real tests
	}

	params := SearchParams{
		Query: "",
		Limit: 10,
	}

	// Test empty query returns empty result
	entries, err := service.VectorSearch(params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "query cannot be empty")
	assert.Nil(t, entries)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHybridSearch(t *testing.T) {
	database, mock := setupMockDB(t)
	defer database.Close()

	service := &JournalService{
		db: database,
	}

	params := SearchParams{
		Query:      "test",
		Limit:      5,
		HybridMode: "balanced",
	}

	// Mock classic search results
	classicRows := sqlmock.NewRows([]string{
		"id", "content", "processed_data", "created_at", "updated_at",
		"is_favorite", "original_entry_id", "processing_stage",
		"processing_started_at", "processing_completed_at", "processing_error",
		"collection_ids",
	}).AddRow(
		"123", "This is a test entry", `{"summary": "test", "topics": [], "entities": [], "sentiment": "neutral"}`,
		time.Now(), time.Now(), false, nil, "completed",
		time.Now(), time.Now(), nil, "{}")

	mock.ExpectQuery(`SELECT DISTINCT(.*)FROM journal_entries(.*)WHERE(.*)plainto_tsquery`).
		WithArgs("test", 5).
		WillReturnRows(classicRows)

	// Note: Vector search would fail due to nil processor, but hybrid search handles this gracefully
	entries, err := service.HybridSearch(params)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "123", entries[0].ID)

	assert.NoError(t, mock.ExpectationsWereMet())
}
