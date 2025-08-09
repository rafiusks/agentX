import { useState, useRef, useEffect, useCallback, useMemo, memo } from 'react'
import { SplitSquareHorizontal } from 'lucide-react'
import { useChatStore } from '../../stores/chat.store'
import { useStreamingMessage, useIsStreaming, useStreamingStore } from '../../stores/streaming.store'
import { useCanvasStore } from '../../stores/canvas.store'
import { useChats, useChat, useChatMessages, useSendStreamingMessage, type Message } from '../../hooks/queries/useChats'
import { useDefaultConnection } from '../../hooks/queries/useConnections'
import { useSessionSummaries, useGenerateSummary } from '../../hooks/queries/useSummaries'
import { ChatMessage } from './ChatMessage'
import { SmartSuggestions } from './SmartSuggestions'
import { SimpleMessageInput } from './SimpleMessageInput'
import { ComposerWithFeatures } from './ComposerWithFeatures'
import { TypingIndicator } from './TypingIndicator'
import { ErrorMessage } from './ErrorMessage'
import { ContextIndicator } from './ContextIndicator'
import { ContextSettings } from './ContextSettings'
import { AccessibilityAnnouncer } from './AccessibilityAnnouncer'
import { SkipLinks } from '../SkipLinks'
import { SummaryActions } from './SummaryActions'
import { ResponsePreferences } from './ResponsePreferences'
import { Canvas } from '../Canvas/Canvas'
import { Button } from '../ui/button'
import { FEATURES } from '@/config/features'
import { useContextStore } from '../../stores/context.store'
import { useKeyboardShortcuts } from '../../hooks/useKeyboardShortcuts'
import { useEnhancedKeyboardShortcuts } from '../../hooks/useEnhancedKeyboardShortcuts'
import { useDebouncedStream } from '../../hooks/useDebouncedStream'
import { useInputDebug } from '../../hooks/useInputDebug'

// Memoized message list component to prevent re-renders
const MessagesList = memo(function MessagesList({ 
  messages, 
  isStreaming, 
  streamingMessage,
  onRegenerate 
}: { 
  messages: Message[], 
  isStreaming: boolean, 
  streamingMessage: any,
  onRegenerate?: (messageId: string) => void 
}) {
  return (
    <>
      {messages.map((message, index) => (
        <ChatMessage 
          key={message.id || `msg-${index}`} 
          message={message as Message} 
          onRegenerate={onRegenerate}
        />
      ))}
      {isStreaming && !streamingMessage && (
        <TypingIndicator />
      )}
    </>
  )
}, (prevProps, nextProps) => {
  // Custom comparison to prevent unnecessary re-renders
  // Only re-render if messages length changes or streaming state changes
  return (
    prevProps.messages.length === nextProps.messages.length &&
    prevProps.isStreaming === nextProps.isStreaming &&
    prevProps.streamingMessage?.content === nextProps.streamingMessage?.content &&
    prevProps.onRegenerate === nextProps.onRegenerate
  )
})

export function Chat() {
  const [isCreatingSession, setIsCreatingSession] = useState(false)
  const [pendingMessage, setPendingMessage] = useState<string | null>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const chatContainerRef = useRef<HTMLDivElement>(null)
  const [shouldAutoScroll, setShouldAutoScroll] = useState(true)
  
  // Enable keyboard shortcuts
  useKeyboardShortcuts()
  useEnhancedKeyboardShortcuts()
  
  const { 
    currentChatId,
    currentConnectionId,
    setCurrentChatId,
    setCurrentConnectionId,
    composerDraft,
    setComposerDraft,
    createSession
  } = useChatStore()
  
  const rawStreamingMessage = useStreamingMessage()
  const streamingMessage = useDebouncedStream(rawStreamingMessage, 30) // 30ms debounce for smooth streaming
  const isStreaming = useIsStreaming()
  const { streamError, contextUsage } = useStreamingStore(
    (state) => ({ streamError: state.streamError, contextUsage: state.contextUsage })
  )
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

  // Check if user is near bottom of chat - memoized
  const checkIfNearBottom = useCallback(() => {
    if (!chatContainerRef.current) return true
    
    const container = chatContainerRef.current
    const threshold = 100 // pixels from bottom
    const isNearBottom = 
      container.scrollHeight - container.scrollTop - container.clientHeight < threshold
    
    return isNearBottom
  }, [])

  // Handle scroll to track if user scrolls up - throttled
  const handleScroll = useMemo(() => {
    let ticking = false;
    
    return () => {
      if (!ticking) {
        requestAnimationFrame(() => {
          setShouldAutoScroll(checkIfNearBottom());
          ticking = false;
        });
        ticking = true;
      }
    };
  }, [checkIfNearBottom])

  useEffect(() => {
    // Only auto-scroll if user is near bottom or we just sent a message
    if (shouldAutoScroll && messagesEndRef.current) {
      // Use RAF to avoid forced reflow
      requestAnimationFrame(() => {
        messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
      })
    }
  }, [messages, streamingMessage, shouldAutoScroll])

  // Always scroll to bottom when user sends a new message
  useEffect(() => {
    if (messages.length > 0) {
      const lastMessage = messages[messages.length - 1]
      if (lastMessage.role === 'user') {
        setShouldAutoScroll(true)
        // Use RAF to avoid forced reflow
        requestAnimationFrame(() => {
          messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
        })
      }
    }
  }, [messages])

  

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
  }, [currentChatId, messages.length, summaries.length]) // Only depend on lengths, not arrays themselves

  const handleRegenerate = useCallback(async (messageId: string) => {
    if (!currentChatId || !currentConnectionId || isStreaming) return
    
    // Find the last user message before this assistant message
    const messageIndex = messages.findIndex(m => m.id === messageId)
    if (messageIndex <= 0) return // No previous message or not found
    
    // Get the previous user message
    const previousMessage = messages[messageIndex - 1]
    if (!previousMessage || previousMessage.role !== 'user') return
    
    console.log('[Chat] Regenerating response for message:', messageId)
    
    // Re-send the previous user message to get a new response
    sendMessageMutation.mutate({
      chat_id: currentChatId,
      content: previousMessage.content,
      connection_id: currentConnectionId,
      stream: true,
      regenerate_from: messageId // Signal backend this is a regeneration
    }, {
      onError: (error) => {
        console.error('Failed to regenerate message:', error)
      }
    })
  }, [currentChatId, currentConnectionId, isStreaming, messages, sendMessageMutation])

  const handleSubmit = useCallback(async (messageContent: string) => {
    if (!messageContent || isStreaming || !currentConnectionId || isCreatingSession) return
    
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
        // Cannot restore input as it's managed by IsolatedComposer
      } finally {
        setIsCreatingSession(false)
      }
      return
    }
    
    // Input is managed by IsolatedComposer, no need to clear here
    setComposerDraft('') // Clear draft on submit
    
    // Send message using mutation
    sendMessageMutation.mutate({
      chat_id: currentChatId,
      content: messageContent,
      connection_id: currentConnectionId,
      stream: true
    }, {
      onError: (error) => {
        console.error('Failed to send message:', error)
        // Cannot restore input as it's managed by IsolatedComposer
      }
    })
  }, [isStreaming, currentConnectionId, isCreatingSession, currentChatId, createSession, sendMessageMutation])




  // Combine regular messages with streaming message - memoized to prevent recalculation
  const allMessages = useMemo(() => [
    ...(Array.isArray(messages) ? messages : []),
    ...(streamingMessage ? [streamingMessage] : [])
  ], [messages, streamingMessage])

  return (
    <div className="flex w-full h-full overflow-hidden">
      <SkipLinks />
      <AccessibilityAnnouncer />
      
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
              className="chat-container chat-scroll-smooth momentum-scroll"
              id="chat-messages"
              role="log"
              aria-label="Chat messages">
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
                    onRegenerate={handleRegenerate} 
                  />
                )}
                <div ref={messagesEndRef} />
              </div>
            </div>

            {/* Context Indicator and Settings */}
            {contextUsage && (
              <div className="border-t border-border-subtle">
                <div className="max-w-4xl mx-auto px-4 py-2 flex items-center justify-between">
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
            
            {/* Unified Message Composer - Split architecture for performance */}
            <div className="border-t border-border-subtle">
              <div className="max-w-4xl mx-auto p-4">
                <ComposerWithFeatures
                  onSubmit={handleSubmit}
                  isLoading={isStreaming || isCreatingSession}
                  disabled={isStreaming || isCreatingSession}
                  placeholder={isStreaming ? "AI is responding..." : isCreatingSession ? "Creating session..." : "Ask me anything..."}
                  connectionId={currentConnectionId}
                  maxTokens={4096}
                />
              </div>
            </div>
        
        {/* Smart Suggestions - Removed as input is now isolated */}
          </>
        )}
      </div>
      
      {/* Canvas */}
      {FEATURES.CANVAS_MODE && <Canvas />}
    </div>
  )
}