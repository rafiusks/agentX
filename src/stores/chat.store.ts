import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { apiClient } from '../lib/api-client'

/**
 * Chat Store - Only handles client-side chat UI state
 * Server state (sessions, messages) is managed by TanStack Query
 */

interface ChatUIState {
  // Current active chat/connection
  currentChatId: string | null
  currentConnectionId: string | null
  
  // UI state
  isComposerFocused: boolean
  composerDraft: string
  showChatHistory: boolean
  selectedMessageId: string | null
  
  // Actions
  setCurrentChatId: (chatId: string | null) => void
  setCurrentConnectionId: (connectionId: string | null) => void
  setComposerFocused: (focused: boolean) => void
  setComposerDraft: (draft: string) => void
  toggleChatHistory: () => void
  setSelectedMessageId: (messageId: string | null) => void
  clearChatUI: () => void
  createSession: () => Promise<void>
}

export const useChatStore = create<ChatUIState>()(
  persist(
    (set) => ({
      // State
      currentChatId: null,
      currentConnectionId: null,
      isComposerFocused: false,
      composerDraft: '',
      showChatHistory: true,
      selectedMessageId: null,
      
      // Actions
      setCurrentChatId: (chatId) => set({ currentChatId: chatId }),
      
      setCurrentConnectionId: (connectionId) => set({ currentConnectionId: connectionId }),
      
      setComposerFocused: (focused) => set({ isComposerFocused: focused }),
      
      setComposerDraft: (draft) => set({ composerDraft: draft }),
      
      toggleChatHistory: () => set((state) => ({ showChatHistory: !state.showChatHistory })),
      
      setSelectedMessageId: (messageId) => set({ selectedMessageId: messageId }),
      
      clearChatUI: () => set({
        currentChatId: null,
        currentConnectionId: null,
        isComposerFocused: false,
        composerDraft: '',
        selectedMessageId: null,
      }),
      
      createSession: async () => {
        try {
          // Create a new session through the API
          const response = await apiClient.post<{ id: string; ID?: string }>('/sessions', {
            title: 'New Chat'
          })
          
          // Set the new session as current
          if (response && (response as any).ID) {
            set({ currentChatId: (response as any).ID })
          } else if (response && response.id) {
            set({ currentChatId: response.id })
          }
        } catch (error) {
          console.error('Failed to create session:', error)
        }
      },
    }),
    {
      name: 'agentx-chat-ui',
      partialize: (state) => ({
        // Only persist drafts and preferences
        composerDraft: state.composerDraft,
        showChatHistory: state.showChatHistory,
      }),
    }
  )
)