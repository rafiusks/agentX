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

// Add a loading indicator while the app initializes
if (root) {
  root.innerHTML = '<div style="background: #0a0a0a; color: white; height: 100vh; display: flex; align-items: center; justify-content: center; font-family: Inter, system-ui, sans-serif;">Loading AgentX...</div>';
  
  // Small delay to ensure styles are loaded
  setTimeout(() => {
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
  }, 100);
} else {
  console.error('Root element not found!');
}