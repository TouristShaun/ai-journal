import { useState, useEffect } from 'react';
import { Sparkles, Brain, Lightbulb, Zap } from 'lucide-react';

function VectorSearch({ onSearch, searchParams }) {
  const [query, setQuery] = useState(searchParams.query || '');
  const [semanticMode, setSemanticMode] = useState('similar');

  useEffect(() => {
    const debounceTimer = setTimeout(() => {
      // Only search if there's a query
      if (query.trim()) {
        onSearch({
          query,
          search_type: 'vector',
          semantic_mode: semanticMode,
        });
      } else {
        // Clear results when query is empty
        onSearch({
          query: '',
          search_type: 'vector',
          semantic_mode: semanticMode,
        });
      }
    }, 500);

    return () => clearTimeout(debounceTimer);
  }, [query, semanticMode]);

  const semanticModes = [
    { id: 'similar', name: 'Find Similar', icon: Brain, description: 'Find entries with similar meaning' },
    { id: 'explore', name: 'Explore Concepts', icon: Lightbulb, description: 'Discover related ideas' },
    { id: 'contrast', name: 'Find Contrasts', icon: Zap, description: 'Find opposing viewpoints' },
  ];

  return (
    <div className="space-y-6">
      {/* Vector Search Input */}
      <div>
        <div className="relative">
          <Sparkles className="absolute left-3 top-1/2 transform -translate-y-1/2 text-purple-500 w-5 h-5" />
          <input
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Describe what you're looking for..."
            className="w-full pl-10 pr-4 py-3 border-2 border-purple-300 dark:border-purple-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-white focus:ring-2 focus:ring-purple-500 focus:border-transparent"
          />
        </div>
        <p className="mt-2 text-sm text-gray-600 dark:text-gray-400">
          Use natural language to find semantically similar entries
        </p>
      </div>

      {/* Semantic Modes */}
      <div>
        <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
          Semantic Mode
        </h3>
        <div className="space-y-2">
          {semanticModes.map(mode => {
            const Icon = mode.icon;
            return (
              <button
                key={mode.id}
                onClick={() => setSemanticMode(mode.id)}
                className={`w-full text-left p-3 rounded-lg transition-all ${
                  semanticMode === mode.id
                    ? 'bg-purple-100 dark:bg-purple-900/30 text-purple-800 dark:text-purple-200 ring-2 ring-purple-500'
                    : 'bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700'
                }`}
              >
                <div className="flex items-start gap-3">
                  <Icon className="w-5 h-5 mt-0.5" />
                  <div className="flex-1">
                    <div className="font-medium">{mode.name}</div>
                    <div className="text-sm opacity-80">{mode.description}</div>
                  </div>
                </div>
              </button>
            );
          })}
        </div>
      </div>

      {/* Example Queries */}
      <div>
        <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
          Try these examples:
        </h3>
        <div className="space-y-2">
          {[
            'Moments of personal growth',
            'Challenges I overcame',
            'Happy memories with friends',
            'Professional achievements',
            'Travel experiences',
          ].map((example) => (
            <button
              key={example}
              onClick={() => setQuery(example)}
              className="w-full text-left px-3 py-2 text-sm bg-purple-50 dark:bg-purple-900/20 text-purple-700 dark:text-purple-300 rounded-lg hover:bg-purple-100 dark:hover:bg-purple-900/30 transition-colors"
            >
              {example}
            </button>
          ))}
        </div>
      </div>

      {/* Vector Space Visualization (placeholder) */}
      <div className="p-4 bg-gradient-to-br from-purple-100 to-pink-100 dark:from-purple-900/20 dark:to-pink-900/20 rounded-lg">
        <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
          Semantic Space
        </h3>
        <div className="h-32 flex items-center justify-center text-gray-500 dark:text-gray-400">
          <p className="text-sm text-center">
            Your entries are embedded in a 768-dimensional space,
            <br />
            enabling powerful semantic search
          </p>
        </div>
      </div>
    </div>
  );
}

export default VectorSearch;