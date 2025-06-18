import { useState } from 'react';
import { X, Plus, Edit2, Trash2, Folder } from 'lucide-react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { journalAPI } from '../api/client';

function CollectionsModal({ isOpen, onClose }) {
  const [isCreating, setIsCreating] = useState(false);
  const [newCollectionName, setNewCollectionName] = useState('');
  const [newCollectionDescription, setNewCollectionDescription] = useState('');
  const queryClient = useQueryClient();

  const { data: collections = [], isLoading } = useQuery({
    queryKey: ['collections'],
    queryFn: journalAPI.getCollections,
    enabled: isOpen,
  });

  const { data: entries = [] } = useQuery({
    queryKey: ['entries', { limit: 1000 }],
    queryFn: () => journalAPI.search({ limit: 1000 }),
    enabled: isOpen,
  });

  // Count entries per collection
  const entryCounts = {};
  entries.forEach(entry => {
    entry.collection_ids?.forEach(collId => {
      entryCounts[collId] = (entryCounts[collId] || 0) + 1;
    });
  });

  const createCollectionMutation = useMutation({
    mutationFn: ({ name, description }) => journalAPI.createCollection(name, description),
    onSuccess: () => {
      queryClient.invalidateQueries(['collections']);
      setNewCollectionName('');
      setNewCollectionDescription('');
      setIsCreating(false);
    },
  });

  const handleCreateCollection = () => {
    if (!newCollectionName.trim()) return;
    createCollectionMutation.mutate({
      name: newCollectionName,
      description: newCollectionDescription,
    });
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-gray-800 rounded-lg w-full max-w-2xl max-h-[80vh] flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-gray-200 dark:border-gray-700">
          <h2 className="text-xl font-semibold text-gray-900 dark:text-white flex items-center gap-2">
            <Folder className="w-5 h-5" />
            Collections
          </h2>
          <button
            onClick={onClose}
            className="p-1 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
          >
            <X className="w-5 h-5 text-gray-500 dark:text-gray-400" />
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-6">
          {/* Create New Collection */}
          <div className="mb-6">
            {!isCreating ? (
              <button
                onClick={() => setIsCreating(true)}
                className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
              >
                <Plus className="w-4 h-4" />
                New Collection
              </button>
            ) : (
              <div className="bg-gray-50 dark:bg-gray-900 p-4 rounded-lg space-y-3">
                <input
                  type="text"
                  value={newCollectionName}
                  onChange={(e) => setNewCollectionName(e.target.value)}
                  placeholder="Collection name"
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  autoFocus
                />
                <textarea
                  value={newCollectionDescription}
                  onChange={(e) => setNewCollectionDescription(e.target.value)}
                  placeholder="Description (optional)"
                  rows={2}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white resize-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
                <div className="flex gap-2">
                  <button
                    onClick={handleCreateCollection}
                    disabled={!newCollectionName.trim() || createCollectionMutation.isPending}
                    className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    {createCollectionMutation.isPending ? 'Creating...' : 'Create'}
                  </button>
                  <button
                    onClick={() => {
                      setIsCreating(false);
                      setNewCollectionName('');
                      setNewCollectionDescription('');
                    }}
                    className="px-4 py-2 bg-gray-200 dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg hover:bg-gray-300 dark:hover:bg-gray-600 transition-colors"
                  >
                    Cancel
                  </button>
                </div>
              </div>
            )}
          </div>

          {/* Collections List */}
          {isLoading ? (
            <div className="text-center py-8 text-gray-500 dark:text-gray-400">
              Loading collections...
            </div>
          ) : collections.length === 0 ? (
            <div className="text-center py-8 text-gray-500 dark:text-gray-400">
              No collections yet. Create your first collection to organize your entries.
            </div>
          ) : (
            <div className="space-y-3">
              {collections.map((collection) => (
                <div
                  key={collection.id}
                  className="flex items-center justify-between p-4 bg-gray-50 dark:bg-gray-900 rounded-lg"
                >
                  <div className="flex-1">
                    <h3 className="font-medium text-gray-900 dark:text-white">
                      {collection.name}
                    </h3>
                    {collection.description && (
                      <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                        {collection.description}
                      </p>
                    )}
                    <p className="text-sm text-gray-500 dark:text-gray-500 mt-1">
                      {entryCounts[collection.id] || 0} entries
                    </p>
                  </div>
                  <div className="flex items-center gap-2">
                    <button
                      className="p-2 hover:bg-gray-200 dark:hover:bg-gray-700 rounded-lg transition-colors"
                      title="Edit collection"
                    >
                      <Edit2 className="w-4 h-4 text-gray-600 dark:text-gray-400" />
                    </button>
                    <button
                      className="p-2 hover:bg-red-100 dark:hover:bg-red-900/20 rounded-lg transition-colors"
                      title="Delete collection"
                    >
                      <Trash2 className="w-4 h-4 text-red-600 dark:text-red-400" />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export default CollectionsModal;