import { useState } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import JournalLayout from './components/JournalLayout';

const queryClient = new QueryClient();

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <div className="h-screen bg-gray-50 dark:bg-gray-900">
        <JournalLayout />
      </div>
    </QueryClientProvider>
  );
}

export default App;