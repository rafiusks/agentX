import { AlertCircle, RefreshCw } from 'lucide-react'
import { memo } from 'react'

interface ErrorMessageProps {
  error: string | Error
  onRetry?: () => void
  timestamp?: string
}

export const ErrorMessage = memo(function ErrorMessage({ 
  error, 
  onRetry,
  timestamp 
}: ErrorMessageProps) {
  const errorMessage = typeof error === 'string' ? error : error.message
  
  return (
    <div className="flex gap-3 relative group message-animate-in">
      <div className="message-avatar bg-red-500/10">
        <AlertCircle size={16} className="text-red-500" />
      </div>
      
      <div className="bg-red-500/5 border border-red-500/20 rounded-2xl px-6 py-5 max-w-[1200px]">
        <div className="flex items-start justify-between gap-4">
          <div className="flex-1">
            <p className="text-red-400 font-medium mb-2">Error</p>
            <p className="text-foreground-secondary text-[15px] leading-relaxed">
              {errorMessage}
            </p>
            {timestamp && (
              <p className="text-xs text-foreground-muted mt-2">
                {new Date(timestamp).toLocaleTimeString()}
              </p>
            )}
          </div>
          
          {onRetry && (
            <button
              onClick={onRetry}
              className="p-2 rounded-lg bg-red-500/10 hover:bg-red-500/20 
                       text-red-400 transition-all duration-200"
              title="Retry"
            >
              <RefreshCw size={16} />
            </button>
          )}
        </div>
      </div>
    </div>
  )
})