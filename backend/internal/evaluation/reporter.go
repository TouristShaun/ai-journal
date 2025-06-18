package evaluation

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"time"
)

// Reporter generates evaluation reports in various formats
type Reporter struct {
	outputDir string
}

// NewReporter creates a new reporter instance
func NewReporter(outputDir string) *Reporter {
	return &Reporter{
		outputDir: outputDir,
	}
}

// GenerateHTMLReport creates an HTML report with visualizations
func (r *Reporter) GenerateHTMLReport(metrics map[string]*SearchMetrics) (string, error) {
	tmpl := `<!DOCTYPE html>
<html>
<head>
    <title>Search Evaluation Report</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            margin: 0;
            padding: 20px;
            background: #f5f5f5;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            padding: 30px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        h1 {
            color: #333;
            margin-bottom: 30px;
        }
        h2 {
            color: #555;
            margin-top: 40px;
            margin-bottom: 20px;
        }
        .timestamp {
            color: #666;
            font-size: 14px;
            margin-bottom: 30px;
        }
        .metrics-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
            gap: 20px;
            margin-bottom: 40px;
        }
        .metric-card {
            background: #f8f9fa;
            border: 1px solid #e9ecef;
            border-radius: 6px;
            padding: 20px;
        }
        .metric-card h3 {
            margin-top: 0;
            color: #495057;
            font-size: 18px;
        }
        .metric-value {
            font-size: 36px;
            font-weight: bold;
            color: #007bff;
            margin: 10px 0;
        }
        .metric-label {
            color: #6c757d;
            font-size: 14px;
        }
        table {
            width: 100%;
            border-collapse: collapse;
            margin-top: 20px;
        }
        th, td {
            text-align: left;
            padding: 12px;
            border-bottom: 1px solid #e9ecef;
        }
        th {
            background: #f8f9fa;
            font-weight: 600;
            color: #495057;
        }
        tr:hover {
            background: #f8f9fa;
        }
        .status-good {
            color: #28a745;
        }
        .status-warning {
            color: #ffc107;
        }
        .status-poor {
            color: #dc3545;
        }
        .chart-container {
            margin: 20px 0;
            padding: 20px;
            background: #f8f9fa;
            border-radius: 6px;
        }
        .bar {
            display: inline-block;
            height: 20px;
            background: #007bff;
            margin-right: 10px;
            border-radius: 3px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Search Evaluation Report</h1>
        <p class="timestamp">Generated: {{.Timestamp}}</p>
        
        <h2>Summary</h2>
        <div class="metrics-grid">
            {{range $mode, $metrics := .Metrics}}
            <div class="metric-card">
                <h3>{{$mode}} Search</h3>
                <div class="metric-value {{GetF1Status $metrics}}">{{printf "%.2f%%" (GetF1Percentage $metrics)}}</div>
                <div class="metric-label">F1 Score</div>
                <div style="margin-top: 15px;">
                    <div>Precision: {{printf "%.3f" $metrics.Precision}}</div>
                    <div>Recall: {{printf "%.3f" $metrics.Recall}}</div>
                    <div>Avg Latency: {{printf "%.1fms" $metrics.AvgLatency}}</div>
                </div>
            </div>
            {{end}}
        </div>

        <h2>Detailed Metrics</h2>
        <table>
            <thead>
                <tr>
                    <th>Search Mode</th>
                    <th>Precision</th>
                    <th>Recall</th>
                    <th>F1 Score</th>
                    <th>NDCG</th>
                    <th>MRR</th>
                    <th>Avg Latency</th>
                    <th>Test Cases</th>
                </tr>
            </thead>
            <tbody>
                {{range $mode, $metrics := .Metrics}}
                <tr>
                    <td><strong>{{$mode}}</strong></td>
                    <td>{{printf "%.3f" $metrics.Precision}}</td>
                    <td>{{printf "%.3f" $metrics.Recall}}</td>
                    <td>{{printf "%.3f" $metrics.F1Score}}</td>
                    <td>{{printf "%.3f" $metrics.NDCG}}</td>
                    <td>{{printf "%.3f" $metrics.MRR}}</td>
                    <td>{{printf "%.1fms" $metrics.AvgLatency}}</td>
                    <td>{{len $metrics.TestCases}}</td>
                </tr>
                {{end}}
            </tbody>
        </table>

        <h2>Performance Distribution</h2>
        <div class="chart-container">
            {{range $mode, $metrics := .Metrics}}
            <div style="margin: 10px 0;">
                <strong>{{$mode}}:</strong>
                <span class="bar" style="width: {{GetF1BarWidth $metrics}}px;"></span>
                <span>{{printf "%.1f%%" (GetF1Percentage $metrics)}}</span>
            </div>
            {{end}}
        </div>

        <h2>Test Case Results</h2>
        {{range $mode, $metrics := .Metrics}}
        <h3>{{$mode}} Search</h3>
        <table>
            <thead>
                <tr>
                    <th>Test ID</th>
                    <th>Query</th>
                    <th>Precision</th>
                    <th>Recall</th>
                    <th>Latency</th>
                    <th>Expected</th>
                    <th>Found</th>
                </tr>
            </thead>
            <tbody>
                {{range $i, $test := $metrics.TestCases}}
                {{if lt $i 10}}
                <tr>
                    <td>{{$test.TestID}}</td>
                    <td>{{GetTruncatedQuery $test}}</td>
                    <td>{{printf "%.2f" $test.Precision}}</td>
                    <td>{{printf "%.2f" $test.Recall}}</td>
                    <td>{{printf "%.0fms" (GetLatencyMS $test)}}</td>
                    <td>{{len $test.ExpectedIDs}}</td>
                    <td>{{len $test.ActualIDs}}</td>
                </tr>
                {{end}}
                {{end}}
            </tbody>
        </table>
        {{end}}
    </div>
</body>
</html>`

	// Create template functions
	funcMap := template.FuncMap{
		"GetF1Status": func(metrics *SearchMetrics) string {
			if metrics.F1Score >= 0.8 {
				return "status-good"
			} else if metrics.F1Score >= 0.6 {
				return "status-warning"
			}
			return "status-poor"
		},
		"GetF1Percentage": func(metrics *SearchMetrics) float64 {
			return metrics.F1Score * 100
		},
		"GetF1BarWidth": func(metrics *SearchMetrics) int {
			return int(metrics.F1Score * 300)
		},
		"GetTruncatedQuery": func(test TestResult) string {
			if len(test.Query) > 50 {
				return test.Query[:50] + "..."
			}
			return test.Query
		},
		"GetLatencyMS": func(test TestResult) float64 {
			return float64(test.Latency.Milliseconds())
		},
	}

	// Parse and execute template
	t, err := template.New("report").Funcs(funcMap).Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	data := struct {
		Timestamp time.Time
		Metrics   map[string]*SearchMetrics
	}{
		Timestamp: time.Now(),
		Metrics:   metrics,
	}

	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	// Save report
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("evaluation_report_%s.html", timestamp)
	filePath := filepath.Join(r.outputDir, "reports", filename)

	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return "", fmt.Errorf("failed to create reports directory: %w", err)
	}

	if err := os.WriteFile(filePath, buf.Bytes(), 0644); err != nil {
		return "", fmt.Errorf("failed to write report: %w", err)
	}

	return filePath, nil
}

// GenerateJSONReport creates a JSON report
func (r *Reporter) GenerateJSONReport(metrics map[string]*SearchMetrics) (string, error) {
	report := struct {
		GeneratedAt time.Time                 `json:"generated_at"`
		Summary     map[string]SummaryMetrics `json:"summary"`
		Detailed    map[string]*SearchMetrics `json:"detailed"`
	}{
		GeneratedAt: time.Now(),
		Summary:     make(map[string]SummaryMetrics),
		Detailed:    metrics,
	}

	// Create summary
	for mode, m := range metrics {
		report.Summary[mode] = SummaryMetrics{
			Precision:  m.Precision,
			Recall:     m.Recall,
			F1Score:    m.F1Score,
			NDCG:       m.NDCG,
			MRR:        m.MRR,
			AvgLatency: m.AvgLatency,
			TestCount:  len(m.TestCases),
		}
	}

	// Save report
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("evaluation_report_%s.json", timestamp)
	filePath := filepath.Join(r.outputDir, "reports", filename)

	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return "", fmt.Errorf("failed to create reports directory: %w", err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create report file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		return "", fmt.Errorf("failed to encode report: %w", err)
	}

	return filePath, nil
}

// GenerateCSVReport creates a CSV report
func (r *Reporter) GenerateCSVReport(metrics map[string]*SearchMetrics) (string, error) {
	// Save report
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("evaluation_report_%s.csv", timestamp)
	filePath := filepath.Join(r.outputDir, "reports", filename)

	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return "", fmt.Errorf("failed to create reports directory: %w", err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create report file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{
		"Search Mode", "Precision", "Recall", "F1 Score",
		"NDCG", "MRR", "Avg Latency (ms)", "Test Cases",
	}
	if err := writer.Write(header); err != nil {
		return "", fmt.Errorf("failed to write header: %w", err)
	}

	// Write metrics
	for mode, m := range metrics {
		row := []string{
			mode,
			fmt.Sprintf("%.3f", m.Precision),
			fmt.Sprintf("%.3f", m.Recall),
			fmt.Sprintf("%.3f", m.F1Score),
			fmt.Sprintf("%.3f", m.NDCG),
			fmt.Sprintf("%.3f", m.MRR),
			fmt.Sprintf("%.1f", m.AvgLatency),
			fmt.Sprintf("%d", len(m.TestCases)),
		}
		if err := writer.Write(row); err != nil {
			return "", fmt.Errorf("failed to write row: %w", err)
		}
	}

	// Write test case details
	writer.Write([]string{}) // Empty row
	writer.Write([]string{"Test Case Details"})
	writer.Write([]string{
		"Mode", "Test ID", "Query", "Precision", "Recall",
		"Latency (ms)", "Expected Count", "Actual Count",
	})

	for mode, m := range metrics {
		for _, test := range m.TestCases {
			row := []string{
				mode,
				test.TestID,
				test.Query,
				fmt.Sprintf("%.2f", test.Precision),
				fmt.Sprintf("%.2f", test.Recall),
				fmt.Sprintf("%.0f", float64(test.Latency.Milliseconds())),
				fmt.Sprintf("%d", len(test.ExpectedIDs)),
				fmt.Sprintf("%d", len(test.ActualIDs)),
			}
			if err := writer.Write(row); err != nil {
				return "", fmt.Errorf("failed to write test case: %w", err)
			}
		}
	}

	return filePath, nil
}

// SummaryMetrics for JSON report
type SummaryMetrics struct {
	Precision  float64 `json:"precision"`
	Recall     float64 `json:"recall"`
	F1Score    float64 `json:"f1_score"`
	NDCG       float64 `json:"ndcg"`
	MRR        float64 `json:"mrr"`
	AvgLatency float64 `json:"avg_latency_ms"`
	TestCount  int     `json:"test_count"`
}
