package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// HTTPFetchParams matches what the journal backend expects
type HTTPFetchParams struct {
	URL    string `json:"url"`
	Reason string `json:"reason"`
}

// HTTPFetchResult matches the ExtractedURL model in the journal backend
type HTTPFetchResult struct {
	URL         string    `json:"url"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	ExtractedAt time.Time `json:"extracted_at"`
	Source      string    `json:"source"`
}

// QwenRequest represents a request to Ollama/Qwen
type QwenRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
	Options struct {
		Temperature float32 `json:"temperature"`
	} `json:"options"`
}

// QwenResponse represents the response from Ollama
type QwenResponse struct {
	Response string `json:"response"`
}

func startHTTPServer() {
	// HTTP handler for fetching URLs
	http.HandleFunc("/fetch", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var params HTTPFetchParams
		if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		log.Printf("Fetching URL: %s (reason: %s)", params.URL, params.Reason)

		result, err := fetchURLWithAnalysis(r.Context(), params)
		if err != nil {
			log.Printf("Error fetching URL: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Printf("Successfully fetched URL: %s (title: %s, content length: %d)", 
			result.URL, result.Title, len(result.Content))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "healthy"}`))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("Starting HTTP server on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("HTTP server error: %v", err)
	}
}

func fetchURLWithAnalysis(ctx context.Context, params HTTPFetchParams) (*HTTPFetchResult, error) {
	// Validate URL
	parsedURL, err := url.Parse(params.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// Ensure HTTPS
	if parsedURL.Scheme == "http" {
		parsedURL.Scheme = "https"
	}

	// Fetch the content
	content, title, err := fetchWebContent(ctx, parsedURL)
	if err != nil {
		return nil, err
	}

	// Analyze with Qwen to create a summary
	summary, err := analyzeContentWithQwen(content, params.Reason, title, parsedURL.String())
	if err != nil {
		log.Printf("Warning: Failed to analyze with Qwen: %v", err)
		// Don't fail if Qwen analysis fails, just use the raw content
		summary = content
	}

	result := &HTTPFetchResult{
		URL:         parsedURL.String(),
		Title:       title,
		Content:     summary, // Use the AI-enhanced summary
		ExtractedAt: time.Now(),
		Source:      parsedURL.Host,
	}

	return result, nil
}

func fetchWebContent(ctx context.Context, parsedURL *url.URL) (string, string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", parsedURL.String(), nil)
	if err != nil {
		return "", "", err
	}

	req.Header.Set("User-Agent", "Journal-MCP-Agent/1.0")
	req.Header.Set("Accept", "text/html,application/json,text/plain,*/*")

	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Read body (limit to 1MB)
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return "", "", err
	}

	// Extract content based on content type
	contentType := resp.Header.Get("Content-Type")
	var title, content string

	if strings.Contains(contentType, "text/html") {
		title = extractTitle(string(body))
		content = extractTextContent(string(body))
	} else if strings.Contains(contentType, "application/json") {
		var jsonData interface{}
		if err := json.Unmarshal(body, &jsonData); err == nil {
			prettyJSON, _ := json.MarshalIndent(jsonData, "", "  ")
			content = string(prettyJSON)
		} else {
			content = string(body)
		}
		title = "JSON Data from " + parsedURL.Host
	} else {
		content = string(body)
		title = "Content from " + parsedURL.Host
	}

	// Limit content length for processing
	if len(content) > 10000 {
		content = content[:10000] + "\n\n... (content truncated for processing)"
	}

	return content, title, nil
}

func analyzeContentWithQwen(content, reason, title, url string) (string, error) {
	// Call Ollama API with Qwen
	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}

	prompt := fmt.Sprintf(`You are analyzing web content for a journal entry. The user mentioned this URL because: "%s"

URL: %s
Title: %s

Content:
%s

Please provide a concise summary (200-300 words) that:
1. Captures the main points of the content
2. Highlights information relevant to why the user included this URL
3. Extracts key facts, insights, or quotes
4. Notes any important dates, names, or statistics mentioned

Format your response as a clear, readable summary that will be embedded alongside the journal entry.`, reason, url, title, content)

	reqBody := QwenRequest{
		Model:  "qwen2.5:7b",
		Prompt: prompt,
		Stream: false,
	}
	reqBody.Options.Temperature = 0.3

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return content, err
	}

	resp, err := http.Post(ollamaURL+"/api/generate", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return content, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return content, fmt.Errorf("Ollama returned status %d", resp.StatusCode)
	}

	var qwenResp QwenResponse
	if err := json.NewDecoder(resp.Body).Decode(&qwenResp); err != nil {
		return content, err
	}

	// Return the AI-generated summary
	return fmt.Sprintf("## %s\n\n%s\n\n---\n*Source: %s*", title, qwenResp.Response, url), nil
}