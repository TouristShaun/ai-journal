package evaluation

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/journal/internal/db"
	"github.com/journal/internal/models"
	"github.com/journal/internal/service"
)

// Evaluator handles the evaluation of search functionality
type Evaluator struct {
	db             *db.DB
	outputDir      string
	generator      *TestDataGenerator
	journalService *service.JournalService
}

// NewEvaluator creates a new evaluator instance
func NewEvaluator(database *db.DB, outputDir string, journalService *service.JournalService) *Evaluator {
	return &Evaluator{
		db:             database,
		outputDir:      outputDir,
		generator:      NewTestDataGenerator(database),
		journalService: journalService,
	}
}

// SearchMetrics holds evaluation metrics for a search mode
type SearchMetrics struct {
	Mode       string       `json:"mode"`
	Precision  float64      `json:"precision"`
	Recall     float64      `json:"recall"`
	F1Score    float64      `json:"f1_score"`
	NDCG       float64      `json:"ndcg"`
	MRR        float64      `json:"mrr"`
	AvgLatency float64      `json:"avg_latency_ms"`
	TestCases  []TestResult `json:"test_cases"`
	Timestamp  time.Time    `json:"timestamp"`
}

// TestResult represents the result of a single test case
type TestResult struct {
	TestID      string          `json:"test_id"`
	Query       string          `json:"query"`
	ExpectedIDs []string        `json:"expected_ids"`
	ActualIDs   []string        `json:"actual_ids"`
	Precision   float64         `json:"precision"`
	Recall      float64         `json:"recall"`
	Latency     time.Duration   `json:"latency_ms"`
	Filters     json.RawMessage `json:"filters,omitempty"`
}

// GenerateTestData creates synthetic test data
func (e *Evaluator) GenerateTestData(size int) error {
	log.Printf("Generating %d test entries...", size)

	// Ensure output directory exists
	dataDir := filepath.Join(e.outputDir, "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Generate test entries
	entries, err := e.generator.GenerateEntries(size)
	if err != nil {
		return fmt.Errorf("failed to generate entries: %w", err)
	}

	// Generate test cases for each search mode
	testSets := map[string]interface{}{
		"classic_tests": e.generator.GenerateClassicSearchTests(entries),
		"vector_tests":  e.generator.GenerateVectorSearchTests(entries),
		"hybrid_tests":  e.generator.GenerateHybridSearchTests(entries),
		"entries":       entries,
	}

	// Save test data
	for name, data := range testSets {
		filePath := filepath.Join(dataDir, fmt.Sprintf("%s.json", name))
		file, err := os.Create(filePath)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", filePath, err)
		}
		defer file.Close()

		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(data); err != nil {
			return fmt.Errorf("failed to encode %s: %w", name, err)
		}

		log.Printf("Saved %s to %s", name, filePath)
	}

	return nil
}

// RunEvaluation executes evaluation for specified search modes
func (e *Evaluator) RunEvaluation(mode string) (map[string]*SearchMetrics, error) {
	results := make(map[string]*SearchMetrics)

	modes := []string{}
	switch mode {
	case "all":
		modes = []string{"classic", "vector", "hybrid"}
	case "classic", "vector", "hybrid":
		modes = []string{mode}
	default:
		return nil, fmt.Errorf("invalid search mode: %s", mode)
	}

	for _, searchMode := range modes {
		log.Printf("Evaluating %s search...", searchMode)

		metrics, err := e.evaluateSearchMode(searchMode)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate %s search: %w", searchMode, err)
		}

		results[searchMode] = metrics

		// Save intermediate results
		if err := e.saveMetrics(searchMode, metrics); err != nil {
			log.Printf("Warning: failed to save metrics for %s: %v", searchMode, err)
		}
	}

	return results, nil
}

// evaluateSearchMode runs evaluation for a specific search mode
func (e *Evaluator) evaluateSearchMode(mode string) (*SearchMetrics, error) {
	// Load test cases
	testCases, err := e.loadTestCases(mode)
	if err != nil {
		return nil, fmt.Errorf("failed to load test cases: %w", err)
	}

	metrics := &SearchMetrics{
		Mode:      mode,
		Timestamp: time.Now(),
		TestCases: make([]TestResult, 0, len(testCases)),
	}

	var totalPrecision, totalRecall, totalLatency float64
	var validCases int

	// Run each test case
	for _, testCase := range testCases {
		result, err := e.runTestCase(mode, testCase)
		if err != nil {
			log.Printf("Warning: test case %s failed: %v", testCase.ID, err)
			continue
		}

		metrics.TestCases = append(metrics.TestCases, *result)

		if result.Precision > 0 || result.Recall > 0 {
			totalPrecision += result.Precision
			totalRecall += result.Recall
			totalLatency += float64(result.Latency.Milliseconds())
			validCases++
		}
	}

	// Calculate aggregate metrics
	if validCases > 0 {
		metrics.Precision = totalPrecision / float64(validCases)
		metrics.Recall = totalRecall / float64(validCases)
		metrics.F1Score = 2 * (metrics.Precision * metrics.Recall) / (metrics.Precision + metrics.Recall)
		metrics.AvgLatency = totalLatency / float64(validCases)

		// Calculate NDCG and MRR
		metrics.NDCG = e.calculateNDCG(metrics.TestCases)
		metrics.MRR = e.calculateMRR(metrics.TestCases)
	}

	return metrics, nil
}

// loadTestCases loads test cases for a specific search mode
func (e *Evaluator) loadTestCases(mode string) ([]TestCase, error) {
	filePath := filepath.Join(e.outputDir, "data", fmt.Sprintf("%s_tests.json", mode))

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open test file: %w", err)
	}
	defer file.Close()

	var testCases []TestCase
	if err := json.NewDecoder(file).Decode(&testCases); err != nil {
		return nil, fmt.Errorf("failed to decode test cases: %w", err)
	}

	return testCases, nil
}

// runTestCase executes a single test case
func (e *Evaluator) runTestCase(mode string, testCase TestCase) (*TestResult, error) {
	start := time.Now()

	// Prepare search parameters
	searchParams := service.SearchParams{
		Query: testCase.Query,
		Limit: 50, // Default limit for evaluation
	}

	// Add filters if provided
	if len(testCase.Filters) > 0 {
		var filters map[string]interface{}
		if err := json.Unmarshal(testCase.Filters, &filters); err == nil {
			// Parse filters into SearchParams fields
			if favorite, ok := filters["favorites"].(bool); ok {
				searchParams.IsFavorite = &favorite
			}
			if collections, ok := filters["collection_ids"].([]interface{}); ok {
				searchParams.CollectionIDs = make([]string, len(collections))
				for i, c := range collections {
					searchParams.CollectionIDs[i] = c.(string)
				}
			}
			if fromDate, ok := filters["from_date"].(string); ok {
				if t, err := time.Parse("2006-01-02", fromDate); err == nil {
					searchParams.StartDate = &t
				}
			}
			if toDate, ok := filters["to_date"].(string); ok {
				if t, err := time.Parse("2006-01-02", toDate); err == nil {
					searchParams.EndDate = &t
				}
			}
		}
	}

	// Set mode based on search type
	switch mode {
	case "vector":
		searchParams.SemanticMode = testCase.VectorMode
		if searchParams.SemanticMode == "" {
			searchParams.SemanticMode = "similar" // Default
		}
	case "hybrid":
		searchParams.HybridMode = "balanced" // Default hybrid mode
	}

	// Execute search using the journal service
	var entries []models.JournalEntry
	var err error
	
	switch mode {
	case "classic":
		entries, err = e.journalService.ClassicSearch(searchParams)
	case "vector":
		entries, err = e.journalService.VectorSearch(searchParams)
	case "hybrid":
		entries, err = e.journalService.HybridSearch(searchParams)
	default:
		return nil, fmt.Errorf("invalid search mode: %s", mode)
	}
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Extract actual IDs
	actualIDs := make([]string, len(entries))
	for i, entry := range entries {
		actualIDs[i] = entry.ID
	}

	// Calculate metrics
	precision, recall := calculatePrecisionRecall(testCase.ExpectedIDs, actualIDs)

	return &TestResult{
		TestID:      testCase.ID,
		Query:       testCase.Query,
		ExpectedIDs: testCase.ExpectedIDs,
		ActualIDs:   actualIDs,
		Precision:   precision,
		Recall:      recall,
		Latency:     time.Since(start),
		Filters:     testCase.Filters,
	}, nil
}

// calculatePrecisionRecall calculates precision and recall metrics
func calculatePrecisionRecall(expected, actual []string) (precision, recall float64) {
	if len(actual) == 0 {
		return 0, 0
	}

	// Create set of expected IDs for O(1) lookup
	expectedSet := make(map[string]bool)
	for _, id := range expected {
		expectedSet[id] = true
	}

	// Count true positives
	truePositives := 0
	for _, id := range actual {
		if expectedSet[id] {
			truePositives++
		}
	}

	// Calculate metrics
	if len(actual) > 0 {
		precision = float64(truePositives) / float64(len(actual))
	}
	if len(expected) > 0 {
		recall = float64(truePositives) / float64(len(expected))
	}

	return precision, recall
}

// calculateNDCG calculates Normalized Discounted Cumulative Gain
func (e *Evaluator) calculateNDCG(results []TestResult) float64 {
	// Simplified NDCG calculation
	// In practice, this would consider ranking positions
	var totalNDCG float64
	validResults := 0

	for _, result := range results {
		if len(result.ExpectedIDs) > 0 {
			// Calculate DCG for actual results
			dcg := 0.0
			for i, id := range result.ActualIDs {
				relevance := 0.0
				for j, expectedID := range result.ExpectedIDs {
					if id == expectedID {
						// Higher relevance for items appearing earlier in expected list
						relevance = float64(len(result.ExpectedIDs)-j) / float64(len(result.ExpectedIDs))
						break
					}
				}
				if i == 0 {
					dcg += relevance
				} else {
					dcg += relevance / (float64(i) + 1)
				}
			}

			// Calculate ideal DCG
			idealDCG := 0.0
			for i := 0; i < len(result.ExpectedIDs); i++ {
				relevance := float64(len(result.ExpectedIDs)-i) / float64(len(result.ExpectedIDs))
				if i == 0 {
					idealDCG += relevance
				} else {
					idealDCG += relevance / (float64(i) + 1)
				}
			}

			if idealDCG > 0 {
				totalNDCG += dcg / idealDCG
				validResults++
			}
		}
	}

	if validResults > 0 {
		return totalNDCG / float64(validResults)
	}
	return 0.0
}

// calculateMRR calculates Mean Reciprocal Rank
func (e *Evaluator) calculateMRR(results []TestResult) float64 {
	var totalRR float64
	validResults := 0

	for _, result := range results {
		if len(result.ExpectedIDs) > 0 && len(result.ActualIDs) > 0 {
			// Find the rank of the first relevant result
			for i, id := range result.ActualIDs {
				for _, expectedID := range result.ExpectedIDs {
					if id == expectedID {
						totalRR += 1.0 / float64(i+1)
						validResults++
						goto nextResult
					}
				}
			}
		}
	nextResult:
	}

	if validResults > 0 {
		return totalRR / float64(validResults)
	}
	return 0.0
}

// saveMetrics saves evaluation metrics to file
func (e *Evaluator) saveMetrics(mode string, metrics *SearchMetrics) error {
	reportsDir := filepath.Join(e.outputDir, "reports")
	if err := os.MkdirAll(reportsDir, 0755); err != nil {
		return fmt.Errorf("failed to create reports directory: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	filePath := filepath.Join(reportsDir, fmt.Sprintf("%s_metrics_%s.json", mode, timestamp))

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create metrics file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(metrics)
}

// GenerateReport creates a formatted report of evaluation results
func (e *Evaluator) GenerateReport(format string) (string, error) {
	// Load latest metrics for all modes
	metrics := make(map[string]*SearchMetrics)

	for _, mode := range []string{"classic", "vector", "hybrid"} {
		latestMetrics, err := e.loadLatestMetrics(mode)
		if err != nil {
			log.Printf("Warning: no metrics found for %s search", mode)
			continue
		}
		metrics[mode] = latestMetrics
	}

	if len(metrics) == 0 {
		return "", fmt.Errorf("no evaluation results found")
	}

	reporter := NewReporter(e.outputDir)

	switch format {
	case "html":
		return reporter.GenerateHTMLReport(metrics)
	case "json":
		return reporter.GenerateJSONReport(metrics)
	case "csv":
		return reporter.GenerateCSVReport(metrics)
	default:
		return "", fmt.Errorf("unsupported report format: %s", format)
	}
}

// loadLatestMetrics loads the most recent metrics for a search mode
func (e *Evaluator) loadLatestMetrics(mode string) (*SearchMetrics, error) {
	reportsDir := filepath.Join(e.outputDir, "reports")
	pattern := filepath.Join(reportsDir, fmt.Sprintf("%s_metrics_*.json", mode))

	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no metrics files found for %s", mode)
	}

	// Get the latest file (files are named with timestamp)
	latestFile := files[len(files)-1]

	file, err := os.Open(latestFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var metrics SearchMetrics
	if err := json.NewDecoder(file).Decode(&metrics); err != nil {
		return nil, err
	}

	return &metrics, nil
}

// GetLatestMetrics is a public method for getting latest metrics
func (e *Evaluator) GetLatestMetrics(mode string) (*SearchMetrics, error) {
	return e.loadLatestMetrics(mode)
}
