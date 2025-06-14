import React, { useMemo } from 'react';
import { CheckCircle, AlertCircle, Loader, Link, Package, Cpu, Database } from 'lucide-react';

const ProcessingTracker = ({ entry, onViewLogs }) => {
  const stages = useMemo(() => [
    {
      id: 'created',
      name: 'Created',
      icon: Package,
      description: 'Entry saved to database'
    },
    {
      id: 'analyzing',
      name: 'Analyzing',
      icon: Cpu,
      description: 'AI analyzing content'
    },
    {
      id: 'fetching_urls',
      name: 'Fetching URLs',
      icon: Link,
      description: 'Retrieving linked content'
    },
    {
      id: 'generating_embeddings',
      name: 'Generating Embeddings',
      icon: Database,
      description: 'Creating semantic search data'
    },
    {
      id: 'completed',
      name: 'Complete',
      icon: CheckCircle,
      description: 'Processing finished'
    }
  ], []);

  const currentStageIndex = useMemo(() => {
    if (!entry.processing_stage) return 0;
    
    const stageOrder = ['created', 'analyzing', 'fetching_urls', 'generating_embeddings', 'completed', 'failed'];
    return stageOrder.indexOf(entry.processing_stage);
  }, [entry.processing_stage]);

  const getStageStatus = (stageIndex) => {
    if (entry.processing_stage === 'failed') {
      return stageIndex <= currentStageIndex ? 'failed' : 'pending';
    }
    
    if (stageIndex < currentStageIndex) return 'completed';
    if (stageIndex === currentStageIndex) return 'active';
    return 'pending';
  };

  const getStageIcon = (stage, status) => {
    const Icon = stage.icon;
    
    if (status === 'completed') {
      return <CheckCircle className="w-6 h-6 text-green-500" />;
    }
    
    if (status === 'failed') {
      return <AlertCircle className="w-6 h-6 text-red-500" />;
    }
    
    if (status === 'active') {
      return (
        <div className="relative">
          <Icon className="w-6 h-6 text-blue-500 animate-pulse" />
          <Loader className="absolute inset-0 w-6 h-6 text-blue-500 animate-spin" />
        </div>
      );
    }
    
    return <Icon className="w-6 h-6 text-gray-400" />;
  };

  const getProgressPercentage = () => {
    if (entry.processing_stage === 'failed') {
      return (currentStageIndex / 4) * 100;
    }
    return (currentStageIndex / 4) * 100;
  };

  const getProcessingTime = () => {
    if (!entry.processing_started_at) return null;
    
    const start = new Date(entry.processing_started_at);
    const end = entry.processing_completed_at 
      ? new Date(entry.processing_completed_at)
      : new Date();
    
    const seconds = Math.floor((end - start) / 1000);
    
    if (seconds < 60) return `${seconds}s`;
    const minutes = Math.floor(seconds / 60);
    const remainingSeconds = seconds % 60;
    return `${minutes}m ${remainingSeconds}s`;
  };

  if (entry.processing_stage === 'completed' && !entry.processing_error) {
    return null; // Don't show tracker for successfully completed entries
  }

  return (
    <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6 mb-4">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-semibold text-gray-900">Processing Status</h3>
        <div className="flex items-center gap-2">
          {getProcessingTime() && (
            <span className="text-sm text-gray-500">
              Time: {getProcessingTime()}
            </span>
          )}
          {entry.processing_stage === 'failed' && (
            <button
              onClick={onViewLogs}
              className="text-sm text-blue-600 hover:text-blue-700 flex items-center gap-1"
            >
              <AlertCircle className="w-4 h-4" />
              View Logs
            </button>
          )}
        </div>
      </div>

      {/* Progress Bar */}
      <div className="relative mb-8">
        <div className="absolute top-1/2 left-0 right-0 h-2 bg-gray-200 rounded-full -translate-y-1/2" />
        <div 
          className="absolute top-1/2 left-0 h-2 bg-blue-500 rounded-full -translate-y-1/2 transition-all duration-500"
          style={{ width: `${getProgressPercentage()}%` }}
        />
        
        {/* Stage Indicators */}
        <div className="relative flex justify-between">
          {stages.map((stage, index) => {
            const status = getStageStatus(index);
            const isActive = status === 'active';
            
            return (
              <div
                key={stage.id}
                className={`relative flex flex-col items-center ${
                  isActive ? 'animate-bounce' : ''
                }`}
              >
                <div className={`
                  relative z-10 w-12 h-12 rounded-full flex items-center justify-center
                  ${status === 'completed' ? 'bg-green-100' : ''}
                  ${status === 'active' ? 'bg-blue-100' : ''}
                  ${status === 'failed' ? 'bg-red-100' : ''}
                  ${status === 'pending' ? 'bg-gray-100' : ''}
                  transition-all duration-300
                `}>
                  {getStageIcon(stage, status)}
                </div>
                
                <div className="mt-2 text-center">
                  <p className={`text-sm font-medium ${
                    status === 'active' ? 'text-blue-700' : 
                    status === 'completed' ? 'text-green-700' :
                    status === 'failed' ? 'text-red-700' :
                    'text-gray-500'
                  }`}>
                    {stage.name}
                  </p>
                  <p className="text-xs text-gray-500 mt-1 max-w-[100px]">
                    {stage.description}
                  </p>
                </div>
              </div>
            );
          })}
        </div>
      </div>

      {/* Error Message */}
      {entry.processing_error && (
        <div className="mt-4 p-4 bg-red-50 rounded-lg border border-red-200">
          <div className="flex items-start gap-3">
            <AlertCircle className="w-5 h-5 text-red-600 flex-shrink-0 mt-0.5" />
            <div className="flex-1">
              <h4 className="text-sm font-semibold text-red-800 mb-1">
                Processing Failed
              </h4>
              <p className="text-sm text-red-700">
                {entry.processing_error}
              </p>
              <button
                onClick={onViewLogs}
                className="mt-2 text-sm text-red-600 hover:text-red-700 underline"
              >
                View detailed logs and troubleshooting guide
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Current Stage Message */}
      {entry.processing_stage !== 'failed' && entry.processing_stage !== 'completed' && (
        <div className="mt-4 text-center">
          <p className="text-sm text-gray-600">
            {entry.processing_stage === 'analyzing' && 'AI is analyzing your journal entry...'}
            {entry.processing_stage === 'fetching_urls' && 'Fetching content from linked URLs...'}
            {entry.processing_stage === 'generating_embeddings' && 'Creating semantic search index...'}
            {entry.processing_stage === 'created' && 'Preparing to process your entry...'}
          </p>
        </div>
      )}
    </div>
  );
};

export default ProcessingTracker;