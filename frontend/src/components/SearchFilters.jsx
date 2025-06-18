import { useState, useEffect } from 'react';
import { Search, Star, Calendar, Tag } from 'lucide-react';
import SearchSuggestions from './SearchSuggestions';

function SearchFilters({ collections, onSearch, searchParams }) {
  const [localParams, setLocalParams] = useState({
    query: searchParams.query || '',
    is_favorite: searchParams.is_favorite,
    collection_ids: searchParams.collection_ids || [],
    start_date: searchParams.start_date || null,
    end_date: searchParams.end_date || null,
  });

  useEffect(() => {
    const debounceTimer = setTimeout(() => {
      onSearch(localParams);
    }, 300);

    return () => clearTimeout(debounceTimer);
  }, [localParams]);

  const toggleFavorite = () => {
    setLocalParams(prev => ({
      ...prev,
      is_favorite: prev.is_favorite === true ? null : true,
    }));
  };

  const toggleCollection = (collectionId) => {
    setLocalParams(prev => ({
      ...prev,
      collection_ids: prev.collection_ids.includes(collectionId)
        ? prev.collection_ids.filter(id => id !== collectionId)
        : [...prev.collection_ids, collectionId],
    }));
  };

  return (
    <div className="space-y-4">
      {/* Search Input */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 w-5 h-5" />
        <input
          type="text"
          value={localParams.query}
          onChange={(e) => setLocalParams(prev => ({ ...prev, query: e.target.value }))}
          placeholder="Search entries..."
          className="w-full pl-10 pr-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
        />
      </div>

      {/* Quick Filters */}
      <div className="space-y-2">
        <button
          onClick={toggleFavorite}
          className={`w-full flex items-center gap-3 px-4 py-2 rounded-lg transition-colors ${
            localParams.is_favorite
              ? 'bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-200'
              : 'bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700'
          }`}
        >
          <Star className="w-4 h-4" />
          <span>Favorites</span>
        </button>
      </div>

      {/* Search Suggestions */}
      {!localParams.query && (
        <SearchSuggestions 
          onSelectSuggestion={(text) => setLocalParams(prev => ({ ...prev, query: text }))}
        />
      )}

      {/* Collections */}
      <div>
        <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
          Collections
        </h3>
        <div className="space-y-1">
          {collections.map(collection => (
            <button
              key={collection.id}
              onClick={() => toggleCollection(collection.id)}
              className={`w-full text-left flex items-center gap-3 px-3 py-2 rounded-lg transition-colors ${
                localParams.collection_ids.includes(collection.id)
                  ? 'bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-200'
                  : 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800'
              }`}
            >
              <Tag className="w-4 h-4" />
              <span className="truncate">{collection.name}</span>
            </button>
          ))}
        </div>
      </div>

      {/* Date Range */}
      <div>
        <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2 flex items-center gap-2">
          <Calendar className="w-4 h-4" />
          Date Range
        </h3>
        <div className="space-y-2">
          <input
            type="date"
            value={localParams.start_date || ''}
            onChange={(e) => setLocalParams(prev => ({ ...prev, start_date: e.target.value || null }))}
            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-white text-sm"
          />
          <input
            type="date"
            value={localParams.end_date || ''}
            onChange={(e) => setLocalParams(prev => ({ ...prev, end_date: e.target.value || null }))}
            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-white text-sm"
          />
        </div>
      </div>
    </div>
  );
}

export default SearchFilters;