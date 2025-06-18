package ollama

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/journal/internal/models"
)

type Processor struct {
	client *Client
}

func NewProcessor(client *Client) *Processor {
	return &Processor{client: client}
}

// JournalAnalysis represents the structured output from Qwen
type JournalAnalysis struct {
	Summary   string         `json:"summary"`
	Entities  []string       `json:"entities"`
	Topics    []string       `json:"topics"`
	Sentiment string         `json:"sentiment"`
	URLs      []URLToFetch   `json:"urls_to_fetch"`
	Metadata  map[string]any `json:"metadata"`
}

type URLToFetch struct {
	URL    string `json:"url"`
	Reason string `json:"reason"`
}

// ProcessJournalEntry analyzes a journal entry and returns structured data
func (p *Processor) ProcessJournalEntry(content string) (*models.ProcessedData, error) {
	// Define the JSON schema for structured output
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"summary": {"type": "string", "description": "A brief summary of the journal entry"},
			"entities": {"type": "array", "items": {"type": "string"}, "description": "Named entities mentioned (people, places, organizations)"},
			"topics": {"type": "array", "items": {"type": "string"}, "description": "Main topics or themes"},
			"sentiment": {"type": "string", "enum": ["positive", "negative", "neutral", "mixed"], "description": "Overall sentiment"},
			"urls_to_fetch": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"url": {"type": "string"},
						"reason": {"type": "string", "description": "Why this URL is relevant to the entry"}
					},
					"required": ["url", "reason"]
				},
				"description": "URLs mentioned that should be fetched for additional context"
			},
			"metadata": {"type": "object", "description": "Additional metadata extracted from the entry"}
		},
		"required": ["summary", "entities", "topics", "sentiment", "urls_to_fetch", "metadata"]
	}`)

	prompt := fmt.Sprintf(`Analyze the following journal entry and extract structured information according to the provided schema.

Example Analysis:
Journal Entry: "Had an amazing meeting with Sarah Chen from TechCorp today at their Seattle office. We discussed the new AI project and she seemed really excited about our proposal. Check out their recent blog post about ML trends: https://techcorp.com/blog/ml-2024. Feeling optimistic about this partnership!"

Expected Output:
- Summary: "Successful meeting with TechCorp representative about AI project proposal, positive reception"
- Entities: ["Sarah Chen" (person), "TechCorp" (organization), "Seattle" (place)]
- Topics: ["business meeting", "AI project", "partnership", "machine learning"]
- Sentiment: "positive"
- URLs: ["https://techcorp.com/blog/ml-2024"]

Now analyze this journal entry:
%s

Extract:
1. A concise summary (2-3 sentences max)
2. Named entities (people, places, organizations, products) - be specific
3. Main topics or themes (3-5 most relevant)
4. Overall sentiment (positive, negative, neutral, or mixed)
5. Any URLs mentioned that would provide valuable context
6. Additional metadata that might be useful for search and organization

Important:
- Keep summaries factual and concise
- Extract full names when mentioned
- Identify specific locations, not just general areas
- For sentiment, consider the overall emotional tone
- Only extract complete, valid URLs`, content)

	request := ChatRequest{
		Model: "qwen3:8b",
		Messages: []Message{
			{Role: "user", Content: prompt},
		},
		Format: schema,
		Stream: false,
		Options: Options{
			Temperature: 0.3,
		},
	}

	response, err := p.client.Chat(request)
	if err != nil {
		return nil, fmt.Errorf("failed to process with Qwen: %w", err)
	}

	// Log raw response for debugging
	log.Printf("Raw Qwen response: %s", response.Message.Content)

	var analysis JournalAnalysis
	if err := json.Unmarshal([]byte(response.Message.Content), &analysis); err != nil {
		log.Printf("Failed to parse JSON response: %v\nResponse was: %s", err, response.Message.Content)
		return nil, fmt.Errorf("failed to parse Qwen response: %w", err)
	}

	log.Printf("Processed journal entry: found %d entities, %d topics, %d URLs",
		len(analysis.Entities), len(analysis.Topics), len(analysis.URLs))

	// Debug log entities
	if len(analysis.Entities) > 0 {
		log.Printf("Entities: %v", analysis.Entities)
	} else {
		log.Printf("No entities found in response")
	}

	// Convert to ProcessedData model
	processedData := &models.ProcessedData{
		Summary:       analysis.Summary,
		Entities:      analysis.Entities,
		Topics:        analysis.Topics,
		Sentiment:     analysis.Sentiment,
		Metadata:      analysis.Metadata,
		ExtractedURLs: make([]models.ExtractedURL, 0, len(analysis.URLs)),
	}

	// Initialize metadata if nil
	if processedData.Metadata == nil {
		processedData.Metadata = make(map[string]any)
	}

	// Convert URLs to ExtractedURL format
	for _, url := range analysis.URLs {
		processedData.ExtractedURLs = append(processedData.ExtractedURLs, models.ExtractedURL{
			URL:     url.URL,
			Title:   url.Reason, // Use reason as temporary title
			Content: "",         // Will be filled by MCP agent
		})

		// Also keep in metadata for backward compatibility
		processedData.Metadata["url_"+url.URL] = map[string]interface{}{
			"url":    url.URL,
			"reason": url.Reason,
			"status": "pending_fetch",
		}
	}

	return processedData, nil
}

// CreateEmbedding generates embeddings for journal entry with metadata
func (p *Processor) CreateEmbedding(entry models.JournalEntry) ([]float32, error) {
	// Combine content with metadata for richer embeddings
	embeddingText := fmt.Sprintf("%s\n\nSummary: %s\nTopics: %s\nEntities: %s\nSentiment: %s",
		entry.Content,
		entry.ProcessedData.Summary,
		strings.Join(entry.ProcessedData.Topics, ", "),
		strings.Join(entry.ProcessedData.Entities, ", "),
		entry.ProcessedData.Sentiment,
	)

	// Add extracted URL content if available
	for _, url := range entry.ProcessedData.ExtractedURLs {
		embeddingText += fmt.Sprintf("\n\nFrom %s: %s", url.URL, url.Title)
	}

	embeddings, err := p.client.CreateEmbedding("nomic-embed-text", embeddingText)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding: %w", err)
	}

	return embeddings, nil
}

// ProcessWithSchema processes a prompt and returns structured JSON according to the provided schema
func (p *Processor) ProcessWithSchema(ctx context.Context, prompt string, schemaExample interface{}) (string, error) {
	// Generate schema from the example struct
	schemaJSON, err := json.Marshal(schemaExample)
	if err != nil {
		return "", fmt.Errorf("failed to marshal schema: %w", err)
	}

	request := ChatRequest{
		Model: "qwen3:8b",
		Messages: []Message{
			{Role: "user", Content: prompt},
		},
		Format: json.RawMessage(schemaJSON),
		Stream: false,
		Options: Options{
			Temperature: 0.3,
		},
	}

	response, err := p.client.Chat(request)
	if err != nil {
		return "", fmt.Errorf("failed to process with Qwen: %w", err)
	}

	return response.Message.Content, nil
}
