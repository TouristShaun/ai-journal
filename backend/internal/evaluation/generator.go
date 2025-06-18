package evaluation

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/journal/internal/db"
)

// TestDataGenerator creates synthetic test data for evaluation
type TestDataGenerator struct {
	db   *db.DB
	rand *rand.Rand
}

// NewTestDataGenerator creates a new test data generator
func NewTestDataGenerator(database *db.DB) *TestDataGenerator {
	return &TestDataGenerator{
		db:   database,
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// TestEntry represents a test journal entry with known characteristics
type TestEntry struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Topics    []string  `json:"topics"`
	Entities  []string  `json:"entities"`
	Sentiment string    `json:"sentiment"`
	Keywords  []string  `json:"keywords"`
	Category  string    `json:"category"`
	CreatedAt time.Time `json:"created_at"`
	Embedding []float32 `json:"embedding,omitempty"`
}

// TestCase represents a search test case
type TestCase struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Query       string          `json:"query"`
	Filters     json.RawMessage `json:"filters,omitempty"`
	ExpectedIDs []string        `json:"expected_ids"`
	SearchMode  string          `json:"search_mode,omitempty"`
	VectorMode  string          `json:"vector_mode,omitempty"`
}

// Predefined test data patterns
var (
	topics = []string{
		"productivity", "health", "technology", "travel", "relationships",
		"work", "hobbies", "learning", "goals", "reflection",
		"creativity", "mindfulness", "finance", "family", "career",
	}

	entities = []string{
		"John Doe", "Jane Smith", "New York", "San Francisco", "Google",
		"Apple", "Microsoft", "Amazon", "Tesla", "OpenAI",
		"Harvard", "MIT", "Stanford", "Oxford", "Cambridge",
	}

	sentiments = []string{"positive", "negative", "neutral", "mixed"}

	contentTemplates = []string{
		"Today I learned about %s and it made me think about %s. %s was particularly interesting.",
		"Had a great conversation with %s about %s. We discussed how %s impacts our daily lives.",
		"Reflecting on my experience with %s. It's been a journey with %s, learning about %s.",
		"Working on %s project with %s. The main challenge is %s but I'm making progress.",
		"Visited %s and met %s. The %s experience was really memorable.",
	}

	keywords = map[string][]string{
		"productivity":  {"efficiency", "workflow", "automation", "tasks", "schedule"},
		"health":        {"exercise", "nutrition", "wellness", "fitness", "meditation"},
		"technology":    {"software", "hardware", "innovation", "digital", "AI"},
		"travel":        {"adventure", "culture", "exploration", "destination", "journey"},
		"relationships": {"communication", "trust", "connection", "friendship", "love"},
	}
)

// GenerateEntries creates synthetic journal entries
func (g *TestDataGenerator) GenerateEntries(count int) ([]TestEntry, error) {
	entries := make([]TestEntry, count)

	for i := 0; i < count; i++ {
		entry := g.generateSingleEntry(i)
		entries[i] = entry

		// Also insert into database for realistic testing
		if err := g.insertTestEntry(entry); err != nil {
			return nil, fmt.Errorf("failed to insert test entry: %w", err)
		}
	}

	return entries, nil
}

// generateSingleEntry creates a single test entry
func (g *TestDataGenerator) generateSingleEntry(index int) TestEntry {
	// Select random topics (1-3)
	numTopics := g.rand.Intn(3) + 1
	selectedTopics := g.selectRandom(topics, numTopics)

	// Select random entities (1-3)
	numEntities := g.rand.Intn(3) + 1
	selectedEntities := g.selectRandom(entities, numEntities)

	// Select sentiment
	sentiment := sentiments[g.rand.Intn(len(sentiments))]

	// Generate keywords based on topics
	entryKeywords := []string{}
	for _, topic := range selectedTopics {
		if words, ok := keywords[topic]; ok {
			entryKeywords = append(entryKeywords, g.selectRandom(words, 2)...)
		}
	}

	// Generate content
	template := contentTemplates[g.rand.Intn(len(contentTemplates))]
	
	// Ensure we have at least one keyword, use topic as fallback
	var keyword string
	if len(entryKeywords) > 0 {
		keyword = g.selectRandom(entryKeywords, 1)[0]
	} else {
		keyword = selectedTopics[0] // Fallback to topic if no keywords
	}
	
	content := fmt.Sprintf(template,
		selectedTopics[0],
		selectedEntities[0],
		keyword,
	)

	// Generate title and incorporate it into content
	title := fmt.Sprintf("Entry %d: %s and %s", index+1, selectedTopics[0], selectedEntities[0])
	content = title + "\n\n" + content

	// Add keywords to content for classic search testing
	content += "\n\nKeywords: " + strings.Join(entryKeywords, ", ")

	// Create entry
	return TestEntry{
		ID:        uuid.New().String(),
		Title:     title,
		Content:   content,
		Topics:    selectedTopics,
		Entities:  selectedEntities,
		Sentiment: sentiment,
		Keywords:  entryKeywords,
		Category:  selectedTopics[0], // Primary topic as category
		CreatedAt: time.Now().Add(-time.Duration(g.rand.Intn(365)) * 24 * time.Hour),
		// Embedding will be generated when inserted
	}
}

// selectRandom selects n random items from a slice
func (g *TestDataGenerator) selectRandom(items []string, n int) []string {
	// Handle empty input
	if len(items) == 0 {
		return []string{}
	}
	
	// Ensure n is not larger than available items
	if n > len(items) {
		n = len(items)
	}
	
	// Handle zero or negative n
	if n <= 0 {
		return []string{}
	}

	shuffled := make([]string, len(items))
	copy(shuffled, items)

	g.rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	return shuffled[:n]
}

// insertTestEntry inserts a test entry into the database
func (g *TestDataGenerator) insertTestEntry(entry TestEntry) error {
	// Create processed data matching the expected format
	processedData := map[string]interface{}{
		"summary":   fmt.Sprintf("Test entry about %s", strings.Join(entry.Topics, " and ")),
		"topics":    entry.Topics,
		"entities":  entry.Entities,
		"sentiment": entry.Sentiment,
		"keywords":  entry.Keywords,
	}

	processedJSON, err := json.Marshal(processedData)
	if err != nil {
		return err
	}

	// Insert entry
	query := `
		INSERT INTO journal_entries (
			id, content, processed_data, 
			processing_stage, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err = g.db.Exec(query,
		entry.ID,
		entry.Content,
		processedJSON,
		"completed", // Mark as already processed
		entry.CreatedAt,
		entry.CreatedAt,
	)

	return err
}

// GenerateClassicSearchTests creates test cases for classic search
func (g *TestDataGenerator) GenerateClassicSearchTests(entries []TestEntry) []TestCase {
	testCases := []TestCase{}

	// Test 1: Single keyword search
	for i := 0; i < 5 && i < len(entries); i++ {
		entry := entries[i]
		if len(entry.Keywords) > 0 {
			testCases = append(testCases, TestCase{
				ID:          fmt.Sprintf("classic_keyword_%d", i),
				Name:        fmt.Sprintf("Single keyword: %s", entry.Keywords[0]),
				Description: "Test single keyword matching",
				Query:       entry.Keywords[0],
				ExpectedIDs: []string{entry.ID},
				SearchMode:  "classic",
			})
		}
	}

	// Test 2: Topic search
	topicGroups := g.groupByTopic(entries)
	for topic, topicEntries := range topicGroups {
		if len(topicEntries) >= 2 {
			expectedIDs := []string{}
			for _, e := range topicEntries[:min(5, len(topicEntries))] {
				expectedIDs = append(expectedIDs, e.ID)
			}

			testCases = append(testCases, TestCase{
				ID:          fmt.Sprintf("classic_topic_%s", topic),
				Name:        fmt.Sprintf("Topic search: %s", topic),
				Description: "Test topic-based search",
				Query:       topic,
				ExpectedIDs: expectedIDs,
				SearchMode:  "classic",
			})
		}
	}

	// Test 3: Entity search
	entityGroups := g.groupByEntity(entries)
	for entity, entityEntries := range entityGroups {
		if len(entityEntries) >= 2 {
			expectedIDs := []string{}
			for _, e := range entityEntries[:min(3, len(entityEntries))] {
				expectedIDs = append(expectedIDs, e.ID)
			}

			testCases = append(testCases, TestCase{
				ID:          fmt.Sprintf("classic_entity_%s", strings.ReplaceAll(entity, " ", "_")),
				Name:        fmt.Sprintf("Entity search: %s", entity),
				Description: "Test entity-based search",
				Query:       fmt.Sprintf("\"%s\"", entity), // Exact phrase
				ExpectedIDs: expectedIDs,
				SearchMode:  "classic",
			})
		}
	}

	// Test 4: Search with filters
	filters := map[string]interface{}{
		"favorites": false,
		"from_date": time.Now().Add(-30 * 24 * time.Hour).Format("2006-01-02"),
	}
	filtersJSON, _ := json.Marshal(filters)

	recentEntries := g.filterRecent(entries, 30)
	if len(recentEntries) > 0 {
		expectedIDs := []string{}
		for _, e := range recentEntries[:min(5, len(recentEntries))] {
			expectedIDs = append(expectedIDs, e.ID)
		}

		testCases = append(testCases, TestCase{
			ID:          "classic_filtered_recent",
			Name:        "Recent entries with filters",
			Description: "Test search with date filters",
			Query:       "", // Empty query to test filter-only search
			Filters:     filtersJSON,
			ExpectedIDs: expectedIDs,
			SearchMode:  "classic",
		})
	}

	return testCases
}

// GenerateVectorSearchTests creates test cases for vector search
func (g *TestDataGenerator) GenerateVectorSearchTests(entries []TestEntry) []TestCase {
	testCases := []TestCase{}

	// Test different vector search modes
	modes := []string{"similar", "explore", "contrast"}

	for _, mode := range modes {
		// Select diverse entries as query sources
		queryEntries := g.selectDiverseEntries(entries, 3)

		for i, queryEntry := range queryEntries {
			// Find related entries based on mode
			var expectedIDs []string

			switch mode {
			case "similar":
				// Find entries with similar topics/entities
				expectedIDs = g.findSimilarEntries(queryEntry, entries, 5)
			case "explore":
				// Find entries with some overlap but different focus
				expectedIDs = g.findExploratoryEntries(queryEntry, entries, 5)
			case "contrast":
				// Find entries with different sentiment or opposing topics
				expectedIDs = g.findContrastingEntries(queryEntry, entries, 5)
			}

			testCases = append(testCases, TestCase{
				ID:          fmt.Sprintf("vector_%s_%d", mode, i),
				Name:        fmt.Sprintf("Vector %s: %s", mode, queryEntry.Title),
				Description: fmt.Sprintf("Test %s mode vector search", mode),
				Query:       queryEntry.Content[:min(100, len(queryEntry.Content))],
				ExpectedIDs: expectedIDs,
				SearchMode:  "vector",
				VectorMode:  mode,
			})
		}
	}

	return testCases
}

// GenerateHybridSearchTests creates test cases for hybrid search
func (g *TestDataGenerator) GenerateHybridSearchTests(entries []TestEntry) []TestCase {
	testCases := []TestCase{}

	// Test 1: Keyword + Semantic
	for i := 0; i < 3 && i < len(entries); i++ {
		entry := entries[i]
		if len(entry.Keywords) > 0 && len(entry.Topics) > 0 {
			// Find entries that match keyword OR have similar topics
			expectedIDs := g.findHybridMatches(entry.Keywords[0], entry.Topics, entries, 5)

			testCases = append(testCases, TestCase{
				ID:          fmt.Sprintf("hybrid_keyword_semantic_%d", i),
				Name:        fmt.Sprintf("Hybrid: %s + semantics", entry.Keywords[0]),
				Description: "Test hybrid search combining keyword and semantic",
				Query:       entry.Keywords[0],
				ExpectedIDs: expectedIDs,
				SearchMode:  "hybrid",
			})
		}
	}

	// Test 2: Natural language queries
	naturalQueries := []struct {
		query    string
		topics   []string
		keywords []string
	}{
		{
			query:    "How can I improve my productivity while working from home?",
			topics:   []string{"productivity", "work"},
			keywords: []string{"improve", "productivity", "working", "home"},
		},
		{
			query:    "Tell me about recent technology innovations",
			topics:   []string{"technology"},
			keywords: []string{"recent", "technology", "innovations"},
		},
	}

	for i, nq := range naturalQueries {
		expectedIDs := g.findHybridMatches(nq.query, nq.topics, entries, 5)

		testCases = append(testCases, TestCase{
			ID:          fmt.Sprintf("hybrid_natural_%d", i),
			Name:        fmt.Sprintf("Natural query: %s", truncate(nq.query, 30)),
			Description: "Test hybrid search with natural language",
			Query:       nq.query,
			ExpectedIDs: expectedIDs,
			SearchMode:  "hybrid",
		})
	}

	return testCases
}

// Helper functions

func (g *TestDataGenerator) groupByTopic(entries []TestEntry) map[string][]TestEntry {
	groups := make(map[string][]TestEntry)
	for _, entry := range entries {
		for _, topic := range entry.Topics {
			groups[topic] = append(groups[topic], entry)
		}
	}
	return groups
}

func (g *TestDataGenerator) groupByEntity(entries []TestEntry) map[string][]TestEntry {
	groups := make(map[string][]TestEntry)
	for _, entry := range entries {
		for _, entity := range entry.Entities {
			groups[entity] = append(groups[entity], entry)
		}
	}
	return groups
}

func (g *TestDataGenerator) filterRecent(entries []TestEntry, days int) []TestEntry {
	cutoff := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	recent := []TestEntry{}
	for _, entry := range entries {
		if entry.CreatedAt.After(cutoff) {
			recent = append(recent, entry)
		}
	}
	return recent
}

func (g *TestDataGenerator) selectDiverseEntries(entries []TestEntry, count int) []TestEntry {
	// Select entries with different primary topics
	topicMap := make(map[string]TestEntry)
	for _, entry := range entries {
		if len(entry.Topics) > 0 {
			if _, exists := topicMap[entry.Topics[0]]; !exists {
				topicMap[entry.Topics[0]] = entry
			}
		}
		if len(topicMap) >= count {
			break
		}
	}

	diverse := []TestEntry{}
	for _, entry := range topicMap {
		diverse = append(diverse, entry)
	}
	return diverse
}

func (g *TestDataGenerator) findSimilarEntries(query TestEntry, entries []TestEntry, count int) []string {
	similar := []string{}

	for _, entry := range entries {
		if entry.ID == query.ID {
			continue
		}

		// Check topic overlap
		overlap := 0
		for _, qt := range query.Topics {
			for _, et := range entry.Topics {
				if qt == et {
					overlap++
				}
			}
		}

		if overlap > 0 {
			similar = append(similar, entry.ID)
		}

		if len(similar) >= count {
			break
		}
	}

	return similar
}

func (g *TestDataGenerator) findExploratoryEntries(query TestEntry, entries []TestEntry, count int) []string {
	exploratory := []string{}

	for _, entry := range entries {
		if entry.ID == query.ID {
			continue
		}

		// Look for entries with partial overlap
		topicOverlap := 0
		entityOverlap := 0

		for _, qt := range query.Topics {
			for _, et := range entry.Topics {
				if qt == et {
					topicOverlap++
				}
			}
		}

		for _, qe := range query.Entities {
			for _, ee := range entry.Entities {
				if qe == ee {
					entityOverlap++
				}
			}
		}

		// Exploratory: some overlap but not too similar
		if (topicOverlap == 1 && entityOverlap == 0) || (topicOverlap == 0 && entityOverlap == 1) {
			exploratory = append(exploratory, entry.ID)
		}

		if len(exploratory) >= count {
			break
		}
	}

	return exploratory
}

func (g *TestDataGenerator) findContrastingEntries(query TestEntry, entries []TestEntry, count int) []string {
	contrasting := []string{}

	for _, entry := range entries {
		if entry.ID == query.ID {
			continue
		}

		// Different sentiment or no topic overlap
		differentSentiment := entry.Sentiment != query.Sentiment
		noTopicOverlap := true

		for _, qt := range query.Topics {
			for _, et := range entry.Topics {
				if qt == et {
					noTopicOverlap = false
					break
				}
			}
		}

		if differentSentiment || noTopicOverlap {
			contrasting = append(contrasting, entry.ID)
		}

		if len(contrasting) >= count {
			break
		}
	}

	return contrasting
}

func (g *TestDataGenerator) findHybridMatches(query string, topics []string, entries []TestEntry, count int) []string {
	matches := []string{}
	queryLower := strings.ToLower(query)

	for _, entry := range entries {
		// Check keyword match
		contentLower := strings.ToLower(entry.Content)
		keywordMatch := strings.Contains(contentLower, queryLower)

		// Check topic match
		topicMatch := false
		for _, qt := range topics {
			for _, et := range entry.Topics {
				if qt == et {
					topicMatch = true
					break
				}
			}
		}

		if keywordMatch || topicMatch {
			matches = append(matches, entry.ID)
		}

		if len(matches) >= count {
			break
		}
	}

	return matches
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length] + "..."
}
