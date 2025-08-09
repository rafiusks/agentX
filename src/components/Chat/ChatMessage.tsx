import { useState } from 'react'
import { User, Bot, Code, Cpu, Copy } from 'lucide-react'
import ReactMarkdown from 'react-markdown'
import { useUIStore } from '../../stores/ui.store'
import { FunctionCall } from '../FunctionCall'
import type { Message } from '../../hooks/queries/useChats'

interface ChatMessageProps {
  message: Message
}

export function ChatMessage({ message }: ChatMessageProps) {
  const [showActions, setShowActions] = useState(false)
  const [isCopied, setIsCopied] = useState(false)
  const isUser = message.role === 'user'
  const isFunction = message.role === 'function'
  const isAssistant = message.role === 'assistant'
  const { mode } = useUIStore()
  
  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(message.content)
      setIsCopied(true)
      setTimeout(() => setIsCopied(false), 2000)
    } catch (error) {
      console.error('Failed to copy:', error)
    }
  }
  
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
    <div 
      className={`flex gap-3 ${isUser ? 'justify-end' : ''} relative group message-animate-in`}
      onMouseEnter={() => setShowActions(true)}
      onMouseLeave={() => setShowActions(false)}
    >
      {!isUser && !isFunction && (
        <div className="message-avatar bg-gradient-to-br from-accent-blue to-accent-green">
          <Bot size={16} className="text-white" />
        </div>
      )}
      
      {isFunction && (
        <div className="message-avatar bg-amber-500/10">
          <Cpu size={16} className="text-amber-600" />
        </div>
      )}
      
      <div className={`
        relative
        ${isUser 
          ? 'message-user' 
          : isFunction
          ? 'message-function'
          : 'message-assistant'
        }
        ${message.isStreaming ? 'animate-pulse-subtle' : ''}
      `}>
        {/* Action buttons for assistant messages */}
        {isAssistant && !message.isStreaming && showActions && (
          <div className="absolute -top-9 right-0 flex items-center gap-1 bg-background-secondary 
                          border border-border-subtle rounded-lg px-2 py-1 shadow-lg">
            <button
              onClick={handleCopy}
              className="p-1 hover:bg-background-tertiary rounded transition-colors"
              title="Copy message"
            >
              {isCopied ? (
                <Code size={14} className="text-accent-green" />
              ) : (
                <Copy size={14} className="text-foreground-secondary" />
              )}
            </button>
          </div>
        )}
        
        {isUser ? (
          <p className="message-content-user">{message.content}</p>
        ) : (
          <div className="prose-chat">
            <ReactMarkdown
              components={{
                h1: ({ children }) => <h1 className="text-xl font-semibold mt-6 mb-3 text-foreground-primary">{children}</h1>,
                h2: ({ children }) => <h2 className="text-lg font-semibold mt-5 mb-2.5 text-foreground-primary">{children}</h2>,
                h3: ({ children }) => <h3 className="text-base font-semibold mt-4 mb-2 text-foreground-primary">{children}</h3>,
                p: ({ children }) => <p className="mb-4 text-foreground-primary">{children}</p>,
                ul: ({ children }) => <ul className="mb-4 ml-4 space-y-2">{children}</ul>,
                ol: ({ children }) => <ol className="mb-4 ml-4 space-y-2 list-decimal">{children}</ol>,
                li: ({ children }) => <li className="pl-2">{children}</li>,
                blockquote: ({ children }) => (
                  <blockquote className="border-l-4 border-accent-blue/30 pl-4 py-2 my-4 bg-accent-blue/[0.03] italic text-foreground-secondary">
                    {children}
                  </blockquote>
                ),
                hr: () => <hr className="my-6 border-border-subtle/30" />,
                pre: ({ children }) => (
                  <div className="message-code-block">
                    <pre className="m-0">
                      {children}
                    </pre>
                  </div>
                ),
                code: ({ children, ...props }) => {
                  const isInline = !props.className?.includes('language-');
                  return isInline 
                    ? <code className="message-inline-code">{children}</code>
                    : <code className="text-[13px]">{children}</code>
                },
                strong: ({ children }) => <strong className="font-semibold text-foreground-primary">{children}</strong>,
                em: ({ children }) => <em className="italic">{children}</em>,
                a: ({ children, href }) => (
                  <a href={href} className="text-accent-blue hover:text-accent-blue/80 underline underline-offset-2 transition-colors" target="_blank" rel="noopener noreferrer">
                    {children}
                  </a>
                ),
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
              <span>ID: {message.id ? message.id.slice(0, 8) : 'temp'}</span>
              {(message.created_at) && (
                <>
                  <span>•</span>
                  <span>{new Date(message.created_at).toLocaleTimeString()}</span>
                </>
              )}
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
        <div className="message-avatar bg-background-tertiary">
          <User size={16} className="text-foreground-secondary" />
        </div>
      )}
    </div>
  )
}