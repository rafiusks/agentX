import { User, Bot, Code, Cpu } from 'lucide-react'
import ReactMarkdown from 'react-markdown'
import { Message } from '../../stores/chat.store'
import { useUIStore } from '../../stores/ui.store'
import { FunctionCall } from '../FunctionCall'

interface ChatMessageProps {
  message: Message
}

export function ChatMessage({ message }: ChatMessageProps) {
  const isUser = message.role === 'user'
  const isFunction = message.role === 'function'
  const { mode } = useUIStore()
  
  // Show function calls inline
  if (message.functionCall && message.role === 'assistant') {
    return (
      <div className="space-y-2">
        <FunctionCall 
          name={message.functionCall.name} 
          arguments={message.functionCall.arguments} 
          isExecuting={message.isStreaming}
        />
        {message.content && (
          <ChatMessage message={{ ...message, functionCall: undefined }} />
        )}
      </div>
    )
  }
  
  return (
    <div className={`flex gap-4 ${isUser ? 'justify-end' : ''}`}>
      {!isUser && !isFunction && (
        <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-accent-blue to-accent-green 
                      flex items-center justify-center flex-shrink-0">
          <Bot size={16} className="text-white" />
        </div>
      )}
      
      {isFunction && (
        <div className="w-8 h-8 rounded-lg bg-accent/20 
                      flex items-center justify-center flex-shrink-0">
          <Cpu size={16} className="text-accent" />
        </div>
      )}
      
      <div className={`
        max-w-[80%] rounded-lg px-4 py-3
        ${isUser 
          ? 'bg-accent-blue text-white' 
          : 'bg-background-secondary border border-border-subtle'
        }
        ${message.isStreaming ? 'animate-pulse-subtle' : ''}
      `}>
        {isUser ? (
          <p className="text-sm whitespace-pre-wrap">{message.content}</p>
        ) : (
          <div className="prose prose-invert prose-sm max-w-none">
            <ReactMarkdown
              components={{
                pre: ({ children }) => (
                  <pre className="bg-background-tertiary rounded-md p-3 overflow-x-auto">
                    {children}
                  </pre>
                ),
                code: ({ children, ...props }) => {
                  const isInline = !props.className?.includes('language-');
                  return isInline 
                    ? <code className="bg-background-tertiary px-1 py-0.5 rounded text-xs">{children}</code>
                    : <code>{children}</code>
                },
              }}
            >
              {message.content || (message.isStreaming ? '...' : '')}
            </ReactMarkdown>
          </div>
        )}
        
        {/* Debug info for Pro mode */}
        {mode === 'pro' && !isUser && (
          <div className="mt-2 pt-2 border-t border-border-subtle">
            <div className="flex items-center gap-2 text-xs text-foreground-muted">
              <Code size={12} />
              <span>ID: {message.id.slice(0, 8)}</span>
              <span>•</span>
              <span>{new Date(message.timestamp).toLocaleTimeString()}</span>
              {message.isStreaming && (
                <>
                  <span>•</span>
                  <span className="text-accent-blue">Streaming</span>
                </>
              )}
            </div>
          </div>
        )}
      </div>
      
      {isUser && (
        <div className="w-8 h-8 rounded-lg bg-background-tertiary 
                      flex items-center justify-center flex-shrink-0">
          <User size={16} className="text-foreground-secondary" />
        </div>
      )}
    </div>
  )
}