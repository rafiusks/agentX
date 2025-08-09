import { useState, useRef, useEffect } from 'react'
import { SplitSquareHorizontal } from 'lucide-react'
import { useChatStore } from '../../stores/chat.store'
import { useStreamingStore } from '../../stores/streaming.store'
import { useCanvasStore } from '../../stores/canvas.store'
import { useChats, useChat, useChatMessages, useSendStreamingMessage, type Message } from '../../hooks/queries/useChats'
import { useDefaultConnection } from '../../hooks/queries/useConnections'
import { ChatMessage } from './ChatMessage'
import { SmartSuggestions } from './SmartSuggestions'
import { SimpleMessageInput } from './SimpleMessageInput'
import { TypingIndicator } from './TypingIndicator'
import { Canvas } from '../Canvas/Canvas'
import { Button } from '../ui/button'
import { FEATURES } from '@/config/features'

export function Chat() {
  const [input, setInput] = useState('')
  const [isCreatingSession, setIsCreatingSession] = useState(false)
  const [pendingMessage, setPendingMessage] = useState<string | null>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  
  const { 
    currentChatId,
    currentConnectionId,
    setCurrentChatId,
    setCurrentConnectionId,
    composerDraft,
    setComposerDraft,
    createSession
  } = useChatStore()
  
  const { streamingMessage, isStreaming } = useStreamingStore()
  const { isCanvasOpen, toggleCanvas } = useCanvasStore()
  
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

  const handleSubmit = async () => {
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

  const handleInputChange = (value: string) => {
    setInput(value)
    setComposerDraft(value)
  }



  // Combine regular messages with streaming message
  const allMessages = [
    ...(Array.isArray(messages) ? messages : []),
    ...(streamingMessage ? [streamingMessage] : [])
  ]

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
            <div className="chat-container chat-scroll-smooth">
              <div className="chat-messages-wrapper">
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
                      <TypingIndicator />
                    )}
                  </>
                )}
                <div ref={messagesEndRef} />
              </div>
            </div>

            {/* Simple Input */}
            <div className="border-t border-border-subtle">
              <div className="max-w-4xl mx-auto p-4">
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