package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"

	"github.com/journal/internal/db"
	"github.com/journal/internal/evaluation"
	"github.com/journal/internal/events"
	"github.com/journal/internal/jsonrpc"
	"github.com/journal/internal/service"
)

// EvaluationHandler handles evaluation-related RPC methods
type EvaluationHandler struct {
	db             *db.DB
	broadcaster    *events.Broadcaster
	evaluator      *evaluation.Evaluator
	journalService *service.JournalService
}

// NewEvaluationHandler creates a new evaluation handler
func NewEvaluationHandler(database *db.DB, broadcaster *events.Broadcaster, journalService *service.JournalService) *EvaluationHandler {
	outputDir := filepath.Join(".", "evaluation_results")
	return &EvaluationHandler{
		db:             database,
		broadcaster:    broadcaster,
		evaluator:      evaluation.NewEvaluator(database, outputDir, journalService),
		journalService: journalService,
	}
}

// GenerateTestDataParams contains parameters for generating test data
type GenerateTestDataParams struct {
	Size int `json:"size"`
}

// GenerateTestDataResult contains the result of test data generation
type GenerateTestDataResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Count   int    `json:"count"`
}

// GenerateTestData generates synthetic test data for evaluation
func (h *EvaluationHandler) GenerateTestData(rawParams json.RawMessage) (interface{}, error) {
	var params GenerateTestDataParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	if params.Size <= 0 {
		params.Size = 100 // Default size
	}

	// Broadcast start event
	h.broadcaster.Broadcast("evaluation.generate.started", map[string]interface{}{
		"size": params.Size,
	})

	// Generate test data
	err := h.evaluator.GenerateTestData(params.Size)
	if err != nil {
		h.broadcaster.Broadcast("evaluation.generate.failed", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("failed to generate test data: %w", err)
	}

	// Broadcast completion
	h.broadcaster.Broadcast("evaluation.generate.completed", map[string]interface{}{
		"size": params.Size,
	})

	return GenerateTestDataResult{
		Success: true,
		Message: fmt.Sprintf("Successfully generated %d test entries", params.Size),
		Count:   params.Size,
	}, nil
}

// RunEvaluationParams contains parameters for running evaluation
type RunEvaluationParams struct {
	Mode string `json:"mode"` // "all", "classic", "vector", or "hybrid"
}

// RunEvaluationResult contains evaluation metrics
type RunEvaluationResult struct {
	Success bool                                  `json:"success"`
	Metrics map[string]*evaluation.SearchMetrics `json:"metrics"`
}

// RunEvaluation runs evaluation for specified search modes
func (h *EvaluationHandler) RunEvaluation(rawParams json.RawMessage) (interface{}, error) {
	var params RunEvaluationParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	if params.Mode == "" {
		params.Mode = "all"
	}

	// Broadcast start event
	h.broadcaster.Broadcast("evaluation.run.started", map[string]interface{}{
		"mode": params.Mode,
	})

	// Track progress
	modes := []string{}
	switch params.Mode {
	case "all":
		modes = []string{"classic", "vector", "hybrid"}
	case "classic", "vector", "hybrid":
		modes = []string{params.Mode}
	default:
		return nil, fmt.Errorf("invalid search mode: %s", params.Mode)
	}

	// Run evaluation with progress updates
	results := make(map[string]*evaluation.SearchMetrics)
	for i, mode := range modes {
		h.broadcaster.Broadcast("evaluation.run.progress", map[string]interface{}{
			"mode":     mode,
			"current":  i + 1,
			"total":    len(modes),
			"progress": float64(i) / float64(len(modes)) * 100,
		})

		log.Printf("Evaluating %s search...", mode)
		metrics, err := h.evaluator.RunEvaluation(mode)
		if err != nil {
			h.broadcaster.Broadcast("evaluation.run.failed", map[string]interface{}{
				"mode":  mode,
				"error": err.Error(),
			})
			return nil, fmt.Errorf("failed to evaluate %s: %w", mode, err)
		}

		// Merge results
		for k, v := range metrics {
			results[k] = v
		}
	}

	// Broadcast completion
	h.broadcaster.Broadcast("evaluation.run.completed", map[string]interface{}{
		"modes": modes,
	})

	return RunEvaluationResult{
		Success: true,
		Metrics: results,
	}, nil
}

// GenerateReportParams contains parameters for report generation
type GenerateReportParams struct {
	Format string `json:"format"` // "html", "json", or "csv"
}

// GenerateReportResult contains the generated report
type GenerateReportResult struct {
	Success  bool   `json:"success"`
	Format   string `json:"format"`
	FilePath string `json:"file_path"`
	Content  string `json:"content,omitempty"`
}

// GenerateReport generates evaluation report in specified format
func (h *EvaluationHandler) GenerateReport(rawParams json.RawMessage) (interface{}, error) {
	var params GenerateReportParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	if params.Format == "" {
		params.Format = "html"
	}

	// Generate report
	reportPath, err := h.evaluator.GenerateReport(params.Format)
	if err != nil {
		return nil, fmt.Errorf("failed to generate report: %w", err)
	}

	return GenerateReportResult{
		Success:  true,
		Format:   params.Format,
		FilePath: reportPath,
	}, nil
}

// GetLatestResultsParams contains parameters for getting latest results
type GetLatestResultsParams struct{}

// GetLatestResultsResult contains the latest evaluation results
type GetLatestResultsResult struct {
	Success bool                                  `json:"success"`
	Found   bool                                  `json:"found"`
	Metrics map[string]*evaluation.SearchMetrics `json:"metrics,omitempty"`
}

// GetLatestResults retrieves the most recent evaluation results
func (h *EvaluationHandler) GetLatestResults(rawParams json.RawMessage) (interface{}, error) {
	results := make(map[string]*evaluation.SearchMetrics)

	// Try to load latest metrics for each mode
	for _, mode := range []string{"classic", "vector", "hybrid"} {
		metrics, err := h.evaluator.GetLatestMetrics(mode)
		if err == nil {
			results[mode] = metrics
		}
	}

	return GetLatestResultsResult{
		Success: true,
		Found:   len(results) > 0,
		Metrics: results,
	}, nil
}

// RunFullEvaluationParams contains parameters for full evaluation
type RunFullEvaluationParams struct {
	Size int `json:"size"`
}

// RunFullEvaluationResult contains full evaluation results
type RunFullEvaluationResult struct {
	Success     bool                                  `json:"success"`
	TestCount   int                                   `json:"test_count"`
	Metrics     map[string]*evaluation.SearchMetrics `json:"metrics"`
	ReportPaths map[string]string                     `json:"report_paths"`
}

// RunFullEvaluation runs complete evaluation pipeline
func (h *EvaluationHandler) RunFullEvaluation(rawParams json.RawMessage) (interface{}, error) {
	var params RunFullEvaluationParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	if params.Size <= 0 {
		params.Size = 100
	}

	// Step 1: Generate test data
	h.broadcaster.Broadcast("evaluation.full.progress", map[string]interface{}{
		"stage":   "generating_data",
		"message": fmt.Sprintf("Generating %d test entries...", params.Size),
	})

	err := h.evaluator.GenerateTestData(params.Size)
	if err != nil {
		return nil, fmt.Errorf("failed to generate test data: %w", err)
	}

	// Step 2: Run evaluations
	h.broadcaster.Broadcast("evaluation.full.progress", map[string]interface{}{
		"stage":   "running_tests",
		"message": "Running evaluation for all search modes...",
	})

	metrics, err := h.evaluator.RunEvaluation("all")
	if err != nil {
		return nil, fmt.Errorf("failed to run evaluation: %w", err)
	}

	// Step 3: Generate reports
	h.broadcaster.Broadcast("evaluation.full.progress", map[string]interface{}{
		"stage":   "generating_reports",
		"message": "Generating evaluation reports...",
	})

	reportPaths := make(map[string]string)
	for _, format := range []string{"html", "json", "csv"} {
		path, err := h.evaluator.GenerateReport(format)
		if err != nil {
			log.Printf("Warning: failed to generate %s report: %v", format, err)
			continue
		}
		reportPaths[format] = path
	}

	// Broadcast completion
	h.broadcaster.Broadcast("evaluation.full.completed", map[string]interface{}{
		"test_count": params.Size,
		"modes":      []string{"classic", "vector", "hybrid"},
	})

	return RunFullEvaluationResult{
		Success:     true,
		TestCount:   params.Size,
		Metrics:     metrics,
		ReportPaths: reportPaths,
	}, nil
}

// Register registers all evaluation-related RPC methods
func (h *EvaluationHandler) Register(server *jsonrpc.Server) {
	server.RegisterMethod("evaluation.generateTestData", h.GenerateTestData)
	server.RegisterMethod("evaluation.run", h.RunEvaluation)
	server.RegisterMethod("evaluation.generateReport", h.GenerateReport)
	server.RegisterMethod("evaluation.getLatestResults", h.GetLatestResults)
	server.RegisterMethod("evaluation.runFull", h.RunFullEvaluation)
}