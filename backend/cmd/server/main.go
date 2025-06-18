package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/journal/internal/db"
	"github.com/journal/internal/events"
	"github.com/journal/internal/handlers"
	"github.com/journal/internal/jsonrpc"
	"github.com/journal/internal/logger"
	"github.com/journal/internal/mcp"
	"github.com/journal/internal/ollama"
	"github.com/journal/internal/service"
)

func main() {
	// Set up panic recovery
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC RECOVERED: %v", r)
			log.Printf("Stack trace: %s", debug.Stack())
		}
	}()

	// Database configuration
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "")
	dbName := getEnv("DB_NAME", "journal_db")

	// Connect to database
	database, err := db.NewConnection(dbHost, dbPort, dbUser, dbPassword, dbName)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Migrations should be run separately using make db-migrate
	log.Println("Database connected. Run 'make db-migrate' to update schema.")

	// Initialize Ollama client
	ollamaURL := getEnv("OLLAMA_URL", "http://localhost:11434")
	ollamaClient := ollama.NewClient(ollamaURL)
	processor := ollama.NewProcessor(ollamaClient)

	// Initialize MCP client
	mcpURL := getEnv("MCP_AGENT_URL", "http://localhost:8081")
	mcpClient := mcp.NewClient(mcpURL)

	// Initialize event broadcaster
	broadcaster := events.NewBroadcaster()
	broadcaster.Start()

	// Initialize processing logger
	processingLogger := logger.NewProcessingLogger(database.DB)

	// Initialize services
	journalService := service.NewJournalService(database, processor, mcpClient, broadcaster, processingLogger)

	// Initialize handlers
	journalHandlers := handlers.NewJournalHandlers(journalService)
	evaluationHandler := handlers.NewEvaluationHandler(database, broadcaster, journalService)

	// Create JSON-RPC server
	rpcServer := jsonrpc.NewServer()

	// Register journal methods
	rpcServer.RegisterMethod("journal.create", journalHandlers.CreateEntry)
	rpcServer.RegisterMethod("journal.update", journalHandlers.UpdateEntry)
	rpcServer.RegisterMethod("journal.get", journalHandlers.GetEntry)
	rpcServer.RegisterMethod("journal.search", journalHandlers.Search)
	rpcServer.RegisterMethod("journal.toggleFavorite", journalHandlers.ToggleFavorite)
	rpcServer.RegisterMethod("journal.getProcessingLogs", journalHandlers.GetProcessingLogs)
	rpcServer.RegisterMethod("journal.analyzeFailure", journalHandlers.AnalyzeFailure)
	rpcServer.RegisterMethod("journal.retryProcessing", journalHandlers.RetryProcessing)
	rpcServer.RegisterMethod("journal.getSearchSuggestions", journalHandlers.GetSearchSuggestions)
	
	// Register collection methods
	rpcServer.RegisterMethod("collection.create", journalHandlers.CreateCollection)
	rpcServer.RegisterMethod("collection.list", journalHandlers.GetCollections)
	rpcServer.RegisterMethod("collection.addEntry", journalHandlers.AddToCollection)
	rpcServer.RegisterMethod("collection.removeEntry", journalHandlers.RemoveFromCollection)
	
	// Register evaluation methods
	evaluationHandler.Register(rpcServer)

	// Create HTTP router
	router := mux.NewRouter()

	// CORS middleware
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Cache-Control")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	// Routes
	router.Handle("/api/rpc", rpcServer).Methods("POST", "OPTIONS")
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "healthy"}`))
	}).Methods("GET")

	// SSE endpoint
	router.HandleFunc("/api/events", func(w http.ResponseWriter, r *http.Request) {
		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no") // Disable Nginx buffering

		// Generate client ID
		clientID := uuid.New().String()

		// Register client
		client := broadcaster.RegisterClient(clientID)
		defer broadcaster.UnregisterClient(client)

		// Create flusher
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		// Send initial connection event
		fmt.Fprintf(w, "event: connected\ndata: {\"client_id\":\"%s\"}\n\n", clientID)
		flusher.Flush()

		// Create a ticker for heartbeat
		heartbeat := time.NewTicker(30 * time.Second)
		defer heartbeat.Stop()

		for {
			select {
			case event := <-client.Events:
				// Send event to client
				if sseData, err := events.FormatSSE(event); err == nil {
					fmt.Fprint(w, sseData)
					flusher.Flush()
				}

			case <-heartbeat.C:
				// Send heartbeat to keep connection alive
				fmt.Fprint(w, ":heartbeat\n\n")
				flusher.Flush()

			case <-r.Context().Done():
				// Client disconnected
				return
			}
		}
	}).Methods("GET")

	// Export endpoint
	router.HandleFunc("/api/export", func(w http.ResponseWriter, r *http.Request) {
		// Parse query parameters
		format := r.URL.Query().Get("format")
		if format == "" {
			format = "json"
		}

		// Build search params from query
		params := service.SearchParams{
			Query: r.URL.Query().Get("query"),
			Limit: 1000, // Export up to 1000 entries
		}

		// Handle favorites filter
		if fav := r.URL.Query().Get("is_favorite"); fav == "true" {
			favBool := true
			params.IsFavorite = &favBool
		}

		// Handle collection IDs
		if collections := r.URL.Query().Get("collection_ids"); collections != "" {
			params.CollectionIDs = strings.Split(collections, ",")
		}

		// Export entries
		data, contentType, err := journalService.ExportEntries(params, format)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Set headers
		filename := fmt.Sprintf("journal-export-%s.%s", time.Now().Format("2006-01-02"), format)
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

		// Write data
		w.Write(data)
	}).Methods("GET", "OPTIONS")

	// Start server
	port := getEnv("PORT", "8080")
	log.Printf("Starting journal server on port %s", port)

	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
