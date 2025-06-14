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
          case 'entry.processing':
            // Update the entry status in the cache
            queryClient.setQueriesData(['entries'], (oldData) => {
              if (!oldData) return oldData;
              
              return oldData.map(entry => {
                if (entry.id === data.entry_id) {
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
            // Update the entry with processed data
            queryClient.setQueriesData(['entries'], (oldData) => {
              if (!oldData) return oldData;
              
              return oldData.map(entry => {
                if (entry.id === data.entry_id) {
                  return {
                    ...entry,
                    processed_data: data.data.processed_data,
                    updated_at: data.data.updated_at,
                    processing_stage: 'completed',
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
                processed_data: data.data.processed_data,
                updated_at: data.data.updated_at,
                processing_stage: 'completed',
                processing_completed_at: new Date().toISOString()
              };
            });
            
            // Also invalidate queries to ensure fresh data
            queryClient.invalidateQueries(['entries', { id: data.entry_id }]);
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