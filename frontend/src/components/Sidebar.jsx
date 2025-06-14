import { useState, useEffect } from 'react';
import { Plus, Star, Calendar, Hash } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { journalAPI } from '../api/client';
import SearchFilters from './SearchFilters';
import VectorSearch from './VectorSearch';
import HybridSearch from './HybridSearch';

function Sidebar({ activeTab, onTabChange, tabs, onSearch, searchParams }) {
  const [isCreatingEntry, setIsCreatingEntry] = useState(false);
  const [newEntryContent, setNewEntryContent] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState('');

  const { data: collections = [] } = useQuery({
    queryKey: ['collections'],
    queryFn: journalAPI.getCollections,
  });

  const handleCreateEntry = async () => {
    if (!newEntryContent.trim()) return;
    
    setIsSubmitting(true);
    setError('');
    
    try {
      await journalAPI.createEntry(newEntryContent);
      setNewEntryContent('');
      setIsCreatingEntry(false);
      // Trigger refresh
      onSearch({ ...searchParams });
    } catch (error) {
      console.error('Failed to create entry:', error);
      setError('Failed to create entry. Please try again.');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="h-full flex flex-col">
      {/* Header */}
      <div className="p-4 border-b border-gray-200 dark:border-gray-700">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Journal</h1>
        <button
          onClick={() => setIsCreatingEntry(true)}
          className="mt-4 w-full flex items-center justify-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
        >
          <Plus className="w-4 h-4" />
          New Entry
        </button>
      </div>

      {/* Tabs */}
      <div className="flex border-b border-gray-200 dark:border-gray-700">
        {tabs.map((tab) => {
          const Icon = tab.icon;
          return (
            <button
              key={tab.id}
              onClick={() => onTabChange(tab.id)}
              className={`flex-1 flex items-center justify-center gap-2 px-4 py-3 text-sm font-medium transition-colors ${
                activeTab === tab.id
                  ? 'text-blue-600 border-b-2 border-blue-600 bg-blue-50 dark:bg-blue-900/20'
                  : 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white'
              }`}
            >
              <Icon className="w-4 h-4" />
              <span className="hidden lg:inline">{tab.name}</span>
            </button>
          );
        })}
      </div>

      {/* Tab Content */}
      <div className="flex-1 overflow-y-auto p-4">
        {activeTab === 'classic' && (
          <SearchFilters
            collections={collections}
            onSearch={onSearch}
            searchParams={searchParams}
          />
        )}
        {activeTab === 'vector' && (
          <VectorSearch
            onSearch={onSearch}
            searchParams={searchParams}
          />
        )}
        {activeTab === 'hybrid' && (
          <HybridSearch
            collections={collections}
            onSearch={onSearch}
            searchParams={searchParams}
          />
        )}
      </div>

      {/* New Entry Modal */}
      {isCreatingEntry && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-800 rounded-lg p-6 w-full max-w-2xl mx-4">
            <h2 className="text-xl font-semibold mb-4 text-gray-900 dark:text-white">
              New Journal Entry
            </h2>
            <textarea
              value={newEntryContent}
              onChange={(e) => setNewEntryContent(e.target.value)}
              className="w-full h-64 p-4 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-white resize-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              placeholder="Write your journal entry here..."
              autoFocus
            />
            {error && (
              <div className="mt-2 text-red-600 dark:text-red-400 text-sm">
                {error}
              </div>
            )}
            <div className="flex gap-3 mt-4">
              <button
                onClick={handleCreateEntry}
                disabled={isSubmitting || !newEntryContent.trim()}
                className="flex-1 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {isSubmitting ? 'Creating...' : 'Create Entry'}
              </button>
              <button
                onClick={() => {
                  setIsCreatingEntry(false);
                  setNewEntryContent('');
                  setError('');
                }}
                disabled={isSubmitting}
                className="flex-1 px-4 py-2 bg-gray-200 dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg hover:bg-gray-300 dark:hover:bg-gray-600 transition-colors disabled:opacity-50"
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default Sidebar;