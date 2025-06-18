import { Star, Tag, Calendar, Brain, Loader2, AlertCircle, CheckCircle2, Package, Cpu, Link, Database } from 'lucide-react';
import { format } from 'date-fns';

function EntryCard({ entry, isSelected, onClick }) {
  const getSentimentColor = (sentiment) => {
    switch (sentiment) {
      case 'positive': return 'text-green-600 dark:text-green-400';
      case 'negative': return 'text-red-600 dark:text-red-400';
      case 'mixed': return 'text-yellow-600 dark:text-yellow-400';
      default: return 'text-gray-600 dark:text-gray-400';
    }
  };

  const similarityScore = entry.processed_data?.metadata?.similarity_score;
  const processingStage = entry.processing_stage || 'created';
  const isProcessing = processingStage !== 'completed' && processingStage !== 'failed';
  const hasFailed = processingStage === 'failed';
  const processingError = entry.processing_error;
  
  const getStageIcon = (stage) => {
    const stages = {
      created: { icon: Package },
      analyzing: { icon: Cpu },
      fetching_urls: { icon: Link },
      generating_embeddings: { icon: Database },
      completed: { icon: CheckCircle2 },
      failed: { icon: AlertCircle }
    };
    
    const currentStage = stages[stage] || stages.created;
    const Icon = currentStage.icon;
    
    // Active stage pulsates
    if (stage === processingStage && isProcessing) {
      const activeClasses = {
        created: 'w-4 h-4 text-gray-400 animate-pulse',
        analyzing: 'w-4 h-4 text-yellow-500 animate-pulse',
        fetching_urls: 'w-4 h-4 text-yellow-500 animate-pulse',
        generating_embeddings: 'w-4 h-4 text-yellow-500 animate-pulse',
        completed: 'w-4 h-4 text-green-500 animate-pulse',
        failed: 'w-4 h-4 text-red-500 animate-pulse'
      };
      return <Icon className={activeClasses[stage] || 'w-4 h-4 text-gray-400 animate-pulse'} />;
    }
    
    // Completed stages are green
    const stageOrder = ['created', 'analyzing', 'fetching_urls', 'generating_embeddings', 'completed'];
    const currentIndex = stageOrder.indexOf(processingStage);
    const stageIndex = stageOrder.indexOf(stage);
    
    if (stageIndex < currentIndex && !hasFailed) {
      return <Icon className="w-4 h-4 text-green-500" />;
    }
    
    // Failed stage is red
    if (hasFailed && stage === processingStage) {
      return <Icon className="w-4 h-4 text-red-500" />;
    }
    
    // Future stages are gray
    const futureClasses = {
      created: 'w-4 h-4 text-gray-400',
      analyzing: 'w-4 h-4 text-yellow-500',
      fetching_urls: 'w-4 h-4 text-yellow-500',
      generating_embeddings: 'w-4 h-4 text-yellow-500',
      completed: 'w-4 h-4 text-green-500',
      failed: 'w-4 h-4 text-red-500'
    };
    return <Icon className={futureClasses[stage] || 'w-4 h-4 text-gray-400'} />;
  };

  return (
    <div
      onClick={onClick}
      className={`p-4 rounded-lg cursor-pointer transition-all ${
        isSelected
          ? 'bg-blue-50 dark:bg-blue-900/30 ring-2 ring-blue-500'
          : 'bg-white dark:bg-gray-800 hover:bg-gray-50 dark:hover:bg-gray-700 border border-gray-200 dark:border-gray-700'
      }`}
    >
      {/* Header */}
      <div className="flex items-start justify-between mb-2">
        <div className="flex-1">
          <div className="flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
            <Calendar className="w-3 h-3" />
            <time>{format(new Date(entry.created_at), 'MMM d, yyyy • h:mm a')}</time>
            {similarityScore && (
              <>
                <Brain className="w-3 h-3 ml-2" />
                <span>{Math.round(similarityScore * 100)}% match</span>
              </>
            )}
          </div>
          {/* Processing Stage Icons */}
          {processingStage !== 'completed' && (
            <div className="flex items-center gap-1 mt-1">
              {getStageIcon('created')}
              <span className="text-gray-400">→</span>
              {getStageIcon('analyzing')}
              <span className="text-gray-400">→</span>
              {getStageIcon('fetching_urls')}
              <span className="text-gray-400">→</span>
              {getStageIcon('generating_embeddings')}
              <span className="text-gray-400">→</span>
              {getStageIcon('completed')}
            </div>
          )}
        </div>
        {entry.is_favorite && (
          <Star className="w-4 h-4 text-yellow-500 fill-current" />
        )}
      </div>

      {/* Summary */}
      <h3 className="font-medium text-gray-700 dark:text-gray-300 mb-2 line-clamp-2">
        {entry.processed_data.summary || entry.content.substring(0, 100) + '...'}
      </h3>

      {/* Content Preview */}
      <p className="text-sm text-gray-500 dark:text-gray-400 line-clamp-3 mb-3">
        {entry.content}
      </p>

      {/* Metadata */}
      <div className="flex flex-wrap gap-2">
        {/* Show error message if failed */}
        {hasFailed && processingError ? (
          <span className="text-xs text-red-600 dark:text-red-400">
            Error: {processingError}
          </span>
        ) : (
          <>
            {/* Sentiment */}
            {!isProcessing && (
              <span className={`text-xs font-medium ${getSentimentColor(entry.processed_data.sentiment)}`}>
                {entry.processed_data.sentiment}
              </span>
            )}

            {/* Topics */}
            {!isProcessing && entry.processed_data.topics.slice(0, 3).map((topic) => (
              <span
                key={topic}
                className="inline-flex items-center gap-1 px-2 py-1 bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded text-xs"
              >
                <Tag className="w-3 h-3" />
                {topic}
              </span>
            ))}

            {/* URL indicator */}
            {!isProcessing && entry.processed_data.extracted_urls?.length > 0 && (
              <span className="text-xs text-blue-600 dark:text-blue-400">
                {entry.processed_data.extracted_urls.length} link{entry.processed_data.extracted_urls.length > 1 ? 's' : ''}
              </span>
            )}
          </>
        )}
      </div>
    </div>
  );
}

export default EntryCard;