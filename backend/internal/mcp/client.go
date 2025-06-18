package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/journal/internal/models"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = "http://localhost:8081"
	}

	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type FetchRequest struct {
	URL    string `json:"url"`
	Reason string `json:"reason"`
}

func (c *Client) FetchURL(ctx context.Context, url, reason string) (*models.ExtractedURL, error) {
	request := FetchRequest{
		URL:    url,
		Reason: reason,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/fetch", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result models.ExtractedURL
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// FetchURLsForEntry fetches all URLs mentioned in a journal entry's processed data
func (c *Client) FetchURLsForEntry(ctx context.Context, entry *models.JournalEntry) error {
	// Look for URLs to fetch in metadata
	for key, value := range entry.ProcessedData.Metadata {
		if urlData, ok := value.(map[string]interface{}); ok {
			if status, exists := urlData["status"]; exists && status == "pending_fetch" {
				if url, urlOk := urlData["url"].(string); urlOk {
					reason := ""
					if r, rOk := urlData["reason"].(string); rOk {
						reason = r
					}

					// Fetch the URL
					extracted, err := c.FetchURL(ctx, url, reason)
					if err != nil {
						// Log error but continue with other URLs
						entry.ProcessedData.Metadata[key] = map[string]string{
							"url":    url,
							"reason": reason,
							"status": "fetch_failed",
							"error":  err.Error(),
						}
						continue
					}

					// Add to extracted URLs
					entry.ProcessedData.ExtractedURLs = append(entry.ProcessedData.ExtractedURLs, *extracted)

					// Update metadata
					entry.ProcessedData.Metadata[key] = map[string]string{
						"url":    url,
						"reason": reason,
						"status": "fetched",
					}
				}
			}
		}
	}

	return nil
}
