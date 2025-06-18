package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/journal/internal/db"
	"github.com/journal/internal/evaluation"
)

func main() {
	var (
		dbHost      = flag.String("host", "localhost", "PostgreSQL host")
		dbPort      = flag.String("port", "5432", "PostgreSQL port")
		dbUser      = flag.String("user", os.Getenv("USER"), "PostgreSQL user")
		dbPassword  = flag.String("password", "", "PostgreSQL password")
		dbName      = flag.String("dbname", "journal_db", "PostgreSQL database name")
		command     = flag.String("cmd", "", "Command to run: generate, evaluate, report")
		outputDir   = flag.String("output", "evaluation_results", "Output directory for results")
		testSetSize = flag.Int("size", 100, "Number of test entries to generate")
		searchMode  = flag.String("mode", "all", "Search mode to evaluate: classic, vector, hybrid, all")
		format      = flag.String("format", "html", "Report format: html, json, csv")
	)
	flag.Parse()

	// Override with environment variables if set
	if envUser := os.Getenv("DB_USER"); envUser != "" {
		*dbUser = envUser
	}
	if envHost := os.Getenv("DB_HOST"); envHost != "" {
		*dbHost = envHost
	}
	if envPort := os.Getenv("DB_PORT"); envPort != "" {
		*dbPort = envPort
	}
	if envName := os.Getenv("DB_NAME"); envName != "" {
		*dbName = envName
	}

	if *command == "" {
		log.Fatal("Command is required. Use -cmd flag with: generate, evaluate, or report")
	}

	// Connect to database
	database, err := db.NewConnection(*dbHost, *dbPort, *dbUser, *dbPassword, *dbName)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Create evaluator
	evaluator := evaluation.NewEvaluator(database, *outputDir)

	switch *command {
	case "generate":
		log.Printf("Generating %d test entries...", *testSetSize)
		if err := evaluator.GenerateTestData(*testSetSize); err != nil {
			log.Fatalf("Failed to generate test data: %v", err)
		}
		log.Println("Test data generation complete")

	case "evaluate":
		log.Printf("Evaluating search mode: %s", *searchMode)
		results, err := evaluator.RunEvaluation(*searchMode)
		if err != nil {
			log.Fatalf("Failed to run evaluation: %v", err)
		}

		// Print summary
		fmt.Printf("\nEvaluation Results Summary:\n")
		fmt.Printf("==========================\n")
		for mode, metrics := range results {
			fmt.Printf("\n%s Search:\n", mode)
			fmt.Printf("  Precision: %.3f\n", metrics.Precision)
			fmt.Printf("  Recall: %.3f\n", metrics.Recall)
			fmt.Printf("  F1 Score: %.3f\n", metrics.F1Score)
			fmt.Printf("  Avg Latency: %.2fms\n", metrics.AvgLatency)
		}

	case "report":
		log.Printf("Generating %s report...", *format)
		reportPath, err := evaluator.GenerateReport(*format)
		if err != nil {
			log.Fatalf("Failed to generate report: %v", err)
		}
		log.Printf("Report generated: %s", reportPath)

	default:
		log.Fatalf("Unknown command: %s. Use generate, evaluate, or report", *command)
	}
}
