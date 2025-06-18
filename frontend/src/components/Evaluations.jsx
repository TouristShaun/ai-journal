import { useState, useEffect } from 'react';
import { X, Play, FileText, BarChart3, Loader2, Download, RefreshCw } from 'lucide-react';
import { journalAPI } from '../api/client';
import { useEventStream } from '../hooks/useEventStream';

function Evaluations({ isOpen, onClose }) {
  const [isRunning, setIsRunning] = useState(false);
  const [status, setStatus] = useState('');
  const [results, setResults] = useState(null);
  const [error, setError] = useState(null);
  const [progress, setProgress] = useState(0);
  const [currentStage, setCurrentStage] = useState('');

  // Subscribe to evaluation events
  useEventStream((event) => {
    if (event.type.startsWith('evaluation.')) {
      handleEvaluationEvent(event);
    }
  });

  useEffect(() => {
    if (isOpen) {
      // Check for existing evaluation results
      checkExistingResults();
    }
  }, [isOpen]);

  const checkExistingResults = async () => {
    try {
      const latestResults = await journalAPI.getLatestResults();
      if (latestResults.found && latestResults.metrics) {
        setResults(latestResults.metrics);
        // Find the latest timestamp from the metrics
        let latestTime = null;
        Object.values(latestResults.metrics).forEach(metric => {
          if (metric.timestamp && (!latestTime || new Date(metric.timestamp) > new Date(latestTime))) {
            latestTime = metric.timestamp;
          }
        });
        if (latestTime) {
          setStatus(`Last evaluation: ${new Date(latestTime).toLocaleString()}`);
        }
      }
    } catch (err) {
      console.error('Failed to check existing results:', err);
    }
  };

  const handleEvaluationEvent = (event) => {
    switch (event.type) {
      case 'evaluation.generate.started':
        setCurrentStage('Generating test data...');
        setProgress(10);
        break;
      case 'evaluation.generate.completed':
        setCurrentStage('Test data generated');
        setProgress(25);
        break;
      case 'evaluation.run.progress':
        setCurrentStage(`Evaluating ${event.data.mode} search...`);
        setProgress(25 + (event.data.progress * 0.5));
        break;
      case 'evaluation.run.completed':
        setCurrentStage('Evaluation completed');
        setProgress(75);
        break;
      case 'evaluation.full.progress':
        setCurrentStage(event.data.message);
        // Map stages to progress percentage
        if (event.data.stage === 'generating_data') {
          setProgress(25);
        } else if (event.data.stage === 'running_tests') {
          setProgress(50);
        } else if (event.data.stage === 'generating_reports') {
          setProgress(75);
        }
        break;
      case 'evaluation.full.completed':
        setCurrentStage('Full evaluation completed!');
        setProgress(100);
        // Refresh results from API
        checkExistingResults();
        break;
      case 'evaluation.generate.failed':
      case 'evaluation.run.failed':
        setError(`Evaluation failed: ${event.data.error}`);
        setIsRunning(false);
        break;
    }
  };

  const runEvaluation = async (mode = 'all') => {
    setIsRunning(true);
    setError(null);
    setStatus('Starting evaluation...');
    setProgress(0);
    setResults(null); // Clear previous results

    try {
      // Run full evaluation pipeline
      const result = await journalAPI.runFullEvaluation(100);
      
      // Results will be updated via SSE events
      setStatus('Evaluation complete!');
      
      // The results are now available from the evaluation
      if (result.metrics) {
        setResults(result.metrics);
      }
      
      // Generate report automatically
      const report = await journalAPI.generateReport('html');
      console.log('Report generated:', report.file_path);
    } catch (err) {
      setError(err.message || 'Failed to run evaluation. Please check the backend logs.');
      console.error('Evaluation error:', err);
    } finally {
      setIsRunning(false);
      setProgress(0);
      setCurrentStage('');
    }
  };

  const generateReport = async (format = 'html') => {
    console.log('generateReport called with format:', format);
    try {
      setError(null);
      setStatus(`Generating ${format.toUpperCase()} report...`);
      const report = await journalAPI.generateReport(format);
      console.log('Report generated:', report);
      setStatus(`${format.toUpperCase()} report generated: ${report.file_path || 'evaluation_results/reports/'}`);
    } catch (err) {
      console.error('Report generation error:', err);
      setError(`Failed to generate report: ${err.message}`);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-gray-800 rounded-lg w-full max-w-3xl max-h-[80vh] flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-gray-200 dark:border-gray-700">
          <h2 className="text-xl font-semibold text-gray-900 dark:text-white flex items-center gap-2">
            <BarChart3 className="w-5 h-5" />
            Search Evaluations
          </h2>
          <button
            onClick={onClose}
            className="p-1 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
          >
            <X className="w-5 h-5 text-gray-500 dark:text-gray-400" />
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-6">
          {/* Description */}
          <div className="mb-6">
            <p className="text-gray-600 dark:text-gray-400">
              Run comprehensive evaluations of the search functionality to measure performance,
              accuracy, and quality across all search modes.
            </p>
          </div>

          {/* Run Evaluation */}
          {!results && (
            <div className="text-center py-8">
              <button
                onClick={runEvaluation}
                disabled={isRunning}
                className="flex items-center gap-2 px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed mx-auto"
              >
                {isRunning ? (
                  <>
                    <Loader2 className="w-5 h-5 animate-spin" />
                    Running Evaluation...
                  </>
                ) : (
                  <>
                    <Play className="w-5 h-5" />
                    Run Full Evaluation
                  </>
                )}
              </button>
              
              {status && (
                <p className="mt-4 text-sm text-gray-600 dark:text-gray-400">
                  {status}
                </p>
              )}
              
              {currentStage && (
                <div className="mt-4">
                  <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">
                    {currentStage}
                  </p>
                  <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                    <div
                      className="bg-blue-600 h-2 rounded-full transition-all duration-300"
                      style={{ width: `${progress}%` }}
                    />
                  </div>
                </div>
              )}
              
              {error && (
                <div className="mt-4 p-4 bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400 rounded-lg max-w-md mx-auto">
                  {error}
                </div>
              )}
            </div>
          )}

          {/* Results */}
          {results && (
            <div className="space-y-6">
              <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
                Evaluation Results
              </h3>

              {/* Search Mode Results */}
              <div className="grid gap-4">
                {Object.entries(results).map(([mode, metrics]) => (
                  <div
                    key={mode}
                    className="bg-gray-50 dark:bg-gray-900 p-4 rounded-lg"
                  >
                    <h4 className="font-medium text-gray-900 dark:text-white capitalize mb-3">
                      {mode} Search
                    </h4>
                    <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                      <div>
                        <p className="text-sm text-gray-600 dark:text-gray-400">Precision</p>
                        <p className="text-xl font-semibold text-gray-900 dark:text-white">
                          {((metrics.precision || 0) * 100).toFixed(1)}%
                        </p>
                      </div>
                      <div>
                        <p className="text-sm text-gray-600 dark:text-gray-400">Recall</p>
                        <p className="text-xl font-semibold text-gray-900 dark:text-white">
                          {((metrics.recall || 0) * 100).toFixed(1)}%
                        </p>
                      </div>
                      <div>
                        <p className="text-sm text-gray-600 dark:text-gray-400">F1 Score</p>
                        <p className="text-xl font-semibold text-gray-900 dark:text-white">
                          {((metrics.f1_score || 0) * 100).toFixed(1)}%
                        </p>
                      </div>
                      <div>
                        <p className="text-sm text-gray-600 dark:text-gray-400">Avg Latency</p>
                        <p className="text-xl font-semibold text-gray-900 dark:text-white">
                          {(metrics.avg_latency_ms || 0).toFixed(1)}ms
                        </p>
                      </div>
                    </div>
                    {(metrics.ndcg !== undefined || metrics.mrr !== undefined) && (
                      <div className="mt-3 pt-3 border-t border-gray-200 dark:border-gray-700 grid grid-cols-2 gap-4">
                        <div>
                          <p className="text-sm text-gray-600 dark:text-gray-400">NDCG</p>
                          <p className="text-lg font-semibold text-gray-900 dark:text-white">
                            {((metrics.ndcg || 0) * 100).toFixed(1)}%
                          </p>
                        </div>
                        <div>
                          <p className="text-sm text-gray-600 dark:text-gray-400">MRR</p>
                          <p className="text-lg font-semibold text-gray-900 dark:text-white">
                            {((metrics.mrr || 0) * 100).toFixed(1)}%
                          </p>
                        </div>
                      </div>
                    )}
                  </div>
                ))}
              </div>

              {/* Actions */}
              <div className="flex flex-col gap-3">
                <div className="flex gap-3 flex-wrap">
                  <button
                    onClick={() => runEvaluation()}
                    disabled={isRunning}
                    className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    <RefreshCw className="w-4 h-4" />
                    Run Again
                  </button>
                  <button
                    onClick={() => generateReport('html')}
                    disabled={isRunning}
                    className="flex items-center gap-2 px-4 py-2 bg-gray-600 text-white rounded-lg hover:bg-gray-700 active:bg-gray-800 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    <FileText className="w-4 h-4" />
                    Generate HTML Report
                  </button>
                  <button
                    onClick={() => generateReport('csv')}
                    disabled={isRunning}
                    className="flex items-center gap-2 px-4 py-2 bg-gray-600 text-white rounded-lg hover:bg-gray-700 active:bg-gray-800 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    <Download className="w-4 h-4" />
                    Download CSV
                  </button>
                </div>
                
                {/* Info about detailed reports */}
                <div className="p-4 bg-gray-50 dark:bg-gray-900 rounded-lg">
                  <p className="text-sm text-gray-600 dark:text-gray-400">
                    <strong>Reports Location:</strong> Evaluation reports are saved to{' '}
                    <code className="px-1 py-0.5 bg-gray-200 dark:bg-gray-800 rounded text-xs">
                      backend/evaluation_results/reports/
                    </code>
                  </p>
                  <div className="mt-3 space-y-2">
                    <p className="text-sm text-gray-600 dark:text-gray-400">
                      <strong>Available Formats:</strong>
                    </p>
                    <ul className="text-sm text-gray-600 dark:text-gray-400 list-disc list-inside ml-2">
                      <li>HTML - Interactive report with visualizations</li>
                      <li>CSV - Raw data for further analysis</li>
                      <li>JSON - Machine-readable format with full details</li>
                    </ul>
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export default Evaluations;