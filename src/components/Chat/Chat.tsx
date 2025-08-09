import { useState, useRef, useEffect, useCallback, useMemo, memo } from 'react'
import { SplitSquareHorizontal } from 'lucide-react'
import { useChatStore } from '../../stores/chat.store'
import { useStreamingStore } from '../../stores/streaming.store'
import { useCanvasStore } from '../../stores/canvas.store'
import { useChats, useChat, useChatMessages, useSendStreamingMessage, type Message } from '../../hooks/queries/useChats'
import { useDefaultConnection } from '../../hooks/queries/useConnections'
import { useSessionSummaries, useGenerateSummary } from '../../hooks/queries/useSummaries'
import { ChatMessage } from './ChatMessage'
import { SmartSuggestions } from './SmartSuggestions'
import { SimpleMessageInput } from './SimpleMessageInput'
import { TypingIndicator } from './TypingIndicator'
import { ErrorMessage } from './ErrorMessage'
import { ContextIndicator } from './ContextIndicator'
import { ContextSettings } from './ContextSettings'
import { SummaryActions } from './SummaryActions'
import { ResponsePreferences } from './ResponsePreferences'
import { Canvas } from '../Canvas/Canvas'
import { Button } from '../ui/button'
import { FEATURES } from '@/config/features'
import { useContextStore } from '../../stores/context.store'
import { useKeyboardShortcuts } from '../../hooks/useKeyboardShortcuts'

// Memoized message list component to prevent re-renders
const MessagesList = memo(function MessagesList({ 
  messages, 
  isStreaming, 
  streamingMessage 
}: { 
  messages: Message[], 
  isStreaming: boolean, 
  streamingMessage: any 
}) {
  return (
    <>
      {messages.map((message, index) => (
        <ChatMessage key={message.id || `msg-${index}`} message={message as Message} />
      ))}
      {isStreaming && !streamingMessage && (
        <TypingIndicator />
      )}
    </>
  )
})

export function Chat() {
  const [input, setInput] = useState('')
  const [isCreatingSession, setIsCreatingSession] = useState(false)
  const [pendingMessage, setPendingMessage] = useState<string | null>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const chatContainerRef = useRef<HTMLDivElement>(null)
  const [shouldAutoScroll, setShouldAutoScroll] = useState(true)
  
  // Enable keyboard shortcuts
  useKeyboardShortcuts()
  
  const { 
    currentChatId,
    currentConnectionId,
    setCurrentChatId,
    setCurrentConnectionId,
    composerDraft,
    setComposerDraft,
    createSession
  } = useChatStore()
  
  const { streamingMessage, isStreaming, streamError, contextUsage } = useStreamingStore()
  const { isCanvasOpen, toggleCanvas } = useCanvasStore()
  const { config: contextConfig, setStrategy } = useContextStore()
  
  // Query hooks
  const { data: chats = [] } = useChats()
  const { data: _currentChat } = useChat(currentChatId || undefined)
  const { data: messages = [] } = useChatMessages(currentChatId || undefined)
  const { data: defaultConnection } = useDefaultConnection()
  const sendMessageMutation = useSendStreamingMessage()
  const { data: summaries = [] } = useSessionSummaries(currentChatId || undefined)
  const generateSummaryMutation = useGenerateSummary()

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

  // Check if user is near bottom of chat
  const checkIfNearBottom = () => {
    if (!chatContainerRef.current) return true
    
    const container = chatContainerRef.current
    const threshold = 100 // pixels from bottom
    const isNearBottom = 
      container.scrollHeight - container.scrollTop - container.clientHeight < threshold
    
    return isNearBottom
  }

  // Handle scroll to track if user scrolls up
  const handleScroll = () => {
    setShouldAutoScroll(checkIfNearBottom())
  }

  useEffect(() => {
    // Only auto-scroll if user is near bottom or we just sent a message
    if (shouldAutoScroll && messagesEndRef.current) {
      messagesEndRef.current.scrollIntoView({ behavior: 'smooth' })
    }
  }, [messages, streamingMessage, shouldAutoScroll])

  // Always scroll to bottom when user sends a new message
  useEffect(() => {
    if (messages.length > 0) {
      const lastMessage = messages[messages.length - 1]
      if (lastMessage.role === 'user') {
        setShouldAutoScroll(true)
        messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
      }
    }
  }, [messages])

  // Only sync draft with input on mount or when draft changes externally
  useEffect(() => {
    // Only update if they're different to avoid loops
    if (composerDraft !== input) {
      setInput(composerDraft)
    }
  }, [composerDraft]) // Intentionally not including input to avoid circular updates
  
  // Cleanup debounce timer on unmount
  useEffect(() => {
    return () => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current)
      }
    }
  }, [])

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
  
  // Auto-generate summary when we have too many messages without a recent summary
  useEffect(() => {
    if (!currentChatId || messages.length < 30) return
    
    // Check if we need to generate a summary
    // If no summaries exist, or the latest summary is more than 20 messages old
    const shouldGenerateSummary = () => {
      if (summaries.length === 0) return true
      
      const latestSummary = summaries[0] // Summaries are sorted by created_at DESC
      const summaryTime = new Date(latestSummary.created_at)
      const messagesSinceSummary = messages.filter(m => 
        new Date(m.created_at) > summaryTime
      )
      
      return messagesSinceSummary.length >= 20
    }
    
    if (shouldGenerateSummary() && !generateSummaryMutation.isPending) {
      console.log('[Chat] Auto-generating summary for session with', messages.length, 'messages')
      generateSummaryMutation.mutate({
        sessionId: currentChatId,
        messageCount: 20
      })
    }
  }, [currentChatId, messages, summaries, generateSummaryMutation])

  const handleSubmit = async () => {
    if (!input.trim() || isStreaming || !currentConnectionId || isCreatingSession) return
    
    const messageContent = input.trim()
    
    // Enable auto-scroll when sending a message
    setShouldAutoScroll(true)
    
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

  // Debounce timer ref
  const debounceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  
  const handleInputChange = useCallback((value: string) => {
    // Update local state immediately for responsive typing
    setInput(value)
    
    // Debounce the Zustand store update to reduce re-renders
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current)
    }
    
    debounceTimerRef.current = setTimeout(() => {
      setComposerDraft(value)
    }, 300) // 300ms debounce
  }, [setComposerDraft])



  // Combine regular messages with streaming message - memoized to prevent recalculation
  const allMessages = useMemo(() => [
    ...(Array.isArray(messages) ? messages : []),
    ...(streamingMessage ? [streamingMessage] : [])
  ], [messages, streamingMessage])

  return (
    <div className="flex w-full h-full overflow-hidden">
      
      <div className={`flex-1 flex flex-col h-full ${isCanvasOpen ? 'mr-[50%]' : ''}`}>
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
            <div 
              ref={chatContainerRef}
              onScroll={handleScroll}
              className="chat-container chat-scroll-smooth">
              <div className="chat-messages-wrapper">
                {allMessages.length === 0 ? (
                  <div className="flex items-center justify-center h-full">
                    <div className="text-center space-y-2">
                      <p className="text-foreground-secondary">No messages yet. Start the conversation!</p>
                    </div>
                  </div>
                ) : (
                  <MessagesList 
                    messages={allMessages as Message[]} 
                    isStreaming={isStreaming} 
                    streamingMessage={streamingMessage} 
                  />
                )}
                <div ref={messagesEndRef} />
              </div>
            </div>

            {/* Context Indicator and Settings */}
            {contextUsage && (
              <div className="border-t border-border-subtle">
                <div className="max-w-6xl mx-auto px-4 py-2 flex items-center justify-between">
                  <ContextIndicator {...contextUsage} />
                  <div className="flex items-center gap-2">
                    <ResponsePreferences />
                    <SummaryActions 
                      sessionId={currentChatId}
                      messageCount={messages.length}
                    />
                    <ContextSettings 
                      currentStrategy={contextConfig.strategy}
                      onStrategyChange={setStrategy}
                    />
                  </div>
                </div>
              </div>
            )}
            
            {/* Simple Input */}
            <div className="border-t border-border-subtle">
              <div className="max-w-6xl mx-auto p-4">
                <SimpleMessageInput
                  value={input}
                  onChange={handleInputChange}
                  onSubmit={handleSubmit}
                  isLoading={isStreaming || isCreatingSession}
                  disabled={isStreaming || isCreatingSession}
                  placeholder={isStreaming ? "AI is responding..." : isCreatingSession ? "Creating session..." : "Ask me anything..."}
                />
              </div>
            </div>
        
        {/* Smart Suggestions */}
        {input.length === 0 && (
          <SmartSuggestions 
            onSuggestionClick={(suggestion) => {
              setInput(suggestion);
            }}
          />
        )}
          </>
        )}
      </div>
      
      {/* Canvas */}
      {FEATURES.CANVAS_MODE && <Canvas />}
    </div>
  )
}