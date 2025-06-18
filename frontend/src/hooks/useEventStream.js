import { useEffect, useRef, useCallback } from 'react';
import { useQueryClient } from '@tanstack/react-query';

const SSE_URL = 'http://localhost:8080/api/events';

export function useEventStream() {
  const eventSourceRef = useRef(null);
  const reconnectTimeoutRef = useRef(null);
  const reconnectAttemptsRef = useRef(0);
  const queryClient = useQueryClient();

  const connect = useCallback(() => {
    // Clean up any existing connection
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
    }

    console.log('Connecting to SSE endpoint...');
    const eventSource = new EventSource(SSE_URL);
    eventSourceRef.current = eventSource;

    eventSource.onopen = () => {
      console.log('SSE connection established');
      reconnectAttemptsRef.current = 0;
    };

    eventSource.onerror = (error) => {
      console.error('SSE connection error:', error);
      eventSource.close();
      
      // Implement exponential backoff for reconnection
      const backoffTime = Math.min(1000 * Math.pow(2, reconnectAttemptsRef.current), 30000);
      reconnectAttemptsRef.current++;
      
      console.log(`Reconnecting in ${backoffTime}ms...`);
      reconnectTimeoutRef.current = setTimeout(() => {
        connect();
      }, backoffTime);
    };

    // Handle custom connected event
    eventSource.addEventListener('connected', (event) => {
      const data = JSON.parse(event.data);
      console.log('Connected with client ID:', data.client_id);
    });

    // Handle journal entry events
    eventSource.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        console.log('Received event:', data);

        switch (data.type) {
          case 'entry.created':
            // Add new entry to the cache
            queryClient.setQueriesData(['entries'], (oldData) => {
              if (!oldData) return [{
                ...data.data.entry,
                processing_started_at: data.data.entry.processing_started_at || new Date().toISOString()
              }];
              
              // Check if entry already exists (prevent duplicates)
              const exists = oldData.some(entry => entry.id === data.data.entry.id);
              if (exists) return oldData;
              
              // Add new entry at the beginning with processing timestamp
              return [{
                ...data.data.entry,
                processing_started_at: data.data.entry.processing_started_at || new Date().toISOString()
              }, ...oldData];
            });
            
            // Invalidate to ensure proper sorting and filtering
            queryClient.invalidateQueries(['entries']);
            break;

          case 'entry.updated':
            // Update the entry in cache
            queryClient.setQueriesData(['entries'], (oldData) => {
              if (!oldData) return oldData;
              
              return oldData.map(entry => {
                if (entry.id === data.data.original_id) {
                  return data.data.entry;
                }
                return entry;
              });
            });
            
            // Update single entry query
            queryClient.setQueryData(['entry', data.data.original_id], data.data.entry);
            queryClient.setQueryData(['entry', data.data.entry.id], data.data.entry);
            
            // Invalidate to refresh
            queryClient.invalidateQueries(['entries']);
            break;

          case 'entry.deleted':
            // Remove entry from cache
            queryClient.setQueriesData(['entries'], (oldData) => {
              if (!oldData) return oldData;
              return oldData.filter(entry => entry.id !== data.entry_id);
            });
            
            // Remove from single entry cache
            queryClient.removeQueries(['entry', data.entry_id]);
            break;

          case 'entry.processing':
            console.log('Processing stage update received:', data);
            
            // Update the entry status in the cache
            queryClient.setQueriesData(['entries'], (oldData) => {
              if (!oldData) return oldData;
              
              return oldData.map(entry => {
                if (entry.id === data.entry_id) {
                  console.log(`Updating entry ${data.entry_id} to stage: ${data.data.stage}`);
                  return {
                    ...entry,
                    processing_stage: data.data.stage || 'analyzing',
                    processing_started_at: entry.processing_started_at || new Date().toISOString()
                  };
                }
                return entry;
              });
            });
            
            // Also update single entry queries
            queryClient.setQueryData(['entry', data.entry_id], (oldEntry) => {
              if (!oldEntry) return oldEntry;
              return {
                ...oldEntry,
                processing_stage: data.data.stage || 'analyzing',
                processing_started_at: oldEntry.processing_started_at || new Date().toISOString()
              };
            });
            break;

          case 'entry.processed':
            console.log('Processing completed event received:', data);
            
            // Update the entry with processed data
            queryClient.setQueriesData(['entries'], (oldData) => {
              if (!oldData) return oldData;
              
              return oldData.map(entry => {
                if (entry.id === data.entry_id) {
                  // Always expect entry data in the event
                  if (data.data && data.data.entry) {
                    console.log('Updating entry with processed data:', data.data.entry);
                    return {
                      ...data.data.entry,
                      processing_stage: 'completed',
                      processing_completed_at: data.data.entry.processing_completed_at || new Date().toISOString()
                    };
                  } else {
                    console.warn('entry.processed event missing entry data:', data);
                    // Fallback: just update the stage
                    return {
                      ...entry,
                      processing_stage: 'completed',
                      processing_completed_at: new Date().toISOString()
                    };
                  }
                }
                return entry;
              });
            });
            
            // Update single entry queries
            queryClient.setQueryData(['entry', data.entry_id], (oldEntry) => {
              if (!oldEntry) return oldEntry;
              if (data.data && data.data.entry) {
                return {
                  ...data.data.entry,
                  processing_stage: 'completed',
                  processing_completed_at: data.data.entry.processing_completed_at || new Date().toISOString()
                };
              } else {
                return {
                  ...oldEntry,
                  processing_stage: 'completed',
                  processing_completed_at: new Date().toISOString()
                };
              }
            });
            
            // Ensure the UI updates immediately
            queryClient.invalidateQueries(['entries']);
            break;

          case 'entry.failed':
            // Update the entry to show error status
            queryClient.setQueriesData(['entries'], (oldData) => {
              if (!oldData) return oldData;
              
              return oldData.map(entry => {
                if (entry.id === data.entry_id) {
                  return {
                    ...entry,
                    processing_stage: 'failed',
                    processing_error: data.data.error,
                    processing_completed_at: new Date().toISOString()
                  };
                }
                return entry;
              });
            });
            
            // Update single entry queries
            queryClient.setQueryData(['entry', data.entry_id], (oldEntry) => {
              if (!oldEntry) return oldEntry;
              return {
                ...oldEntry,
                processing_stage: 'failed',
                processing_error: data.data.error,
                processing_completed_at: new Date().toISOString()
              };
            });
            break;

          default:
            console.log('Unknown event type:', data.type);
        }
      } catch (error) {
        console.error('Error processing SSE event:', error);
      }
    };
  }, [queryClient]);

  useEffect(() => {
    connect();

    // Cleanup on unmount
    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
      }
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
    };
  }, [connect]);

  // Expose manual reconnect function
  const reconnect = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
    }
    reconnectAttemptsRef.current = 0;
    connect();
  }, [connect]);

  return { reconnect };
}