package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	mcp "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/stdio"
)

// FetchArguments defines the schema for the fetch tool arguments
type FetchArguments struct {
	URL    string `json:"url" jsonschema:"description=The URL to fetch content from,required"`
	Reason string `json:"reason" jsonschema:"description=Why this URL is relevant to the journal entry"`
}

// FetchResult defines the structure of fetched content
type FetchResult struct {
	URL         string            `json:"url"`
	Title       string            `json:"title"`
	Content     string            `json:"content"`
	ExtractedAt time.Time         `json:"extracted_at"`
	Source      string            `json:"source"`
	Metadata    map[string]string `json:"metadata"`
}

// QwenAnalysis represents the AI analysis result
type QwenAnalysis struct {
	Summary      string   `json:"summary"`
	KeyPoints    []string `json:"key_points"`
	Relevance    string   `json:"relevance"`
	MainEntities []string `json:"main_entities"`
}

func main() {
	// Create a new MCP server
	server := mcp.NewServer(
		"journal-fetch-agent",
		"1.0.0",
		mcp.WithServerDescription("Fetches and analyzes web content for journal entries"),
	)

	// Register the fetch tool
	err := server.RegisterTool(
		"fetch_url",
		"Fetch and analyze content from a URL",
		fetchURLHandler,
	)
	if err != nil {
		log.Fatalf("Failed to register fetch tool: %v", err)
	}

	// Register a resource that shows fetched URLs
	err = server.RegisterResource(
		"fetched_urls",
		"List of all fetched URLs and their content",
		listFetchedURLsHandler,
	)
	if err != nil {
		log.Fatalf("Failed to register resource: %v", err)
	}

	// Create stdio transport
	transport := stdio.NewTransport()

	// Start the server
	log.Println("Starting MCP fetch agent...")
	if err := server.Serve(transport); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func fetchURLHandler(arguments FetchArguments) (*mcp.ToolResponse, error) {
	// Validate URL
	parsedURL, err := url.Parse(arguments.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// Ensure HTTPS
	if parsedURL.Scheme == "http" {
		parsedURL.Scheme = "https"
	}

	log.Printf("Fetching URL: %s (reason: %s)", parsedURL.String(), arguments.Reason)

	// Fetch the content
	result, err := fetchContent(parsedURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch content: %w", err)
	}

	// Analyze with Qwen if available
	analysis, err := analyzeWithQwen(result.Content, arguments.Reason)
	if err != nil {
		log.Printf("Warning: Failed to analyze with Qwen: %v", err)
	} else if analysis != nil {
		// Add analysis to metadata
		result.Metadata["ai_summary"] = analysis.Summary
		result.Metadata["relevance"] = analysis.Relevance
		if len(analysis.KeyPoints) > 0 {
			result.Metadata["key_points"] = strings.Join(analysis.KeyPoints, "; ")
		}
		if len(analysis.MainEntities) > 0 {
			result.Metadata["main_entities"] = strings.Join(analysis.MainEntities, ", ")
		}
	}

	// Store the result (in a real implementation, you'd persist this)
	storeResult(result)

	// Convert result to JSON
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return mcp.NewToolResponse(
		mcp.NewTextContent(string(resultJSON)),
	), nil
}

func fetchContent(parsedURL *url.URL) (*FetchResult, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", parsedURL.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Journal-MCP-Agent/1.0")
	req.Header.Set("Accept", "text/html,application/json,text/plain,*/*")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Read body (limit to 1MB)
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return nil, err
	}

	// Extract content based on content type
	contentType := resp.Header.Get("Content-Type")
	var title, content string
	metadata := make(map[string]string)

	if strings.Contains(contentType, "text/html") {
		title = extractTitle(string(body))
		content = extractTextContent(string(body))
		metadata["content_type"] = "html"
		
		// Extract meta description
		if desc := extractMetaDescription(string(body)); desc != "" {
			metadata["description"] = desc
		}
	} else if strings.Contains(contentType, "application/json") {
		var jsonData interface{}
		if err := json.Unmarshal(body, &jsonData); err == nil {
			prettyJSON, _ := json.MarshalIndent(jsonData, "", "  ")
			content = string(prettyJSON)
		} else {
			content = string(body)
		}
		title = "JSON Data"
		metadata["content_type"] = "json"
	} else {
		content = string(body)
		title = "Text Content"
		metadata["content_type"] = "text"
	}

	// Limit content length
	if len(content) > 5000 {
		content = content[:5000] + "\n\n... (content truncated)"
		metadata["truncated"] = "true"
	}

	metadata["status_code"] = fmt.Sprintf("%d", resp.StatusCode)
	metadata["fetched_at"] = time.Now().Format(time.RFC3339)

	return &FetchResult{
		URL:         parsedURL.String(),
		Title:       title,
		Content:     content,
		ExtractedAt: time.Now(),
		Source:      parsedURL.Host,
		Metadata:    metadata,
	}, nil
}

func analyzeWithQwen(content, reason string) (*QwenAnalysis, error) {
	// This would integrate with your Qwen/Ollama service
	// For now, returning a placeholder
	// In real implementation, you'd call your Ollama API here
	
	// Example of what this would look like:
	/*
	client := ollama.NewClient("http://localhost:11434")
	prompt := fmt.Sprintf(`Analyze the following web content in the context of: %s

Content:
%s

Provide:
1. A brief summary (2-3 sentences)
2. Key points relevant to the context
3. How relevant this content is to the context
4. Main entities mentioned`, reason, content)

	response, err := client.Generate("qwen2.5:7b", prompt)
	if err != nil {
		return nil, err
	}
	
	// Parse response into QwenAnalysis struct
	*/
	
	return nil, fmt.Errorf("Qwen integration not implemented yet")
}

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
	title := html[start : start+end]
	// Clean up HTML entities
	title = strings.ReplaceAll(title, "&amp;", "&")
	title = strings.ReplaceAll(title, "&lt;", "<")
	title = strings.ReplaceAll(title, "&gt;", ">")
	title = strings.ReplaceAll(title, "&quot;", "\"")
	title = strings.ReplaceAll(title, "&#39;", "'")
	return strings.TrimSpace(title)
}

func extractMetaDescription(html string) string {
	// Look for meta description tag
	metaStart := strings.Index(html, `<meta name="description"`)
	if metaStart == -1 {
		metaStart = strings.Index(html, `<meta property="og:description"`)
	}
	if metaStart == -1 {
		return ""
	}
	
	contentStart := strings.Index(html[metaStart:], `content="`)
	if contentStart == -1 {
		return ""
	}
	contentStart += metaStart + 9
	
	contentEnd := strings.Index(html[contentStart:], `"`)
	if contentEnd == -1 {
		return ""
	}
	
	return html[contentStart : contentStart+contentEnd]
}

func extractTextContent(html string) string {
	// Remove script and style tags with their content
	content := removeTagsWithContent(html, "script")
	content = removeTagsWithContent(content, "style")
	content = removeTagsWithContent(content, "noscript")
	
	// Remove all HTML tags but preserve line breaks
	result := ""
	inTag := false
	for i, ch := range content {
		if ch == '<' {
			inTag = true
			// Check if this is a block element that should add a line break
			if i+3 < len(content) {
				tag := strings.ToLower(content[i:min(i+10, len(content))])
				if strings.HasPrefix(tag, "<p") || strings.HasPrefix(tag, "<br") ||
					strings.HasPrefix(tag, "<div") || strings.HasPrefix(tag, "<h1") ||
					strings.HasPrefix(tag, "<h2") || strings.HasPrefix(tag, "<h3") ||
					strings.HasPrefix(tag, "<h4") || strings.HasPrefix(tag, "<h5") ||
					strings.HasPrefix(tag, "<h6") || strings.HasPrefix(tag, "<li") ||
					strings.HasPrefix(tag, "</p") || strings.HasPrefix(tag, "</div") {
					result += "\n"
				}
			}
		} else if ch == '>' {
			inTag = false
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
	
	// Remove duplicate empty lines
	finalLines := []string{}
	prevEmpty := false
	for _, line := range cleanLines {
		if line == "" {
			if !prevEmpty {
				finalLines = append(finalLines, line)
			}
			prevEmpty = true
		} else {
			finalLines = append(finalLines, line)
			prevEmpty = false
		}
	}
	
	return strings.Join(finalLines, "\n")
}

func removeTagsWithContent(html, tagName string) string {
	result := html
	for {
		start := strings.Index(strings.ToLower(result), "<"+tagName)
		if start == -1 {
			break
		}
		end := strings.Index(strings.ToLower(result[start:]), "</"+tagName+">")
		if end == -1 {
			break
		}
		end += start + len(tagName) + 3
		result = result[:start] + result[end:]
	}
	return result
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// In-memory storage for demo purposes
var fetchedURLs = make(map[string]*FetchResult)

func storeResult(result *FetchResult) {
	fetchedURLs[result.URL] = result
}

func listFetchedURLsHandler() (*mcp.ResourceResponse, error) {
	urls := make([]map[string]interface{}, 0, len(fetchedURLs))
	for url, result := range fetchedURLs {
		urls = append(urls, map[string]interface{}{
			"url":          url,
			"title":        result.Title,
			"fetched_at":   result.ExtractedAt.Format(time.RFC3339),
			"content_size": len(result.Content),
			"metadata":     result.Metadata,
		})
	}
	
	data, err := json.MarshalIndent(urls, "", "  ")
	if err != nil {
		return nil, err
	}
	
	return mcp.NewResourceResponse(
		mcp.NewTextContent(string(data)),
		"application/json",
	), nil
}