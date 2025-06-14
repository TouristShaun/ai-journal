import { useState, useEffect } from 'react';
import { GitMerge, Wand2, TrendingUp, Filter } from 'lucide-react';

function HybridSearch({ collections, onSearch, searchParams }) {
  const [localParams, setLocalParams] = useState({
    query: searchParams.query || '',
    is_favorite: searchParams.is_favorite,
    collection_ids: searchParams.collection_ids || [],
    hybrid_mode: 'balanced',
  });

  useEffect(() => {
    const debounceTimer = setTimeout(() => {
      onSearch({
        ...localParams,
        search_type: 'hybrid',
      });
    }, 500);

    return () => clearTimeout(debounceTimer);
  }, [localParams]);

  const hybridModes = [
    { 
      id: 'balanced', 
      name: 'Balanced', 
      description: 'Equal weight to semantic and keyword matching',
      color: 'blue'
    },
    { 
      id: 'semantic_boost', 
      name: 'Semantic Boost', 
      description: 'Prioritize meaning over exact matches',
      color: 'purple'
    },
    { 
      id: 'precision', 
      name: 'Precision Mode', 
      description: 'Favor exact matches with semantic enhancement',
      color: 'green'
    },
    { 
      id: 'discovery', 
      name: 'Discovery Mode', 
      description: 'Surface unexpected connections',
      color: 'orange'
    },
  ];

  return (
    <div className="space-y-6">
      {/* Hybrid Search Input */}
      <div>
        <div className="relative">
          <GitMerge className="absolute left-3 top-1/2 transform -translate-y-1/2 text-indigo-500 w-5 h-5" />
          <input
            type="text"
            value={localParams.query}
            onChange={(e) => setLocalParams(prev => ({ ...prev, query: e.target.value }))}
            placeholder="Search with the power of AI..."
            className="w-full pl-10 pr-4 py-3 border-2 border-indigo-300 dark:border-indigo-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-white focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
          />
        </div>
        <p className="mt-2 text-sm text-gray-600 dark:text-gray-400">
          Combines traditional search with semantic understanding
        </p>
      </div>

      {/* Hybrid Modes */}
      <div>
        <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3 flex items-center gap-2">
          <Wand2 className="w-4 h-4" />
          Search Mode
        </h3>
        <div className="grid grid-cols-2 gap-2">
          {hybridModes.map(mode => (
            <button
              key={mode.id}
              onClick={() => setLocalParams(prev => ({ ...prev, hybrid_mode: mode.id }))}
              className={`p-3 rounded-lg transition-all text-left ${
                localParams.hybrid_mode === mode.id
                  ? mode.id === 'precision' 
                    ? 'bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-200 ring-2 ring-yellow-500'
                    : mode.id === 'discovery'
                    ? 'bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-200 ring-2 ring-green-500'
                    : `bg-${mode.color}-100 dark:bg-${mode.color}-900/30 text-${mode.color}-800 dark:text-${mode.color}-200 ring-2 ring-${mode.color}-500`
                  : 'bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700'
              }`}
            >
              <div className="font-medium text-sm">{mode.name}</div>
              <div className="text-xs opacity-80 mt-1">{mode.description}</div>
            </button>
          ))}
        </div>
      </div>

      {/* Smart Filters */}
      <div>
        <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3 flex items-center gap-2">
          <Filter className="w-4 h-4" />
          Smart Filters
        </h3>
        <div className="space-y-2">
          <label className="flex items-center gap-3 p-3 bg-gray-100 dark:bg-gray-800 rounded-lg cursor-pointer hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors">
            <input
              type="checkbox"
              checked={localParams.is_favorite || false}
              onChange={(e) => setLocalParams(prev => ({ ...prev, is_favorite: e.target.checked ? true : null }))}
              className="w-4 h-4 text-indigo-600 rounded focus:ring-indigo-500"
            />
            <span className="text-sm text-gray-700 dark:text-gray-300">Prioritize favorites</span>
          </label>
          
          {collections.length > 0 && (
            <div className="p-3 bg-gray-100 dark:bg-gray-800 rounded-lg">
              <p className="text-sm font-medium mb-2">Filter by collections:</p>
              <div className="space-y-1">
                {collections.map(collection => (
                  <label key={collection.id} className="flex items-center gap-2 text-sm cursor-pointer">
                    <input
                      type="checkbox"
                      checked={localParams.collection_ids.includes(collection.id)}
                      onChange={() => {
                        setLocalParams(prev => ({
                          ...prev,
                          collection_ids: prev.collection_ids.includes(collection.id)
                            ? prev.collection_ids.filter(id => id !== collection.id)
                            : [...prev.collection_ids, collection.id],
                        }));
                      }}
                      className="w-3 h-3 text-indigo-600 rounded focus:ring-indigo-500"
                    />
                    <span>{collection.name}</span>
                  </label>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>

      {/* AI Insights */}
      <div className="p-4 bg-gradient-to-br from-indigo-100 to-purple-100 dark:from-indigo-900/20 dark:to-purple-900/20 rounded-lg">
        <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2 flex items-center gap-2">
          <TrendingUp className="w-4 h-4" />
          AI Insights
        </h3>
        <div className="space-y-2 text-sm text-gray-600 dark:text-gray-400">
          <p>• Automatically groups related entries</p>
          <p>• Identifies emerging themes in your journal</p>
          <p>• Suggests connections you might have missed</p>
          <p>• Learns from your search patterns</p>
        </div>
      </div>
    </div>
  );
}

export default HybridSearch;