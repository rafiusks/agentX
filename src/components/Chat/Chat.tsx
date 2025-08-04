import { useState, useRef, useEffect } from 'react'
import { invoke } from '@tauri-apps/api/core'
import { listen } from '@tauri-apps/api/event'
import { Send, Loader2 } from 'lucide-react'
import { useChatStore } from '../../stores/chat.store'
import { useUIStore } from '../../stores/ui.store'
import { ChatMessage } from './ChatMessage'
import { ChatSidebar } from './ChatSidebar'

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

  useEffect(() => {
    if (!currentSessionId || !currentSession) return

    let unlisten: (() => void) | undefined
    let unlistenEnd: (() => void) | undefined
    let unlistenError: (() => void) | undefined

    const setupListeners = async () => {
      // Set up event listeners for streaming
      unlisten = await listen<string>('stream-chunk', (event) => {
        console.log('Received stream chunk:', event.payload);
        // Use store to get the latest session data
        const state = useChatStore.getState()
        const session = state.sessions.find(s => s.id === currentSessionId)
        if (!session) return
        
        const messages = session.messages
        const lastMessage = messages[messages.length - 1]
        
        if (lastMessage && lastMessage.role === 'assistant' && lastMessage.isStreaming) {
          const currentContent = lastMessage.content === 'Thinking...' ? '' : lastMessage.content
          updateMessage(currentSessionId, lastMessage.id, currentContent + event.payload, true)
        }
      })

      unlistenEnd = await listen('stream-end', () => {
        console.log('Stream ended');
        // Use store to get the latest session data
        const state = useChatStore.getState()
        const session = state.sessions.find(s => s.id === currentSessionId)
        if (!session) return
        
        const messages = session.messages
        const lastMessage = messages[messages.length - 1]
        
        if (lastMessage && lastMessage.role === 'assistant') {
          updateMessage(currentSessionId, lastMessage.id, lastMessage.content, false)
        }
        setIsLoading(false)
      })

      unlistenError = await listen<string>('stream-error', (event) => {
        console.error('Stream error:', event.payload)
        // Use store to get the latest session data
        const state = useChatStore.getState()
        const session = state.sessions.find(s => s.id === currentSessionId)
        if (!session) return
        
        const messages = session.messages
        const lastMessage = messages[messages.length - 1]
        
        if (lastMessage && lastMessage.role === 'assistant') {
          updateMessage(currentSessionId, lastMessage.id, `Error: ${event.payload}`, false)
        }
        setIsLoading(false)
      })
    }

    setupListeners().catch(console.error)

    // Cleanup
    return () => {
      if (unlisten) unlisten()
      if (unlistenEnd) unlistenEnd()
      if (unlistenError) unlistenError()
    }
  }, [currentSessionId, updateMessage])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    
    if (!input.trim() || isLoading || !currentSessionId || !currentProvider) return
    
    const userMessage = input.trim()
    setInput('')
    setIsLoading(true)
    
    // Add user message
    addMessage(currentSessionId, {
      role: 'user',
      content: userMessage
    })
    
    // Add placeholder assistant message with streaming flag
    addMessage(currentSessionId, {
      role: 'assistant',
      content: 'Thinking...',
      isStreaming: true
    })
    
    try {
      console.log('Sending message:', userMessage, 'with provider:', currentProvider);
      
      // Use streaming approach
      await invoke('stream_message', {
        message: userMessage,
        providerId: currentProvider
      })
      
      // Loading state will be set to false by the stream-end event
    } catch (error) {
      console.error('Failed to send message:', error)
      setIsLoading(false)
      
      // Update last message with error
      if (currentSession && currentSession.messages) {
        const lastMessage = currentSession.messages[currentSession.messages.length - 1]
        if (lastMessage && lastMessage.role === 'assistant') {
          updateMessage(currentSessionId, lastMessage.id, `Error: ${error}`)
        }
      }
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSubmit(e as any)
    }
  }

  return (
    <div className="flex h-full">
      {mode !== 'simple' && <ChatSidebar />}
      
      <div className="flex-1 flex flex-col">
        {/* Messages */}
        <div className="flex-1 overflow-y-auto">
          <div className="max-w-4xl mx-auto py-8 px-4">
            {!currentSession || currentSession.messages.length === 0 ? (
              <div className="text-center py-20">
                <h2 className="text-2xl font-semibold text-foreground-secondary mb-4">
                  Start a conversation
                </h2>
                <p className="text-foreground-muted">
                  Ask anything, and I'll help you build amazing things.
                </p>
              </div>
            ) : (
              <div className="space-y-6">
                {currentSession?.messages.map(message => (
                  <ChatMessage key={message.id} message={message} />
                ))}
                <div ref={messagesEndRef} />
              </div>
            )}
          </div>
        </div>
        
        {/* Input */}
        <div className="border-t border-border-subtle glass">
          <form onSubmit={handleSubmit} className="max-w-4xl mx-auto p-4">
            <div className="relative">
              <textarea
                ref={inputRef}
                value={input}
                onChange={(e) => setInput(e.target.value)}
                onKeyDown={handleKeyDown}
                placeholder={currentProvider ? "Type your message..." : "Select a model to start chatting"}
                disabled={!currentProvider || isLoading}
                rows={1}
                className="w-full resize-none bg-background-tertiary rounded-lg px-4 py-3 pr-12
                         text-sm placeholder:text-foreground-muted
                         border border-border-subtle focus:border-accent-blue/50
                         focus:outline-none transition-colors
                         disabled:opacity-50 disabled:cursor-not-allowed"
                style={{
                  minHeight: '48px',
                  maxHeight: '200px'
                }}
                onInput={(e) => {
                  const target = e.target as HTMLTextAreaElement
                  target.style.height = 'auto'
                  target.style.height = `${target.scrollHeight}px`
                }}
              />
              
              <button
                type="submit"
                disabled={!input.trim() || isLoading || !currentProvider}
                className="absolute right-2 bottom-2 p-2 rounded-md
                         bg-accent-blue text-white
                         hover:bg-accent-blue/90 transition-colors
                         disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {isLoading ? (
                  <Loader2 size={18} className="animate-spin" />
                ) : (
                  <Send size={18} />
                )}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  )
}