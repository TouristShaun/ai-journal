import { useState } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import JournalLayout from './components/JournalLayout';
import ErrorBoundary from './components/ErrorBoundary';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 2,
      retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 30000),
      staleTime: 1000 * 60 * 5, // 5 minutes
      cacheTime: 1000 * 60 * 10, // 10 minutes
    },
    mutations: {
      retry: 1,
      retryDelay: 1000,
    },
  },
});

function App() {
  return (
    <ErrorBoundary>
      <QueryClientProvider client={queryClient}>
        <div className="h-screen bg-gray-50 dark:bg-gray-900">
          <JournalLayout />
        </div>
      </QueryClientProvider>
    </ErrorBoundary>
  );
}

export default App;