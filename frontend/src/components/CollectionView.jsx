import { useQuery } from '@tanstack/react-query';
import { journalAPI } from '../api/client';
import EntryCard from './EntryCard';
import { Folder } from 'lucide-react';

function CollectionView({ entries, collectionIds, onSelectEntry, selectedEntry }) {
  const { data: collections = [] } = useQuery({
    queryKey: ['collections'],
    queryFn: journalAPI.getCollections,
  });

  // Group entries by collection
  const entriesByCollection = {};
  
  entries.forEach(entry => {
    entry.collection_ids.forEach(collId => {
      if (collectionIds.includes(collId)) {
        if (!entriesByCollection[collId]) {
          entriesByCollection[collId] = [];
        }
        entriesByCollection[collId].push(entry);
      }
    });
  });

  // Sort collections alphabetically and their entries by date (newest first)
  const sortedCollections = collections
    .filter(coll => collectionIds.includes(coll.id))
    .sort((a, b) => a.name.localeCompare(b.name));

  return (
    <div className="p-4 space-y-6">
      {sortedCollections.map(collection => {
        const collectionEntries = entriesByCollection[collection.id] || [];
        
        if (collectionEntries.length === 0) return null;

        // Sort entries by date (newest first)
        const sortedEntries = [...collectionEntries].sort(
          (a, b) => new Date(b.created_at) - new Date(a.created_at)
        );

        return (
          <div key={collection.id} className="space-y-3">
            {/* Collection Header */}
            <div className="flex items-center gap-2 px-2">
              <Folder className="w-4 h-4 text-blue-600 dark:text-blue-400" />
              <h3 className="font-semibold text-gray-900 dark:text-white">
                {collection.name}
              </h3>
              <span className="text-sm text-gray-500 dark:text-gray-400">
                ({sortedEntries.length})
              </span>
            </div>

            {/* Horizontal Scroll Container */}
            <div className="relative">
              <div className="flex gap-3 overflow-x-auto pb-2 scrollbar-thin scrollbar-thumb-gray-300 dark:scrollbar-thumb-gray-600">
                {sortedEntries.map((entry) => (
                  <div key={entry.id} className="flex-shrink-0 w-80">
                    <EntryCard
                      entry={entry}
                      isSelected={selectedEntry?.id === entry.id}
                      onClick={() => onSelectEntry(entry)}
                    />
                  </div>
                ))}
              </div>
            </div>
          </div>
        );
      })}
    </div>
  );
}

export default CollectionView;