# Search Evaluation System

This directory contains the evaluation framework for testing and measuring the performance of the journal's search functionality.

## Overview

The evaluation system provides comprehensive testing for all three search modes:
- **Classic Search**: Traditional keyword-based search with filters
- **Vector Search**: Semantic similarity search with three modes (similar, explore, contrast)
- **Hybrid Search**: Intelligent combination of classic and vector search

## Usage

### 1. Generate Test Data

Create synthetic journal entries with known characteristics:

```bash
make eval-generate
```

This creates 100 test entries with predefined topics, entities, sentiments, and keywords.

### 2. Run Evaluation

Execute the evaluation suite against all search modes:

```bash
make eval-run
```

Or evaluate a specific search mode:

```bash
cd backend && go run cmd/evaluate/main.go -cmd evaluate -mode classic
```

### 3. Generate Reports

Create an HTML report with visualizations:

```bash
make eval-report
```

Or generate other formats:

```bash
cd backend && go run cmd/evaluate/main.go -cmd report -format json
cd backend && go run cmd/evaluate/main.go -cmd report -format csv
```

### 4. Full Evaluation Pipeline

Run the complete evaluation pipeline:

```bash
make eval-full
```

This will:
1. Generate test data
2. Run evaluation for all search modes
3. Generate an HTML report

### 5. Clean Up

Remove all test data and reports:

```bash
make eval-clean
```

## Metrics

The evaluation system measures:

- **Precision**: Ratio of relevant results in the returned set
- **Recall**: Ratio of relevant results that were returned
- **F1 Score**: Harmonic mean of precision and recall
- **NDCG**: Normalized Discounted Cumulative Gain (ranking quality)
- **MRR**: Mean Reciprocal Rank (position of first relevant result)
- **Latency**: Average query response time in milliseconds

## Directory Structure

```
evaluation_results/
├── data/              # Test data sets
│   ├── entries.json   # Generated test entries
│   ├── classic_tests.json
│   ├── vector_tests.json
│   └── hybrid_tests.json
└── reports/           # Evaluation reports
    ├── evaluation_report_*.html
    ├── evaluation_report_*.json
    └── evaluation_report_*.csv
```

## Configuration

Adjust evaluation parameters:

- `-size`: Number of test entries to generate (default: 100)
- `-mode`: Search mode to evaluate (classic, vector, hybrid, all)
- `-format`: Report format (html, json, csv)
- `-output`: Output directory (default: evaluation_results)

## Test Cases

### Classic Search Tests
- Single keyword matching
- Topic-based search
- Entity search with exact phrases
- Filter combinations (favorites, collections, date ranges)

### Vector Search Tests
- Similar mode: Find entries with similar content
- Explore mode: Find conceptually related entries
- Contrast mode: Find opposing or different viewpoints

### Hybrid Search Tests
- Keyword + semantic combination
- Natural language queries
- Different weighting strategies (balanced, semantic_boost, precision, discovery)