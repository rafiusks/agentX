import React from 'react'
import ReactDOM from 'react-dom/client'
import { QueryClientProvider } from '@tanstack/react-query'
import { ReactQueryDevtools } from '@tanstack/react-query-devtools'
import AppWithAuth from './AppWithAuth.tsx'
import { ErrorBoundary } from './components/ErrorBoundary.tsx'
import { queryClient } from './lib/query-client'
import './styles/globals.css'

console.log('Main.tsx loading...');

const root = document.getElementById('root');
console.log('Root element:', root);

// Render the app directly
if (root) {
  ReactDOM.createRoot(root).render(
    <React.StrictMode>
      <QueryClientProvider client={queryClient}>
        <ErrorBoundary>
          <AppWithAuth />
        </ErrorBoundary>
        <ReactQueryDevtools initialIsOpen={false} />
      </QueryClientProvider>
    </React.StrictMode>,
  )
  console.log('React app rendered');
} else {
  console.error('Root element not found!');
}