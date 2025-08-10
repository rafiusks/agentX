import { memo, useMemo } from 'react'
import { User, Bot, Cpu, Code } from 'lucide-react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { useUIStore } from '../../stores/ui.store'
import { FunctionCall } from '../FunctionCall'
import { SimpleMessageActions } from './SimpleMessageActions'
import { SearchIndicator } from './SearchIndicator'
import { SourceCitations } from './SourceCitations'
import { markdownComponents } from './markdownComponents'
import { parseSearchMetadata } from '../../utils/parse-search-metadata'
import type { Message } from '../../hooks/queries/useChats'

interface ChatMessageProps {
  message: Message
  onRegenerate?: (messageId: string) => void
}

// Memoize the component to prevent unnecessary re-renders
export const ChatMessage = memo(function ChatMessage({ message, onRegenerate }: ChatMessageProps) {
  const isUser = message.role === 'user'
  const isFunction = message.role === 'function'
  const isAssistant = message.role === 'assistant'
  const { mode } = useUIStore()
  
  // Parse search metadata from message content
  const parsedMessage = useMemo(() => {
    if (isAssistant && message.content) {
      return parseSearchMetadata(message.content)
    }
    return { content: message.content || '', sources: [] }
  }, [isAssistant, message.content])
  
  // Debug logging for code detection - disabled to prevent re-renders
  // if (isAssistant && message.content && !message.content.includes('```')) {
  //   const codePatterns = [
  //     'from ', 'import ', 'client =', 'response =', 'tools =', 
  //     'def ', 'class ', 'function ', 'const ', 'let '
  //   ];
  //   const hasCodePattern = codePatterns.some(pattern => message.content?.includes(pattern));
  //   if (hasCodePattern) {
  //     console.log(`[ChatMessage] Message ${message.id?.slice(0, 8)} contains code patterns but no code blocks:`, {
  //       id: message.id,
  //       content: message.content.substring(0, 200),
  //       hasCodeBlocks: message.content.includes('```')
  //     });
  //   }
  // }
  
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
      data-message-id={message.id}
      tabIndex={0}
      role="article"
      aria-label={`${message.role} message`}
    >
      {!isUser && !isFunction && (
        <div className={`message-avatar bg-gradient-to-br from-accent-blue to-accent-green 
                      ${message.isStreaming ? 'animate-shimmer' : ''}`}>
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
        ${message.isStreaming ? 'animate-stream-pulse' : ''}
      `}>
        {/* Message ID Badge */}
        <div className="absolute -top-2 -left-2 px-2 py-0.5 bg-background-tertiary border border-border-subtle rounded-full">
          <span className="text-[10px] font-mono text-foreground-muted">
            {message.id ? `#${message.id.slice(0, 8)}` : '#temp'}
          </span>
        </div>
        {/* Simple Copy/Regenerate Actions */}
        {!message.isStreaming && (
          <SimpleMessageActions 
            content={message.content || ''} 
            isAssistant={isAssistant}
            messageId={message.id}
            onRegenerate={isAssistant && onRegenerate ? () => onRegenerate(message.id!) : undefined}
          />
        )}
        
        {isUser ? (
          <p className="message-content-user" data-message-content>{message.content}</p>
        ) : (
          <>
            {/* Search Indicator */}
            {parsedMessage.searchMetadata && (
              <div className="mb-3">
                <SearchIndicator metadata={parsedMessage.searchMetadata} />
              </div>
            )}
            
            {/* Message Content */}
            <div className="prose-chat" data-message-content>
              <ReactMarkdown
                remarkPlugins={[remarkGfm]}
                components={markdownComponents}
              >
                {parsedMessage.content || (message.isStreaming ? '...' : '')}
              </ReactMarkdown>
            </div>
            
            {/* Source Citations */}
            {parsedMessage.sources.length > 0 && (
              <SourceCitations sources={parsedMessage.sources} />
            )}
          </>
        )}
        
        {/* Debug info for Pro mode */}
        {mode === 'pro' && !isUser && (
          <div className="mt-2 pt-2 border-t border-border-subtle">
            <div className="flex flex-col gap-1">
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
              <details className="text-xs">
                <summary className="cursor-pointer text-foreground-muted hover:text-foreground-secondary">
                  Raw content (first 200 chars)
                </summary>
                <pre className="mt-1 p-2 bg-background-tertiary rounded text-[10px] overflow-x-auto whitespace-pre-wrap">
                  {(message.content || '').substring(0, 200)}
                  {(message.content || '').length > 200 && '...'}
                </pre>
              </details>
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
}, (prevProps, nextProps) => {
  // Only re-render if message content or streaming state changes
  return (
    prevProps.message.id === nextProps.message.id &&
    prevProps.message.content === nextProps.message.content &&
    prevProps.message.isStreaming === nextProps.message.isStreaming &&
    prevProps.onRegenerate === nextProps.onRegenerate
  )
})