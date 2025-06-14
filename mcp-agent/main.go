package main

import (
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

type FetchParams struct {
	URL    string `json:"url"`
	Reason string `json:"reason"`
}

type FetchResult struct {
	URL         string    `json:"url"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	ExtractedAt time.Time `json:"extracted_at"`
	Source      string    `json:"source"`
}

func fetchURL(ctx context.Context, params FetchParams) (*FetchResult, error) {
	// Validate URL
	parsedURL, err := url.Parse(params.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// Ensure HTTPS
	if parsedURL.Scheme == "http" {
		parsedURL.Scheme = "https"
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", parsedURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set user agent
	req.Header.Set("User-Agent", "Journal-MCP-Agent/1.0")

	// Fetch the URL
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Read the body (limit to 1MB)
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Extract content based on content type
	contentType := resp.Header.Get("Content-Type")
	var title, content string

	if strings.Contains(contentType, "text/html") {
		// For HTML, we'd normally parse it properly
		// For now, just extract basic content
		title = extractTitle(string(body))
		content = extractTextContent(string(body))
	} else if strings.Contains(contentType, "application/json") {
		// Pretty print JSON
		var jsonData interface{}
		if err := json.Unmarshal(body, &jsonData); err == nil {
			prettyJSON, _ := json.MarshalIndent(jsonData, "", "  ")
			content = string(prettyJSON)
		} else {
			content = string(body)
		}
		title = "JSON Data"
	} else {
		// Plain text or other
		content = string(body)
		title = "Text Content"
	}

	// Limit content length
	if len(content) > 5000 {
		content = content[:5000] + "... (truncated)"
	}

	return &FetchResult{
		URL:         parsedURL.String(),
		Title:       title,
		Content:     content,
		ExtractedAt: time.Now(),
		Source:      parsedURL.Host,
	}, nil
}

// Simple HTML title extraction
func extractTitle(html string) string {
	start := strings.Index(html, "<title>")
	if start == -1 {
		return "Untitled"
	}
	start += 7
	end := strings.Index(html[start:], "</title>")
	if end == -1 {
		return "Untitled"
	}
	return strings.TrimSpace(html[start : start+end])
}

// Simple text content extraction (removes HTML tags)
func extractTextContent(html string) string {
	// Remove script and style tags
	content := html
	for {
		start := strings.Index(content, "<script")
		if start == -1 {
			break
		}
		end := strings.Index(content[start:], "</script>")
		if end == -1 {
			break
		}
		content = content[:start] + content[start+end+9:]
	}

	for {
		start := strings.Index(content, "<style")
		if start == -1 {
			break
		}
		end := strings.Index(content[start:], "</style>")
		if end == -1 {
			break
		}
		content = content[:start] + content[start+end+8:]
	}

	// Remove all HTML tags
	result := ""
	inTag := false
	for _, ch := range content {
		if ch == '<' {
			inTag = true
		} else if ch == '>' {
			inTag = false
			result += " "
		} else if !inTag {
			result += string(ch)
		}
	}

	// Clean up whitespace
	lines := strings.Split(result, "\n")
	cleanLines := []string{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleanLines = append(cleanLines, line)
		}
	}

	return strings.Join(cleanLines, "\n")
}

func main() {
	// HTTP handler for fetching URLs
	http.HandleFunc("/fetch", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var params FetchParams
		if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		log.Printf("Fetching URL: %s (reason: %s)", params.URL, params.Reason)

		result, err := fetchURL(r.Context(), params)
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

	log.Printf("Starting MCP agent HTTP server on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("HTTP server error: %v", err)
	}
}