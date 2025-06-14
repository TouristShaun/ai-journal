import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { journalAPI } from '../api/client';
import { format } from 'date-fns';
import { X, Star, Edit2, Save, Tag, Link2, Brain, Calendar, Hash } from 'lucide-react';
import ProcessingTracker from './ProcessingTracker';
import ProcessingLogsModal from './ProcessingLogsModal';

function EntryEditor({ entry, onClose, onUpdate }) {
  const [isEditing, setIsEditing] = useState(false);
  const [editContent, setEditContent] = useState(entry.content);
  const [showLogsModal, setShowLogsModal] = useState(false);
  const queryClient = useQueryClient();

  const { data: collections = [] } = useQuery({
    queryKey: ['collections'],
    queryFn: journalAPI.getCollections,
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, content }) => journalAPI.updateEntry(id, content),
    onSuccess: () => {
      queryClient.invalidateQueries(['entries']);
      setIsEditing(false);
      onUpdate();
    },
  });

  const toggleFavoriteMutation = useMutation({
    mutationFn: (id) => journalAPI.toggleFavorite(id),
    onSuccess: () => {
      queryClient.invalidateQueries(['entries']);
      onUpdate();
    },
  });

  const handleSave = () => {
    if (editContent.trim() !== entry.content) {
      updateMutation.mutate({ id: entry.id, content: editContent });
    } else {
      setIsEditing(false);
    }
  };

  const handleToggleFavorite = async () => {
    try {
      await toggleFavoriteMutation.mutateAsync(entry.id);
      // Update local state immediately for better UX
      entry.is_favorite = !entry.is_favorite;
    } catch (error) {
      console.error('Failed to toggle favorite:', error);
    }
  };

  return (
    <div className="h-full flex flex-col">
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b border-gray-200 dark:border-gray-700">
        <div className="flex items-center gap-4">
          <button
            onClick={onClose}
            className="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
          >
            <X className="w-5 h-5" />
          </button>
          <div>
            <div className="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
              <Calendar className="w-4 h-4" />
              <time>{format(new Date(entry.created_at), 'MMMM d, yyyy • h:mm a')}</time>
            </div>
            {entry.updated_at !== entry.created_at && (
              <div className="text-xs text-gray-400 dark:text-gray-500 mt-1">
                Updated {format(new Date(entry.updated_at), 'MMM d, yyyy')}
              </div>
            )}
          </div>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={handleToggleFavorite}
            disabled={toggleFavoriteMutation.isPending}
            className={`p-2 rounded-lg transition-colors ${
              toggleFavoriteMutation.isPending ? 'opacity-50 cursor-not-allowed' :
              entry.is_favorite
                ? 'text-yellow-500 hover:bg-yellow-50 dark:hover:bg-yellow-900/20'
                : 'text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700'
            }`}
          >
            <Star className="w-5 h-5" fill={entry.is_favorite ? 'currentColor' : 'none'} />
          </button>
          {!isEditing ? (
            <button
              onClick={() => setIsEditing(true)}
              className="flex items-center gap-2 px-3 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
            >
              <Edit2 className="w-4 h-4" />
              Edit
            </button>
          ) : (
            <button
              onClick={handleSave}
              disabled={updateMutation.isPending}
              className="flex items-center gap-2 px-3 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 transition-colors disabled:opacity-50"
            >
              <Save className="w-4 h-4" />
              {updateMutation.isPending ? 'Saving...' : 'Save'}
            </button>
          )}
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-6">
        {/* Processing Tracker */}
        {entry.processing_stage && entry.processing_stage !== 'completed' && (
          <ProcessingTracker 
            entry={entry} 
            onViewLogs={() => setShowLogsModal(true)}
          />
        )}
        
        {isEditing ? (
          <textarea
            value={editContent}
            onChange={(e) => setEditContent(e.target.value)}
            className="w-full h-64 p-4 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-white resize-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            autoFocus
          />
        ) : (
          <div className="prose dark:prose-invert max-w-none">
            <p className="whitespace-pre-wrap text-gray-600 dark:text-gray-400">{entry.content}</p>
          </div>
        )}

        {/* Metadata */}
        <div className="mt-8 space-y-6">
          {/* AI Analysis */}
          <div className="bg-gray-50 dark:bg-gray-800 rounded-lg p-4">
            <h3 className="font-semibold text-gray-900 dark:text-white mb-3 flex items-center gap-2">
              <Brain className="w-5 h-5" />
              AI Analysis
            </h3>
            
            <div className="space-y-3">
              <div>
                <h4 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Summary</h4>
                <p className="text-sm text-gray-600 dark:text-gray-400">
                  {entry.processed_data.summary}
                </p>
              </div>

              <div>
                <h4 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Sentiment</h4>
                <span className={`text-sm font-medium ${
                  entry.processed_data.sentiment === 'positive' ? 'text-green-600' :
                  entry.processed_data.sentiment === 'negative' ? 'text-red-600' :
                  entry.processed_data.sentiment === 'mixed' ? 'text-yellow-600' :
                  'text-gray-600'
                }`}>
                  {entry.processed_data.sentiment}
                </span>
              </div>

              {entry.processed_data.entities.length > 0 && (
                <div>
                  <h4 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Entities</h4>
                  <div className="flex flex-wrap gap-2">
                    {entry.processed_data.entities.map((entity) => (
                      <span
                        key={entity}
                        className="inline-flex items-center gap-1 px-2 py-1 bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 rounded text-xs"
                      >
                        <Hash className="w-3 h-3" />
                        {entity}
                      </span>
                    ))}
                  </div>
                </div>
              )}

              {entry.processed_data.topics.length > 0 && (
                <div>
                  <h4 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Topics</h4>
                  <div className="flex flex-wrap gap-2">
                    {entry.processed_data.topics.map((topic) => (
                      <span
                        key={topic}
                        className="inline-flex items-center gap-1 px-2 py-1 bg-purple-100 dark:bg-purple-900/30 text-purple-700 dark:text-purple-300 rounded text-xs"
                      >
                        <Tag className="w-3 h-3" />
                        {topic}
                      </span>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </div>

          {/* Extracted URLs */}
          {entry.processed_data.extracted_urls?.length > 0 && (
            <div className="bg-gray-50 dark:bg-gray-800 rounded-lg p-4">
              <h3 className="font-semibold text-gray-900 dark:text-white mb-3 flex items-center gap-2">
                <Link2 className="w-5 h-5" />
                Extracted Links
              </h3>
              <div className="space-y-2">
                {entry.processed_data.extracted_urls.map((url, idx) => (
                  <div key={idx} className="p-3 bg-white dark:bg-gray-700 rounded border border-gray-200 dark:border-gray-600">
                    <a
                      href={url.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-blue-600 dark:text-blue-400 hover:underline text-sm font-medium"
                    >
                      {url.title || url.url}
                    </a>
                    {url.content && (
                      <p className="text-xs text-gray-600 dark:text-gray-400 mt-1 line-clamp-2">
                        {url.content}
                      </p>
                    )}
                    <div className="text-xs text-gray-500 dark:text-gray-500 mt-1">
                      From: {url.source} • {format(new Date(url.extracted_at), 'MMM d, yyyy')}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Collections */}
          <div className="bg-gray-50 dark:bg-gray-800 rounded-lg p-4">
            <h3 className="font-semibold text-gray-900 dark:text-white mb-3 flex items-center gap-2">
              <Tag className="w-5 h-5" />
              Collections
            </h3>
            <div className="space-y-2">
              {collections.map(collection => {
                const isInCollection = entry.collection_ids.includes(collection.id);
                return (
                  <label
                    key={collection.id}
                    className="flex items-center gap-3 p-2 rounded cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
                  >
                    <input
                      type="checkbox"
                      checked={isInCollection}
                      onChange={(e) => {
                        if (e.target.checked) {
                          journalAPI.addToCollection(entry.id, collection.id).then(() => {
                            queryClient.invalidateQueries(['entries']);
                            onUpdate();
                          });
                        } else {
                          journalAPI.removeFromCollection(entry.id, collection.id).then(() => {
                            queryClient.invalidateQueries(['entries']);
                            onUpdate();
                          });
                        }
                      }}
                      className="w-4 h-4 text-blue-600 rounded focus:ring-blue-500"
                    />
                    <span className="text-sm text-gray-700 dark:text-gray-300">
                      {collection.name}
                    </span>
                  </label>
                );
              })}
            </div>
          </div>
        </div>
      </div>
      
      {/* Processing Logs Modal */}
      <ProcessingLogsModal
        entryId={entry.id}
        isOpen={showLogsModal}
        onClose={() => setShowLogsModal(false)}
      />
    </div>
  );
}

export default EntryEditor;