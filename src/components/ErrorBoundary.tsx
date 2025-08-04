import React from 'react'

interface ErrorBoundaryState {
  hasError: boolean
  error: Error | null
}

export class ErrorBoundary extends React.Component<
  { children: React.ReactNode },
  ErrorBoundaryState
> {
  constructor(props: { children: React.ReactNode }) {
    super(props)
    this.state = { hasError: false, error: null }
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error }
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    console.error('Error caught by boundary:', error, errorInfo)
  }

  render() {
    if (this.state.hasError) {
      return (
        <div style={{
          backgroundColor: '#0a0a0a',
          color: 'white',
          minHeight: '100vh',
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          padding: '20px',
          fontFamily: 'Inter, system-ui, sans-serif'
        }}>
          <h1 style={{ fontSize: '24px', marginBottom: '10px' }}>Something went wrong</h1>
          <p style={{ color: '#a0a0a0', marginBottom: '20px' }}>An error occurred while loading AgentX</p>
          <pre style={{
            backgroundColor: '#1a1a1a',
            padding: '15px',
            borderRadius: '8px',
            maxWidth: '600px',
            overflow: 'auto'
          }}>
            {this.state.error?.toString()}
          </pre>
        </div>
      )
    }

    return this.props.children
  }
}