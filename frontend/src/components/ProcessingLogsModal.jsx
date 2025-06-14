import React, { useState, useEffect } from 'react';
import { X, AlertCircle, Info, AlertTriangle, Bug, Lightbulb, Copy, CheckCircle } from 'lucide-react';
import { journalAPI } from '../api/client';

const ProcessingLogsModal = ({ entryId, isOpen, onClose }) => {
  const [logs, setLogs] = useState([]);
  const [failureAnalysis, setFailureAnalysis] = useState(null);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState('logs');
  const [copiedLog, setCopiedLog] = useState(null);

  useEffect(() => {
    if (isOpen && entryId) {
      fetchData();
    }
  }, [isOpen, entryId]);

  const fetchData = async () => {
    setLoading(true);
    try {
      // Fetch logs
      const logs = await journalAPI.getProcessingLogs(entryId);
      setLogs(logs || []);

      // Fetch failure analysis if available
      try {
        const analysis = await journalAPI.analyzeFailure(entryId);
        setFailureAnalysis(analysis);
      } catch (err) {
        // Entry might not have failed, ignore
        console.log('No failure analysis available');
      }
    } catch (error) {
      console.error('Failed to fetch logs:', error);
    } finally {
      setLoading(false);
    }
  };

  const getLogIcon = (level) => {
    switch (level) {
      case 'error':
        return <AlertCircle className="w-4 h-4 text-red-500" />;
      case 'warn':
        return <AlertTriangle className="w-4 h-4 text-yellow-500" />;
      case 'info':
        return <Info className="w-4 h-4 text-blue-500" />;
      case 'debug':
        return <Bug className="w-4 h-4 text-gray-500" />;
      default:
        return null;
    }
  };

  const getLogLevelColor = (level) => {
    switch (level) {
      case 'error':
        return 'text-red-700 bg-red-50';
      case 'warn':
        return 'text-yellow-700 bg-yellow-50';
      case 'info':
        return 'text-blue-700 bg-blue-50';
      case 'debug':
        return 'text-gray-700 bg-gray-50';
      default:
        return 'text-gray-700 bg-gray-50';
    }
  };

  const formatTimestamp = (timestamp) => {
    const date = new Date(timestamp);
    return date.toLocaleTimeString('en-US', {
      hour12: false,
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      fractionalSecondDigits: 3
    });
  };

  const copyLogToClipboard = async (log) => {
    const logText = `[${formatTimestamp(log.created_at)}] ${log.stage} - ${log.level}: ${log.message}`;
    await navigator.clipboard.writeText(logText);
    setCopiedLog(log.id);
    setTimeout(() => setCopiedLog(null), 2000);
  };

  const groupLogsByStage = () => {
    const grouped = {};
    logs.forEach(log => {
      if (!grouped[log.stage]) {
        grouped[log.stage] = [];
      }
      grouped[log.stage].push(log);
    });
    return grouped;
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-lg shadow-xl max-w-4xl w-full max-h-[90vh] flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b">
          <h2 className="text-xl font-semibold text-gray-900">
            Processing Details
          </h2>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600 transition-colors"
          >
            <X className="w-6 h-6" />
          </button>
        </div>

        {/* Tabs */}
        <div className="flex border-b">
          <button
            onClick={() => setActiveTab('logs')}
            className={`px-6 py-3 font-medium transition-colors ${
              activeTab === 'logs'
                ? 'text-blue-600 border-b-2 border-blue-600'
                : 'text-gray-600 hover:text-gray-900'
            }`}
          >
            Processing Logs
          </button>
          {failureAnalysis && (
            <button
              onClick={() => setActiveTab('analysis')}
              className={`px-6 py-3 font-medium transition-colors ${
                activeTab === 'analysis'
                  ? 'text-blue-600 border-b-2 border-blue-600'
                  : 'text-gray-600 hover:text-gray-900'
              }`}
            >
              Failure Analysis
            </button>
          )}
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-6">
          {loading ? (
            <div className="flex items-center justify-center h-64">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
            </div>
          ) : (
            <>
              {activeTab === 'logs' && (
                <div className="space-y-6">
                  {Object.entries(groupLogsByStage()).map(([stage, stageLogs]) => (
                    <div key={stage} className="space-y-2">
                      <h3 className="text-sm font-semibold text-gray-700 uppercase tracking-wider">
                        {stage.replace(/_/g, ' ')}
                      </h3>
                      <div className="space-y-1">
                        {stageLogs.map((log) => (
                          <div
                            key={log.id}
                            className={`flex items-start gap-3 p-3 rounded-lg ${getLogLevelColor(log.level)} group`}
                          >
                            {getLogIcon(log.level)}
                            <div className="flex-1 min-w-0">
                              <div className="flex items-center gap-2">
                                <span className="text-xs text-gray-500 font-mono">
                                  {formatTimestamp(log.created_at)}
                                </span>
                                <span className={`text-xs font-medium uppercase ${
                                  log.level === 'error' ? 'text-red-600' :
                                  log.level === 'warn' ? 'text-yellow-600' :
                                  log.level === 'info' ? 'text-blue-600' :
                                  'text-gray-600'
                                }`}>
                                  {log.level}
                                </span>
                              </div>
                              <p className="text-sm mt-1 break-words">
                                {log.message}
                              </p>
                              {log.details && Object.keys(log.details).length > 0 && (
                                <pre className="text-xs mt-2 p-2 bg-gray-100 rounded overflow-x-auto">
                                  {JSON.stringify(log.details, null, 2)}
                                </pre>
                              )}
                            </div>
                            <button
                              onClick={() => copyLogToClipboard(log)}
                              className="opacity-0 group-hover:opacity-100 transition-opacity"
                              title="Copy log"
                            >
                              {copiedLog === log.id ? (
                                <CheckCircle className="w-4 h-4 text-green-600" />
                              ) : (
                                <Copy className="w-4 h-4 text-gray-400 hover:text-gray-600" />
                              )}
                            </button>
                          </div>
                        ))}
                      </div>
                    </div>
                  ))}
                </div>
              )}

              {activeTab === 'analysis' && failureAnalysis && (
                <div className="space-y-6">
                  {/* Error Summary */}
                  <div className="bg-red-50 border border-red-200 rounded-lg p-4">
                    <h3 className="flex items-center gap-2 text-lg font-semibold text-red-800 mb-2">
                      <AlertCircle className="w-5 h-5" />
                      Failure Summary
                    </h3>
                    <p className="text-sm text-red-700">
                      Failed at stage: <span className="font-semibold">{failureAnalysis.failed_stage.replace(/_/g, ' ')}</span>
                    </p>
                    {failureAnalysis.error && (
                      <p className="text-sm text-red-700 mt-1">
                        Error: {failureAnalysis.error}
                      </p>
                    )}
                  </div>

                  {/* Likely Causes */}
                  <div>
                    <h3 className="text-lg font-semibold text-gray-900 mb-4">
                      Likely Causes
                    </h3>
                    <div className="space-y-3">
                      {failureAnalysis.likely_causes.map((cause, index) => (
                        <div
                          key={index}
                          className="bg-gray-50 border border-gray-200 rounded-lg p-4"
                        >
                          <div className="flex items-start justify-between mb-2">
                            <h4 className="font-medium text-gray-900">
                              {cause.cause}
                            </h4>
                            <span className="text-sm font-medium text-gray-600 bg-gray-200 px-2 py-1 rounded">
                              {Math.round(cause.probability * 100)}% likely
                            </span>
                          </div>
                          <p className="text-sm text-gray-700 mb-2">
                            {cause.description}
                          </p>
                          <div className="flex items-start gap-2">
                            <Lightbulb className="w-4 h-4 text-yellow-500 flex-shrink-0 mt-0.5" />
                            <p className="text-sm text-gray-600">
                              <span className="font-medium">Solution:</span> {cause.solution}
                            </p>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>

                  {/* Recommendation */}
                  {failureAnalysis.recommendation && (
                    <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
                      <h3 className="flex items-center gap-2 text-lg font-semibold text-blue-800 mb-2">
                        <Lightbulb className="w-5 h-5" />
                        Recommendation
                      </h3>
                      <div className="text-sm text-blue-700 whitespace-pre-line">
                        {failureAnalysis.recommendation}
                      </div>
                    </div>
                  )}
                </div>
              )}
            </>
          )}
        </div>

        {/* Footer */}
        <div className="flex justify-end gap-3 p-6 border-t">
          <button
            onClick={onClose}
            className="px-4 py-2 text-gray-700 bg-gray-100 hover:bg-gray-200 rounded-lg transition-colors"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
};

export default ProcessingLogsModal;