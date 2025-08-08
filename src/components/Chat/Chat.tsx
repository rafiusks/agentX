import { useState, useRef, useEffect } from 'react'
import { Send, Loader2 } from 'lucide-react'
import { useChatStore } from '../../stores/chat.store'
import { useUIStore } from '../../stores/ui.store'
import { useStreamingStore } from '../../stores/streaming.store'
import { useChats, useChat, useChatMessages, useSendStreamingMessage, type Message } from '../../hooks/queries/useChats'
import { useDefaultConnection } from '../../hooks/queries/useConnections'
import { ChatMessage } from './ChatMessage'
import { ChatSidebar } from './ChatSidebar'

export function Chat() {
  const [input, setInput] = useState('')
  const [isCreatingSession, setIsCreatingSession] = useState(false)
  const [pendingMessage, setPendingMessage] = useState<string | null>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLTextAreaElement>(null)
  
  const { 
    currentChatId,
    currentConnectionId,
    setCurrentChatId,
    setCurrentConnectionId,
    composerDraft,
    setComposerDraft,
    createSession
  } = useChatStore()
  
  const { mode } = useUIStore()
  const { streamingMessage, isStreaming } = useStreamingStore()
  
  // Query hooks
  const { data: chats = [] } = useChats()
  const { data: _currentChat } = useChat(currentChatId || undefined)
  const { data: messages = [] } = useChatMessages(currentChatId || undefined)
  const { data: defaultConnection } = useDefaultConnection()
  const sendMessageMutation = useSendStreamingMessage()

  useEffect(() => {
    // Select first chat if none selected but chats exist
    if (chats.length > 0 && !currentChatId) {
      const firstChatId = chats[0].ID || chats[0].id;
      if (firstChatId) {
        setCurrentChatId(firstChatId)
      }
    }
  }, [chats, currentChatId, setCurrentChatId])
  
  // Set connection from default if not set
  useEffect(() => {
    if (defaultConnection && !currentConnectionId) {
      setCurrentConnectionId(defaultConnection.id)
    }
  }, [defaultConnection, currentConnectionId])

  useEffect(() => {
    // Scroll to bottom on new messages
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, streamingMessage])

  // Sync draft with input
  useEffect(() => {
    setInput(composerDraft)
  }, [composerDraft])

  // Submit pending message after session creation
  useEffect(() => {
    if (currentChatId && pendingMessage && !isCreatingSession && currentConnectionId) {
      const messageToSend = pendingMessage
      setPendingMessage(null)
      
      // Send the pending message
      sendMessageMutation.mutate({
        chat_id: currentChatId,
        content: messageToSend,
        connection_id: currentConnectionId,
        stream: true
      }, {
        onError: (error) => {
          console.error('Failed to send message:', error)
          // Restore input on error
          setInput(messageToSend)
          setComposerDraft(messageToSend)
        }
      })
    }
  }, [currentChatId, pendingMessage, isCreatingSession, currentConnectionId, sendMessageMutation, setComposerDraft])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    
    if (!input.trim() || isStreaming || !currentConnectionId || isCreatingSession) return
    
    const messageContent = input.trim()
    
    // Create a session if we don't have one
    if (!currentChatId) {
      setPendingMessage(messageContent)
      setIsCreatingSession(true)
      try {
        await createSession()
        // The session creation will update currentChatId via the store
        // The useEffect will pick up the pending message and send it
      } catch (error) {
        console.error('Failed to create session:', error)
        setPendingMessage(null)
        // Restore input on error
        setInput(messageContent)
        setComposerDraft(messageContent)
      } finally {
        setIsCreatingSession(false)
      }
      return
    }
    
    setInput('')
    setComposerDraft('')
    
    // Send message using mutation
    sendMessageMutation.mutate({
      chat_id: currentChatId,
      content: messageContent,
      connection_id: currentConnectionId,
      stream: true
    }, {
      onError: (error) => {
        console.error('Failed to send message:', error)
        // Restore input on error
        setInput(messageContent)
        setComposerDraft(messageContent)
      }
    })
  }

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSubmit(e)
    }
  }

  const handleInputChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setInput(e.target.value)
    setComposerDraft(e.target.value)
  }

  const showSidebar = mode !== 'simple'

  // Combine regular messages with streaming message
  const allMessages = [
    ...(Array.isArray(messages) ? messages : []),
    ...(streamingMessage ? [streamingMessage] : [])
  ]

  return (
    <div className="flex w-full h-full overflow-hidden">
      {showSidebar && <ChatSidebar />}
      
      <div className="flex-1 flex flex-col h-full">
        {/* Show empty state if no chat is selected */}
        {!currentChatId ? (
          <div className="flex-1 flex items-center justify-center">
            <div className="text-center space-y-4 max-w-md">
              <div className="text-6xl">💬</div>
              <h2 className="text-2xl font-semibold text-foreground-primary">
                Start a New Conversation
              </h2>
              <p className="text-foreground-secondary">
                Click "New Chat" to begin or select an existing conversation from the sidebar
              </p>
            </div>
          </div>
        ) : (
          <>
            {/* Messages */}
            <div className="flex-1 overflow-y-auto px-4 py-6">
              <div className="max-w-4xl mx-auto space-y-6">
                {allMessages.length === 0 ? (
                  <div className="flex items-center justify-center h-full">
                    <div className="text-center space-y-2">
                      <p className="text-foreground-secondary">No messages yet. Start the conversation!</p>
                    </div>
                  </div>
                ) : (
                  <>
                    {allMessages.map((message, index) => (
                      <ChatMessage key={message.id || `msg-${index}`} message={message as Message} />
                    ))}
                    {isStreaming && !streamingMessage && (
                      <div className="flex items-center gap-2 text-foreground-secondary">
                        <Loader2 className="w-4 h-4 animate-spin" />
                        <span>Thinking...</span>
                      </div>
                    )}
                  </>
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
                onChange={handleInputChange}
                onKeyDown={handleKeyDown}
                placeholder={isStreaming ? "Waiting for response..." : isCreatingSession ? "Creating session..." : "Type a message..."}
                disabled={isStreaming || isCreatingSession}
                className="w-full px-4 py-3 pr-12 bg-background-secondary border border-border-subtle 
                         rounded-lg resize-none focus:outline-none focus:ring-2 focus:ring-accent-blue/50 
                         focus:ring-offset-2 focus:ring-offset-background-primary
                         text-foreground-primary placeholder-foreground-tertiary
                         disabled:opacity-50 disabled:cursor-not-allowed transition-all"
                rows={3}
              />
              <button
                type="submit"
                disabled={!input.trim() || isStreaming || isCreatingSession}
                className="absolute bottom-3 right-3 p-2 rounded-lg
                         bg-accent-blue text-white disabled:opacity-50 
                         disabled:cursor-not-allowed hover:bg-accent-blue/90
                         transition-all active:scale-95"
              >
                {isStreaming || isCreatingSession ? (
                  <Loader2 size={18} className="animate-spin" />
                ) : (
                  <Send size={18} />
                )}
              </button>
            </div>
          </div>
        </form>
          </>
        )}
      </div>
    </div>
  )
}