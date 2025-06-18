import React, { useMemo, useState, useEffect } from 'react';
import { CheckCircle, AlertCircle, Loader, Link, Package, Cpu, Database, RefreshCw, ChevronUp } from 'lucide-react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { journalAPI } from '../api/client';

const ProcessingTracker = ({ entry, onViewLogs }) => {
  const [currentTime, setCurrentTime] = useState(Date.now());
  const [showCompleted, setShowCompleted] = useState(false);
  const [hideAfterComplete, setHideAfterComplete] = useState(false);
  const [manuallyCollapsed, setManuallyCollapsed] = useState(false);
  const queryClient = useQueryClient();
  
  // Debug logging
  useEffect(() => {
    console.log(`ProcessingTracker: Entry ${entry.id} stage: ${entry.processing_stage}`);
  }, [entry.id, entry.processing_stage]);
  
  // Handle completion visibility
  useEffect(() => {
    if (entry.processing_stage === 'completed' && !entry.processing_error) {
      setShowCompleted(true);
      
      // Hide after 2 seconds
      const timer = setTimeout(() => {
        setHideAfterComplete(true);
      }, 2000);
      
      return () => clearTimeout(timer);
    }
  }, [entry.processing_stage, entry.processing_error]);
  
  const retryMutation = useMutation({
    mutationFn: () => journalAPI.retryProcessing(entry.id),
    onSuccess: () => {
      queryClient.invalidateQueries(['entries']);
    },
    onError: (error) => {
      console.error('Failed to retry processing:', error);
      alert(`Failed to retry processing: ${error.message || 'Unknown error occurred'}`);
    },
  });
  
  // Update timer every second while processing
  useEffect(() => {
    if (entry.processing_stage && 
        entry.processing_stage !== 'completed' && 
        entry.processing_stage !== 'failed') {
      const interval = setInterval(() => {
        setCurrentTime(Date.now());
      }, 1000);
      
      return () => clearInterval(interval);
    }
  }, [entry.processing_stage]);
  
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
      : new Date(currentTime); // Use currentTime state for live updates
    
    const seconds = Math.floor((end - start) / 1000);
    
    if (seconds < 60) return `${seconds}s`;
    const minutes = Math.floor(seconds / 60);
    const remainingSeconds = seconds % 60;
    return `${minutes}m ${remainingSeconds}s`;
  };

  // Show collapsed view after completion or manual collapse
  if (manuallyCollapsed || hideAfterComplete || (entry.processing_stage === 'completed' && !entry.processing_error && !showCompleted)) {
    const isCompleted = entry.processing_stage === 'completed' && !entry.processing_error;
    
    return (
      <div className={`rounded-lg shadow-sm border p-4 mb-4 transition-all duration-300 ${
        isCompleted 
          ? 'bg-green-50 border-green-200' 
          : 'bg-blue-50 border-blue-200'
      }`}>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            {isCompleted ? (
              <CheckCircle className="w-5 h-5 text-green-600" />
            ) : (
              <Loader className="w-5 h-5 text-blue-600 animate-spin" />
            )}
            <div>
              <p className={`text-sm font-medium ${
                isCompleted ? 'text-green-800' : 'text-blue-800'
              }`}>
                {isCompleted ? 'Processing Complete' : `Processing: ${entry.processing_stage?.replace(/_/g, ' ')}`}
              </p>
              <p className={`text-xs ${
                isCompleted ? 'text-green-600' : 'text-blue-600'
              }`}>
                {getProcessingTime() && `${isCompleted ? 'Completed' : 'Running'} for ${getProcessingTime()}`}
              </p>
            </div>
          </div>
          <button
            onClick={() => {
              setHideAfterComplete(false);
              setShowCompleted(true);
              setManuallyCollapsed(false);
            }}
            className={`text-sm font-medium flex items-center gap-1 ${
              isCompleted 
                ? 'text-green-700 hover:text-green-800' 
                : 'text-blue-700 hover:text-blue-800'
            }`}
          >
            View Details
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
            </svg>
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6 mb-4">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-semibold text-gray-900">Processing Status</h3>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setManuallyCollapsed(true)}
            className="text-sm text-gray-600 hover:text-gray-800 flex items-center gap-1"
            title="Collapse"
          >
            <ChevronUp className="w-4 h-4" />
            Collapse
          </button>
          {getProcessingTime() && (
            <span className="text-sm text-gray-500">
              Time: {getProcessingTime()}
            </span>
          )}
          {entry.processing_stage === 'failed' && (
            <div className="flex items-center gap-2">
              <button
                onClick={onViewLogs}
                className="text-sm text-blue-600 hover:text-blue-700 flex items-center gap-1"
              >
                <AlertCircle className="w-4 h-4" />
                View Logs
              </button>
              <button
                onClick={() => retryMutation.mutate()}
                disabled={retryMutation.isPending}
                className="text-sm text-green-600 hover:text-green-700 flex items-center gap-1 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <RefreshCw className={`w-4 h-4 ${retryMutation.isPending ? 'animate-spin' : ''}`} />
                {retryMutation.isPending ? 'Retrying...' : 'Retry Processing'}
              </button>
            </div>
          )}
        </div>
      </div>

      {/* Stage Indicators */}
      <div className="relative">
        <div className="flex justify-between mb-20">
          {stages.map((stage, index) => {
            const status = getStageStatus(index);
            const isActive = status === 'active';
            
            return (
              <div
                key={stage.id}
                className={isActive ? 'relative flex flex-col items-center flex-1 animate-bounce' : 'relative flex flex-col items-center flex-1'}
              >
                <div className={
                  status === 'completed' ? 'relative z-10 w-12 h-12 rounded-full flex items-center justify-center bg-green-100 transition-all duration-300' :
                  status === 'active' ? 'relative z-10 w-12 h-12 rounded-full flex items-center justify-center bg-blue-100 transition-all duration-300' :
                  status === 'failed' ? 'relative z-10 w-12 h-12 rounded-full flex items-center justify-center bg-red-100 transition-all duration-300' :
                  'relative z-10 w-12 h-12 rounded-full flex items-center justify-center bg-gray-100 transition-all duration-300'
                }>
                  {getStageIcon(stage, status)}
                </div>
                
                <div className="mt-2 text-center w-full">
                  <p className={
                    status === 'active' ? 'text-sm font-medium text-blue-700' : 
                    status === 'completed' ? 'text-sm font-medium text-green-700' :
                    status === 'failed' ? 'text-sm font-medium text-red-700' :
                    'text-sm font-medium text-gray-500'
                  }>
                    {stage.name === 'Generating Embeddings' ? (
                      <>Generating<br />Embeddings</>
                    ) : (
                      stage.name
                    )}
                  </p>
                  <p className="text-xs text-gray-500 mt-1 px-2">
                    {stage.description}
                  </p>
                </div>
              </div>
            );
          })}
        </div>
        
        {/* Progress Bar - positioned to align with icon centers */}
        <div className="absolute top-6 left-[10%] right-[10%] h-2">
          <div className="absolute inset-0 bg-gray-200 rounded-full" />
          <div 
            className={`absolute inset-y-0 left-0 rounded-full transition-all duration-500 ${
              entry.processing_stage === 'completed' ? 'bg-green-500' : 
              entry.processing_stage === 'failed' ? 'bg-red-500' : 
              'bg-blue-500'
            }`}
            style={{ width: `${getProgressPercentage()}%` }}
          />
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

      {/* Success Message */}
      {entry.processing_stage === 'completed' && !entry.processing_error && (
        <div className="mt-4 p-4 bg-green-50 rounded-lg border border-green-200">
          <div className="flex items-start gap-3">
            <CheckCircle className="w-5 h-5 text-green-600 flex-shrink-0 mt-0.5" />
            <div className="flex-1">
              <h4 className="text-sm font-semibold text-green-800 mb-1">
                Processing Complete!
              </h4>
              <p className="text-sm text-green-700">
                Your entry has been analyzed and indexed successfully.
              </p>
              {entry.processed_data && (
                <div className="mt-2 text-xs text-green-600">
                  Found {entry.processed_data.topics?.length || 0} topics, 
                  {' '}{entry.processed_data.entities?.length || 0} entities
                  {entry.processed_data.extracted_urls?.length > 0 && 
                    `, and ${entry.processed_data.extracted_urls.length} URLs`}
                </div>
              )}
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