import { useQuery } from '@tanstack/react-query';
import { journalAPI } from '../api/client';
import { Tag, User, Clock } from 'lucide-react';

function SearchSuggestions({ onSelectSuggestion }) {
  const { data: suggestions, isLoading } = useQuery({
    queryKey: ['searchSuggestions'],
    queryFn: journalAPI.getSearchSuggestions,
    staleTime: 1000 * 60 * 5, // 5 minutes
  });

  if (isLoading || !suggestions) {
    return null;
  }

  const hasContent = (suggestions.topics?.length > 0) || 
                     (suggestions.entities?.length > 0) || 
                     (suggestions.recent?.length > 0);

  if (!hasContent) {
    return null;
  }

  return (
    <div className="space-y-4 text-sm">
      {/* Popular Topics */}
      {suggestions.topics?.length > 0 && (
        <div>
          <h4 className="flex items-center gap-2 text-xs font-medium text-gray-600 dark:text-gray-400 mb-2">
            <Tag className="w-3 h-3" />
            Popular Topics
          </h4>
          <div className="flex flex-wrap gap-2">
            {suggestions.topics.map((topic, idx) => (
              <button
                key={idx}
                onClick={() => onSelectSuggestion(topic.text)}
                className="px-2 py-1 bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-md hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors text-xs"
              >
                {topic.text}
                <span className="ml-1 text-gray-500 dark:text-gray-400">({topic.count})</span>
              </button>
            ))}
          </div>
        </div>
      )}

      {/* Popular Entities */}
      {suggestions.entities?.length > 0 && (
        <div>
          <h4 className="flex items-center gap-2 text-xs font-medium text-gray-600 dark:text-gray-400 mb-2">
            <User className="w-3 h-3" />
            People & Places
          </h4>
          <div className="flex flex-wrap gap-2">
            {suggestions.entities.map((entity, idx) => (
              <button
                key={idx}
                onClick={() => onSelectSuggestion(entity.text)}
                className="px-2 py-1 bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-300 rounded-md hover:bg-blue-100 dark:hover:bg-blue-900/30 transition-colors text-xs"
              >
                {entity.text}
                <span className="ml-1 text-blue-500 dark:text-blue-400">({entity.count})</span>
              </button>
            ))}
          </div>
        </div>
      )}

      {/* Recent Phrases */}
      {suggestions.recent?.length > 0 && (
        <div>
          <h4 className="flex items-center gap-2 text-xs font-medium text-gray-600 dark:text-gray-400 mb-2">
            <Clock className="w-3 h-3" />
            Recent Entries
          </h4>
          <div className="space-y-1">
            {suggestions.recent.map((phrase, idx) => (
              <button
                key={idx}
                onClick={() => onSelectSuggestion(phrase)}
                className="w-full text-left px-2 py-1 text-xs text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 rounded transition-colors truncate"
              >
                "{phrase}"
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

export default SearchSuggestions;