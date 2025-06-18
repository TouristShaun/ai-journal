package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/journal/internal/logger"
	"github.com/journal/internal/models"
	"github.com/journal/internal/ollama"
)

// FailureAnalyzer provides AI-powered analysis of processing failures
type FailureAnalyzer struct {
	processor *ollama.Processor
	logger    *logger.ProcessingLogger
}

// NewFailureAnalyzer creates a new failure analyzer
func NewFailureAnalyzer(processor *ollama.Processor, logger *logger.ProcessingLogger) *FailureAnalyzer {
	return &FailureAnalyzer{
		processor: processor,
		logger:    logger,
	}
}

// FailureCause represents a potential cause of failure with probability
type FailureCause struct {
	Cause       string  `json:"cause"`
	Description string  `json:"description"`
	Probability float64 `json:"probability"`
	Solution    string  `json:"solution"`
}

// FailureAnalysis contains the analysis results
type FailureAnalysis struct {
	EntryID        string         `json:"entry_id"`
	FailedStage    string         `json:"failed_stage"`
	Error          string         `json:"error"`
	LikelyCauses   []FailureCause `json:"likely_causes"`
	Recommendation string         `json:"recommendation"`
}

// AnalyzeFailure analyzes why a journal entry processing failed
func (fa *FailureAnalyzer) AnalyzeFailure(ctx context.Context, entryID string, entry *models.JournalEntry) (*FailureAnalysis, error) {
	// Get all logs for the entry
	logs, err := fa.logger.GetLogs(entryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get processing logs: %w", err)
	}

	// Filter error logs
	var errorLogs []models.ProcessingLog
	var lastStage models.ProcessingStage
	for _, log := range logs {
		if log.Level == "error" {
			errorLogs = append(errorLogs, log)
		}
		lastStage = log.Stage
	}

	// Common failure patterns
	commonFailures := map[string][]FailureCause{
		"analyzing": {
			{
				Cause:       "Ollama service unavailable",
				Description: "The Ollama AI service is not running or not accessible",
				Probability: 0.7,
				Solution:    "Ensure Ollama is running with 'ollama serve' and the qwen3:8b model is installed",
			},
			{
				Cause:       "Content too large",
				Description: "The journal entry content exceeds the model's context window",
				Probability: 0.2,
				Solution:    "Consider splitting very long entries or increasing model context size",
			},
			{
				Cause:       "Invalid JSON response",
				Description: "The AI model returned malformed JSON",
				Probability: 0.1,
				Solution:    "This is usually temporary - retry the operation",
			},
		},
		"fetching_urls": {
			{
				Cause:       "MCP agent unavailable",
				Description: "The MCP agent service is not running or not accessible",
				Probability: 0.6,
				Solution:    "Start the MCP agent with 'make run-mcp-agent'",
			},
			{
				Cause:       "Network timeout",
				Description: "URL fetching timed out due to slow network or unresponsive sites",
				Probability: 0.3,
				Solution:    "Check network connectivity and ensure target URLs are accessible",
			},
			{
				Cause:       "Invalid URL format",
				Description: "One or more URLs in the entry are malformed",
				Probability: 0.1,
				Solution:    "Verify URL formats in the journal entry",
			},
		},
		"generating_embeddings": {
			{
				Cause:       "Embedding model not available",
				Description: "The nomic-embed-text model is not installed in Ollama",
				Probability: 0.8,
				Solution:    "Install the model with 'ollama pull nomic-embed-text'",
			},
			{
				Cause:       "Content preprocessing failed",
				Description: "Failed to prepare content for embedding generation",
				Probability: 0.2,
				Solution:    "Check for special characters or encoding issues in the content",
			},
		},
	}

	// Build context for AI analysis
	logsContext := fa.buildLogsContext(logs, errorLogs)

	// Get likely causes based on the failed stage
	var likelyCauses []FailureCause
	if causes, exists := commonFailures[string(lastStage)]; exists {
		likelyCauses = causes
	}

	// If we have error logs, use AI to refine the analysis
	if len(errorLogs) > 0 && entry != nil {
		refinedAnalysis, err := fa.aiAnalyzeError(ctx, entry.Content, logsContext, errorLogs)
		if err == nil && refinedAnalysis != nil {
			// Merge AI insights with common patterns
			likelyCauses = fa.mergeAnalyses(likelyCauses, refinedAnalysis)
		}
	}

	// Sort by probability
	sort.Slice(likelyCauses, func(i, j int) bool {
		return likelyCauses[i].Probability > likelyCauses[j].Probability
	})

	// Take top 80% probable causes
	topCauses := fa.getTop80PercentCauses(likelyCauses)

	// Generate recommendation
	recommendation := fa.generateRecommendation(topCauses, lastStage)

	// Get the primary error message
	errorMsg := ""
	if entry != nil && entry.ProcessingError != nil {
		errorMsg = *entry.ProcessingError
	} else if len(errorLogs) > 0 {
		errorMsg = errorLogs[0].Message
	}

	return &FailureAnalysis{
		EntryID:        entryID,
		FailedStage:    string(lastStage),
		Error:          errorMsg,
		LikelyCauses:   topCauses,
		Recommendation: recommendation,
	}, nil
}

// buildLogsContext creates a summary of logs for AI analysis
func (fa *FailureAnalyzer) buildLogsContext(logs []models.ProcessingLog, errorLogs []models.ProcessingLog) string {
	var context strings.Builder

	context.WriteString("Processing Timeline:\n")
	for _, log := range logs {
		context.WriteString(fmt.Sprintf("[%s] %s - %s: %s\n",
			log.CreatedAt.Format("15:04:05"),
			log.Stage,
			log.Level,
			log.Message))
	}

	if len(errorLogs) > 0 {
		context.WriteString("\nError Details:\n")
		for _, errLog := range errorLogs {
			context.WriteString(fmt.Sprintf("Stage: %s\nError: %s\n", errLog.Stage, errLog.Message))
			if len(errLog.Details) > 0 {
				detailsJSON, _ := json.MarshalIndent(errLog.Details, "", "  ")
				context.WriteString(fmt.Sprintf("Details: %s\n", string(detailsJSON)))
			}
		}
	}

	return context.String()
}

// aiAnalyzeError uses AI to analyze the error and suggest causes
func (fa *FailureAnalyzer) aiAnalyzeError(ctx context.Context, content string, logsContext string, errorLogs []models.ProcessingLog) ([]FailureCause, error) {
	prompt := fmt.Sprintf(`Analyze this journal processing failure and identify likely causes.

Example Analysis:
Journal Entry: "Today I learned about..."
Processing Logs: 
- ERROR: Failed to fetch URL: connection timeout after 30s
- INFO: Processing stage: fetching_urls
- ERROR: MCP agent request failed: Post "http://localhost:8081": dial tcp [::1]:8081: connect: connection refused

Expected Response:
Causes:
1. Cause: "MCP agent service not running" - Probability: 0.9 - Solution: "Start the MCP agent with 'make dev' or check if port 8081 is already in use"
2. Cause: "Network connectivity issue" - Probability: 0.7 - Solution: "Check network connection and firewall settings for localhost connections"
3. Cause: "URL fetch timeout too short" - Probability: 0.5 - Solution: "Increase timeout in MCP agent configuration or check if target URL is slow"

Now analyze this failure:
Journal Entry (first 500 chars):
%s

Processing Logs:
%s

Based on the error patterns and logs, identify the most likely causes of failure. Consider:
1. Service availability issues (Ollama, MCP agent)
2. Content-related issues (format, size, encoding, special characters)
3. Network or timeout issues
4. Configuration problems (wrong ports, missing environment variables)
5. Resource constraints (memory, CPU, disk space)

Important:
- Look for specific error messages in the logs
- Consider the processing stage where failure occurred
- Match error patterns to common issues
- Provide actionable solutions that users can implement

Provide up to 3 most likely causes with solutions.`,
		truncateString(content, 500),
		logsContext)

	// Use a simpler response format for better reliability
	type AIResponse struct {
		Causes []struct {
			Cause       string  `json:"cause"`
			Probability float64 `json:"probability"`
			Solution    string  `json:"solution"`
		} `json:"causes"`
	}

	responseJSON, err := fa.processor.ProcessWithSchema(ctx, prompt, AIResponse{})
	if err != nil {
		return nil, err
	}

	var response AIResponse
	if err := json.Unmarshal([]byte(responseJSON), &response); err != nil {
		return nil, err
	}

	// Convert to FailureCause slice
	var causes []FailureCause
	for _, c := range response.Causes {
		causes = append(causes, FailureCause{
			Cause:       c.Cause,
			Description: c.Cause, // AI typically provides descriptive causes
			Probability: c.Probability,
			Solution:    c.Solution,
		})
	}

	return causes, nil
}

// mergeAnalyses combines common patterns with AI insights
func (fa *FailureAnalyzer) mergeAnalyses(common []FailureCause, ai []FailureCause) []FailureCause {
	// Create a map to track unique causes
	causeMap := make(map[string]FailureCause)

	// Add common causes
	for _, cause := range common {
		causeMap[cause.Cause] = cause
	}

	// Add or update with AI insights
	for _, aiCause := range ai {
		if existing, exists := causeMap[aiCause.Cause]; exists {
			// Average the probabilities
			existing.Probability = (existing.Probability + aiCause.Probability) / 2
			causeMap[aiCause.Cause] = existing
		} else {
			causeMap[aiCause.Cause] = aiCause
		}
	}

	// Convert back to slice
	var merged []FailureCause
	for _, cause := range causeMap {
		merged = append(merged, cause)
	}

	return merged
}

// getTop80PercentCauses returns causes that cumulatively account for 80% probability
func (fa *FailureAnalyzer) getTop80PercentCauses(causes []FailureCause) []FailureCause {
	if len(causes) == 0 {
		return causes
	}

	var result []FailureCause
	cumulativeProbability := 0.0

	for _, cause := range causes {
		result = append(result, cause)
		cumulativeProbability += cause.Probability

		if cumulativeProbability >= 0.8 {
			break
		}
	}

	// Always include at least one cause
	if len(result) == 0 && len(causes) > 0 {
		result = []FailureCause{causes[0]}
	}

	return result
}

// generateRecommendation creates an actionable recommendation
func (fa *FailureAnalyzer) generateRecommendation(causes []FailureCause, stage models.ProcessingStage) string {
	if len(causes) == 0 {
		return "Unable to determine specific cause. Check service logs and retry the operation."
	}

	// Use the most likely cause for the primary recommendation
	primary := causes[0]

	recommendation := fmt.Sprintf("Most likely issue: %s\n\nRecommended action: %s",
		primary.Description,
		primary.Solution)

	if len(causes) > 1 {
		recommendation += "\n\nOther possible causes:"
		for i, cause := range causes[1:] {
			if i >= 2 { // Limit to top 3 total
				break
			}
			recommendation += fmt.Sprintf("\n- %s (%.0f%% likely)", cause.Cause, cause.Probability*100)
		}
	}

	return recommendation
}

// truncateString truncates a string to the specified length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
