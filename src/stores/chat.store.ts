import { create } from 'zustand'
import { persist } from 'zustand/middleware'

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