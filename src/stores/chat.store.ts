import { create } from 'zustand'

export interface FunctionCall {
  name: string
  arguments: string
}

export interface Message {
  id: string
  role: 'user' | 'assistant' | 'system' | 'function'
  content: string
  timestamp: Date
  isStreaming?: boolean
  functionCall?: FunctionCall
}

export interface ChatSession {
  id: string
  title: string
  messages: Message[]
  createdAt: Date
  updatedAt: Date
}

interface ChatState {
  sessions: ChatSession[]
  currentSessionId: string | null
  currentConnectionId: string | null
  
  // Actions
  createSession: () => string
  deleteSession: (id: string) => void
  setCurrentSession: (id: string) => void
  addMessage: (sessionId: string, message: Omit<Message, 'id' | 'timestamp'>) => void
  updateMessage: (sessionId: string, messageId: string, content: string, isStreaming?: boolean) => void
  setCurrentConnectionId: (connectionId: string) => void
  clearSessions: () => void
}

export const useChatStore = create<ChatState>((set) => ({
  sessions: [],
  currentSessionId: null,
  currentConnectionId: null,
  
  createSession: () => {
    const id = crypto.randomUUID()
    const newSession: ChatSession = {
      id,
      title: 'New Chat',
      messages: [],
      createdAt: new Date(),
      updatedAt: new Date(),
    }
    
    set(state => ({
      sessions: [...state.sessions, newSession],
      currentSessionId: id
    }))
    
    return id
  },
  
  deleteSession: (id) => {
    set(state => ({
      sessions: state.sessions.filter(s => s.id !== id),
      currentSessionId: state.currentSessionId === id ? null : state.currentSessionId
    }))
  },
  
  setCurrentSession: (id) => {
    set({ currentSessionId: id })
  },
  
  addMessage: (sessionId, message) => {
    const newMessage: Message = {
      ...message,
      id: crypto.randomUUID(),
      timestamp: new Date()
    }
    
    set(state => ({
      sessions: state.sessions.map(session => 
        session.id === sessionId
          ? {
              ...session,
              messages: [...session.messages, newMessage],
              updatedAt: new Date(),
              title: session.messages.length === 0 && message.role === 'user' 
                ? message.content.slice(0, 50) + (message.content.length > 50 ? '...' : '')
                : session.title
            }
          : session
      )
    }))
  },
  
  updateMessage: (sessionId, messageId, content, isStreaming = false) => {
    set(state => ({
      sessions: state.sessions.map(session => 
        session.id === sessionId
          ? {
              ...session,
              messages: session.messages.map(msg => 
                msg.id === messageId
                  ? { ...msg, content, isStreaming }
                  : msg
              ),
              updatedAt: new Date()
            }
          : session
      )
    }))
  },
  
  setCurrentConnectionId: (connectionId) => {
    set({ currentConnectionId: connectionId })
  },
  
  clearSessions: () => {
    set({ sessions: [], currentSessionId: null })
  }
}))