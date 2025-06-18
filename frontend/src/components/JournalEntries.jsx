import { useQuery } from '@tanstack/react-query';
import { journalAPI } from '../api/client';
import EntryCard from './EntryCard';
import CollectionView from './CollectionView';
import ExportButton from './ExportButton';
import { Loader2 } from 'lucide-react';

function JournalEntries({ searchParams, onSelectEntry, selectedEntry }) {
  const { data: entries = [], isLoading, error } = useQuery({
    queryKey: ['entries', searchParams],
    queryFn: () => journalAPI.search(searchParams),
    enabled: true,
  });

  // Group entries by collection if we're filtering by collections
  const isCollectionView = searchParams.collection_ids && searchParams.collection_ids.length > 0;

  if (isLoading) {
    return (
      <div className="h-full flex items-center justify-center">
        <Loader2 className="w-8 h-8 animate-spin text-blue-600" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="h-full flex items-center justify-center p-8">
        <div className="text-center text-red-600 dark:text-red-400">
          <p className="font-medium">Error loading entries</p>
          <p className="text-sm mt-1">{error.message}</p>
        </div>
      </div>
    );
  }

  if (entries.length === 0) {
    return (
      <div className="h-full flex items-center justify-center p-8">
        <div className="text-center text-gray-500 dark:text-gray-400">
          <p className="text-lg font-medium">No entries found</p>
          <p className="text-sm mt-1">Try adjusting your search criteria</p>
        </div>
      </div>
    );
  }

  if (isCollectionView) {
    return (
      <CollectionView
        entries={entries}
        collectionIds={searchParams.collection_ids}
        onSelectEntry={onSelectEntry}
        selectedEntry={selectedEntry}
      />
    );
  }

  // Default card view (newest first)
  return (
    <div className="h-full flex flex-col">
      {/* Header with export button */}
      <div className="p-4 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
        <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
          {entries.length} {entries.length === 1 ? 'Entry' : 'Entries'}
        </h2>
        <ExportButton searchParams={searchParams} />
      </div>
      
      {/* Entries list */}
      <div className="flex-1 overflow-y-auto p-4 space-y-3">
        {entries.map((entry) => (
        <EntryCard
          key={entry.id}
          entry={entry}
          isSelected={selectedEntry?.id === entry.id}
          onClick={() => onSelectEntry(entry)}
        />
      ))}
      </div>
    </div>
  );
}

export default JournalEntries;