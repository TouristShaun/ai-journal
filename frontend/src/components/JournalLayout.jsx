import { useState } from 'react';
import { Search, BookMarked, Sparkles, GitMerge } from 'lucide-react';
import Sidebar from './Sidebar';
import JournalEntries from './JournalEntries';
import EntryEditor from './EntryEditor';
import { useEventStream } from '../hooks/useEventStream';

const tabs = [
  { id: 'classic', name: 'Classic Search', icon: Search },
  { id: 'vector', name: 'Vector Only', icon: Sparkles },
  { id: 'hybrid', name: 'Hybrid', icon: GitMerge },
];

function JournalLayout() {
  const [activeTab, setActiveTab] = useState('classic');
  const [selectedEntry, setSelectedEntry] = useState(null);
  const [searchParams, setSearchParams] = useState({
    query: '',
    is_favorite: null,
    collection_ids: [],
    search_type: 'classic',
  });
  
  // Initialize SSE connection
  useEventStream();

  const handleTabChange = (tabId) => {
    setActiveTab(tabId);
    setSearchParams(prev => ({
      ...prev,
      search_type: tabId,
    }));
  };

  const handleSearch = (params) => {
    setSearchParams(prev => ({
      ...prev,
      ...params,
      search_type: activeTab,
    }));
  };

  return (
    <div className="flex h-full">
      {/* Sidebar */}
      <div className="w-80 bg-white dark:bg-gray-800 border-r border-gray-200 dark:border-gray-700">
        <Sidebar
          activeTab={activeTab}
          onTabChange={handleTabChange}
          tabs={tabs}
          onSearch={handleSearch}
          searchParams={searchParams}
        />
      </div>

      {/* Main Content */}
      <div className="flex-1 flex">
        {/* Journal Entries List */}
        <div className="w-1/2 bg-gray-50 dark:bg-gray-900 overflow-y-auto">
          <JournalEntries
            searchParams={searchParams}
            onSelectEntry={setSelectedEntry}
            selectedEntry={selectedEntry}
          />
        </div>

        {/* Entry Detail/Editor */}
        <div className="w-1/2 bg-white dark:bg-gray-800 border-l border-gray-200 dark:border-gray-700">
          {selectedEntry ? (
            <EntryEditor
              entry={selectedEntry}
              onClose={() => setSelectedEntry(null)}
              onUpdate={() => {
                // Trigger refresh of entries
                setSearchParams(prev => ({ ...prev }));
              }}
            />
          ) : (
            <div className="h-full flex items-center justify-center text-gray-500 dark:text-gray-400">
              <div className="text-center">
                <BookMarked className="w-12 h-12 mx-auto mb-4 opacity-50" />
                <p>Select an entry to view or create a new one</p>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export default JournalLayout;