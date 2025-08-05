import { useState, useRef, useEffect } from 'react'
import { Send, Loader2 } from 'lucide-react'
import { useChatStore } from '../../stores/chat.store'
import { useUIStore } from '../../stores/ui.store'
import { ChatMessage } from './ChatMessage'
import { ChatSidebar } from './ChatSidebar'
import { api } from '../../services/api'

export function Chat() {
  const [input, setInput] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLTextAreaElement>(null)
  
  const { 
    sessions, 
    currentSessionId, 
    currentProvider,
    createSession, 
    addMessage,
    updateMessage 
  } = useChatStore()
  
  const { mode } = useUIStore()

  const currentSession = sessions.find(s => s.id === currentSessionId)

  useEffect(() => {
    // Create initial session if none exists
    if (sessions.length === 0) {
      createSession()
    }
  }, [])

  useEffect(() => {
    // Scroll to bottom on new messages
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [currentSession?.messages])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    
    if (!input.trim() || isLoading || !currentSessionId) return
    
    const userMessage = {
      id: Date.now().toString(),
      role: 'user' as const,
      content: input.trim()
    }
    
    addMessage(currentSessionId, userMessage)
    setInput('')
    setIsLoading(true)
    
    try {
      // Create assistant message placeholder
      const assistantMessage = {
        id: Date.now().toString() + '-assistant',
        role: 'assistant' as const,
        content: ''
      }
      addMessage(currentSessionId, assistantMessage)
      
      // Stream the response
      let fullContent = ''
      await api.streamChat(
        {
          messages: currentSession?.messages || [],
          session_id: currentSessionId,
          preferences: {
            provider: currentProvider
          }
        },
        (chunk) => {
          if (chunk.type === 'content') {
            fullContent += chunk.content
            updateMessage(currentSessionId, assistantMessage.id, fullContent)
          } else if (chunk.type === 'error') {
            console.error('Stream error:', chunk.error)
            updateMessage(currentSessionId, assistantMessage.id, 'Error: ' + chunk.error.message)
          }
        }
      )
    } catch (error) {
      console.error('Failed to send message:', error)
      addMessage(currentSessionId, {
        id: Date.now().toString() + '-error',
        role: 'assistant' as const,
        content: `Error: ${error instanceof Error ? error.message : 'Failed to send message'}`
      })
    } finally {
      setIsLoading(false)
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSubmit(e)
    }
  }

  const showSidebar = mode !== 'simple'

  return (
    <div className="flex flex-1 overflow-hidden">
      {showSidebar && <ChatSidebar />}
      
      <div className="flex-1 flex flex-col">
        {/* Messages */}
        <div className="flex-1 overflow-y-auto px-4 py-6">
          <div className="max-w-4xl mx-auto space-y-6">
            {currentSession?.messages.map((message) => (
              <ChatMessage key={message.id} message={message} />
            ))}
            {isLoading && (
              <div className="flex items-center gap-2 text-foreground-secondary">
                <Loader2 className="w-4 h-4 animate-spin" />
                <span>Thinking...</span>
              </div>
            )}
            <div ref={messagesEndRef} />
          </div>
        </div>

        {/* Input */}
        <form onSubmit={handleSubmit} className="border-t border-border-subtle">
          <div className="max-w-4xl mx-auto p-4">
            <div className="relative">
              <textarea
                ref={inputRef}
                value={input}
                onChange={(e) => setInput(e.target.value)}
                onKeyDown={handleKeyDown}
                placeholder="Send a message..."
                className="w-full px-4 py-3 pr-12 bg-background-secondary rounded-lg border border-border-subtle focus:border-accent-primary focus:outline-none resize-none"
                rows={1}
                style={{
                  minHeight: '48px',
                  maxHeight: '200px'
                }}
              />
              <button
                type="submit"
                disabled={!input.trim() || isLoading}
                className="absolute right-2 bottom-2 p-2 rounded-md bg-accent-primary text-white disabled:opacity-50 disabled:cursor-not-allowed hover:bg-accent-primary/90 transition-colors"
              >
                {isLoading ? (
                  <Loader2 className="w-4 h-4 animate-spin" />
                ) : (
                  <Send className="w-4 h-4" />
                )}
              </button>
            </div>
          </div>
        </form>
      </div>
    </div>
  )
}