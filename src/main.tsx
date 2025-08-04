import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App.tsx'
import { ErrorBoundary } from './components/ErrorBoundary.tsx'
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
        <ErrorBoundary>
          <App />
        </ErrorBoundary>
      </React.StrictMode>,
    )
    console.log('React app rendered');
  }, 100);
} else {
  console.error('Root element not found!');
}